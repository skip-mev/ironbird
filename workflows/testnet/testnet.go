package testnet

import (
	"fmt"
	"time"

	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/observability"
	"github.com/skip-mev/ironbird/activities/testnet"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var testnetActivities *testnet.Activity
var githubActivities *github.NotifierActivity
var observabilityActivities *observability.Activity
var loadTestActivities *loadtest.Activity

func Workflow(ctx workflow.Context, opts WorkflowOptions) (string, error) {
	if err := opts.Validate(); err != nil {
		return "", temporal.NewApplicationErrorWithOptions(
			"invalid workflow options",
			err.Error(),
			temporal.ApplicationErrorOptions{NonRetryable: true},
		)
	}

	name := fmt.Sprintf("Testnet (%s) bake", opts.ChainConfig.Name)

	if opts.LoadTestSpec != nil {
		name = fmt.Sprintf("%s/loadtest-%s", opts.ChainConfig.Name, opts.LoadTestSpec.Name)
	}

	checkName := fmt.Sprintf("Testnet (%s) bake", name)
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runName := fmt.Sprintf("ib-%s-%s", opts.ChainConfig.Name, runID[:6])
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
		&opts,
	)

	if err != nil {
		return "", err
	}

	buildResult, err := buildImage(ctx, opts)

	if err != nil {
		return "", err
	}

	if err := report.SetBuildResult(ctx, buildResult); err != nil {
		return "", err
	}

	testnetOptions := testnet.TestnetOptions{
		Name:                 runName,
		Image:                buildResult.FQDNTag,
		UID:                  opts.ChainConfig.Image.UID,
		GID:                  opts.ChainConfig.Image.GID,
		BinaryName:           opts.ChainConfig.Image.BinaryName,
		HomeDir:              opts.ChainConfig.Image.HomeDir,
		GenesisModifications: opts.ChainConfig.GenesisModifications,
		RunnerType:           string(opts.RunnerType),
		NumOfValidators:      opts.ChainConfig.NumOfValidators,
		NumOfNodes:           opts.ChainConfig.NumOfNodes,
	}

	if opts.RunnerType == testnettypes.DigitalOcean {
		testnetOptions.ProviderSpecificOptions = map[string]string{
			"region":   "ams3",
			"image_id": "177869680",
			"size":     "s-4vcpu-8gb",
		}
	}

	var providerState string
	if err = workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, testnetOptions).Get(ctx, &providerState); err != nil {
		return "", err
	}

	testnetOptions.ProviderState = []byte(providerState)

	defer func() {
		newCtx, _ := workflow.NewDisconnectedContext(ctx)
		err := workflow.ExecuteActivity(newCtx, testnetActivities.TeardownProvider, testnetOptions).Get(newCtx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("failed to teardown provider", "error", err)
		}
	}()

	var chainState testnet.PackagedState

	if err = workflow.ExecuteActivity(ctx, testnetActivities.LaunchTestnet, testnetOptions).Get(ctx, &chainState); err != nil {
		return "", err
	}

	testnetOptions.ChainState = chainState.ChainState
	testnetOptions.ProviderState = chainState.ProviderState

	if err := report.SetNodes(ctx, chainState.Nodes); err != nil {
		return "", err
	}

	var observabilityPackagedState observability.PackagedState
	var metricsIps []string

	for _, node := range chainState.Nodes {
		metricsIps = append(metricsIps, fmt.Sprintf("%s:26660", node.Metrics))
		metricsIps = append(metricsIps, fmt.Sprintf("%s:1317", node.Metrics))
	}

	workflow.GetLogger(ctx).Info("metrics ips", zap.Strings("ips", metricsIps))

	if err := workflow.ExecuteActivity(
		ctx,
		observabilityActivities.LaunchObservabilityStack,
		observability.Options{
			PrometheusTargets:      metricsIps,
			ProviderState:          testnetOptions.ProviderState,
			ProviderSpecificConfig: testnetOptions.ProviderSpecificOptions,
			RunnerType:             string(opts.RunnerType),
		},
	).Get(ctx, &observabilityPackagedState); err != nil {
		return "", err
	}

	testnetOptions.ProviderState = observabilityPackagedState.ProviderState

	if err := report.SetObservabilityURL(ctx, observabilityPackagedState.ExternalGrafanaURL); err != nil {
		return "", err
	}

	var loadTestRuntime time.Duration
	if opts.LoadTestSpec != nil {
		workflow.Go(ctx, func(ctx workflow.Context) {
			updateErr := report.UpdateLoadTest(ctx, "Load test in progress", "", nil)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
			}

			// assume ~ 2 sec block times
			loadTestRuntime = time.Duration(opts.LoadTestSpec.NumOfBlocks*2) * time.Second
			// buffer for load test run & wallets creating
			loadTestRuntime += 30 * time.Minute

			var state loadtest.PackagedState
			if err := workflow.ExecuteActivity(
				workflow.WithStartToCloseTimeout(ctx, loadTestRuntime),
				loadTestActivities.RunLoadTest,
				testnetOptions.ChainState,
				opts.LoadTestSpec,
				opts.RunnerType,
				testnetOptions.ProviderState,
			).Get(ctx, &state); err != nil {
				workflow.GetLogger(ctx).Error("Load test failed with error", zap.Error(err))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+err.Error(), "", nil)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
				return
			}

			if state.Result.Error != "" {
				workflow.GetLogger(ctx).Error("Load test reported an error", zap.String("error", state.Result.Error))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+state.Result.Error, "", &state.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			} else {
				updateErr := report.UpdateLoadTest(ctx, "✅ Load test completed successfully!", "", &state.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			}
		})
	}

	if err := monitorTestnet(ctx, testnetOptions, report, loadTestRuntime, observabilityPackagedState.ExternalGrafanaURL); err != nil {
		return "", err
	}

	if err := report.Conclude(ctx, "completed", "success", "Testnet bake completed"); err != nil {
		return "", err
	}

	return "", nil
}

func monitorTestnet(ctx workflow.Context, testnetOptions testnet.TestnetOptions, report *Report, loadTestRuntime time.Duration, grafanaUrl string) error {
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

		var status string
		// TODO: metrics checks
		err := workflow.ExecuteActivity(ctx, testnetActivities.MonitorTestnet, testnetOptions).Get(ctx, &status)

		if err != nil {
			return err
		}

		if err := report.SetStatus(ctx, "in_progress", "Monitoring the testnet", fmt.Sprintf("Monitoring the testnet - %d", i)); err != nil {
			return err
		}
	}

	return nil
}
