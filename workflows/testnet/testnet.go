package testnet

import (
	"fmt"
	"github.com/skip-mev/ironbird/messages"
	"time"

	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/testnet"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var testnetActivities *testnet.Activity
var githubActivities *github.NotifierActivity
var loadTestActivities *loadtest.Activity

func Workflow(ctx workflow.Context, req messages.TestnetWorkflowRequest) (messages.TestnetWorkflowResponse, error) {
	if err := req.Validate(); err != nil {
		return "", temporal.NewApplicationErrorWithOptions(
			"invalid workflow options",
			err.Error(),
			temporal.ApplicationErrorOptions{NonRetryable: true},
		)
	}

	name := fmt.Sprintf("Testnet (%s) bake", req.ChainConfig.Name)

	if req.LoadTestSpec != nil {
		name = fmt.Sprintf("%s/loadtest-%s", req.ChainConfig.Name, req.LoadTestSpec.Name)
	}

	checkName := fmt.Sprintf("Testnet (%s) bake", name)
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runName := fmt.Sprintf("ib-%s-%s", req.ChainConfig.Name, runID[:6])

	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	report, err := NewReport(
		ctx,
		checkName,
		"Launching testnet",
		"",
		req,
	)

	if err != nil {
		return "", err
	}

	buildResult, err := buildImage(ctx, req)

	if err != nil {
		return "", err
	}

	if err := report.SetBuildResult(ctx, buildResult); err != nil {
		return "", err
	}

	var providerState, chainState []byte
	var providerSpecificOptions map[string]string

	if req.RunnerType == testnettypes.DigitalOcean {
		providerSpecificOptions = map[string]string{
			"region":   "ams3",
			"image_id": "185210261",
			"size":     "s-4vcpu-8gb",
		}
	}

	var createProviderResp messages.CreateProviderResponse
	if err = workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, messages.CreateProviderRequest{
		RunnerType: req.RunnerType,
		Name:       runName,
	}).Get(ctx, &createProviderResp); err != nil {
		return "", err
	}

	providerState = createProviderResp.ProviderState

	defer func() {
		newCtx, _ := workflow.NewDisconnectedContext(ctx)
		err := workflow.ExecuteActivity(newCtx, testnetActivities.TeardownProvider, messages.TeardownProviderRequest{
			RunnerType:    req.RunnerType,
			ProviderState: providerState,
		}).Get(newCtx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("failed to teardown provider", "error", err)
		}
	}()

	var testnetResp messages.LaunchTestnetResponse

	if err = workflow.ExecuteActivity(ctx, testnetActivities.LaunchTestnet, messages.LaunchTestnetRequest{
		Name:                    runName,
		Image:                   buildResult.FQDNTag,
		UID:                     req.ChainConfig.Image.UID,
		GID:                     req.ChainConfig.Image.GID,
		BinaryName:              req.ChainConfig.Image.BinaryName,
		HomeDir:                 req.ChainConfig.Image.HomeDir,
		GenesisModifications:    req.ChainConfig.GenesisModifications,
		RunnerType:              req.RunnerType,
		NumOfValidators:         req.ChainConfig.NumOfValidators,
		NumOfNodes:              req.ChainConfig.NumOfNodes,
		ProviderSpecificOptions: providerSpecificOptions,
		ProviderState:           providerState,
	}).Get(ctx, &testnetResp); err != nil {
		return "", err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	if err := report.SetNodes(ctx, testnetResp.Nodes); err != nil {
		return "", err
	}

	if err := report.SetDashboards(ctx, req.GrafanaConfig, testnetResp.ChainID); err != nil {
		return "", err
	}

	var loadTestRuntime time.Duration
	if req.LoadTestSpec != nil {
		workflow.Go(ctx, func(ctx workflow.Context) {
			updateErr := report.UpdateLoadTest(ctx, "Load test in progress", "", nil)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
			}

			// assume ~ 2 sec block times
			loadTestRuntime = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second
			// buffer for load test run & wallets creating
			loadTestRuntime += 30 * time.Minute

			var loadTestResp messages.RunLoadTestResponse
			if err := workflow.ExecuteActivity(
				workflow.WithStartToCloseTimeout(ctx, loadTestRuntime),
				loadTestActivities.RunLoadTest,
				messages.RunLoadTestRequest{
					ChainState:    chainState,
					ProviderState: providerState,
					LoadTestSpec:  *req.LoadTestSpec,
					RunnerType:    req.RunnerType,
				},
			).Get(ctx, &loadTestResp); err != nil {
				workflow.GetLogger(ctx).Error("Load test failed with error", zap.Error(err))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+err.Error(), "", nil)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
				return
			}

			if loadTestResp.Result.Error != "" {
				workflow.GetLogger(ctx).Error("Load test reported an error", zap.String("error", loadTestResp.Result.Error))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+loadTestResp.Result.Error, "", &loadTestResp.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			} else {
				updateErr := report.UpdateLoadTest(ctx, "✅ Load test completed successfully!", "", &loadTestResp.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			}
		})
	}

	if err := monitorTestnet(ctx, chainState, providerState, req.RunnerType, report, loadTestRuntime); err != nil {
		return "", err
	}

	if err := report.Conclude(ctx, "completed", "success", "Testnet bake completed"); err != nil {
		return "", err
	}

	return "", nil
}

func monitorTestnet(ctx workflow.Context, chainState, providerState []byte, runnerType testnettypes.RunnerType, report *Report, loadTestRuntime time.Duration) error {
	// Calculate number of iterations (each iteration is 10 seconds)
	iterations := 360 // default to 1 hour (360 * 10 seconds)

	// Check if loadTestRuntime is longer than the default 1 hour
	defaultDuration := time.Second * 10 * time.Duration(iterations)
	if loadTestRuntime > defaultDuration {
		// Calculate iterations based on loadTestRuntime (each iteration is 10 seconds)
		iterations = int(loadTestRuntime.Seconds() / 10)
	}

	for i := 0; i < iterations; i++ {
		if err := workflow.Sleep(ctx, 10*time.Second); err != nil {
			return err
		}

		var monitorTestnetResp messages.MonitorTestnetResponse
		// TODO: metrics checks
		err := workflow.ExecuteActivity(ctx, testnetActivities.MonitorTestnet, messages.MonitorTestnetRequest{
			RunnerType:    runnerType,
			ChainState:    chainState,
			ProviderState: providerState,
		}).Get(ctx, &monitorTestnetResp)

		if err != nil {
			return err
		}

		if err := report.SetStatus(ctx, "in_progress", "Monitoring the testnet", fmt.Sprintf("Monitoring the testnet - %d", i)); err != nil {
			return err
		}
	}

	return nil
}
