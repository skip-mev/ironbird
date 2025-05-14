package testnet

import (
	"context"
	"fmt"
	"time"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"

	"github.com/skip-mev/ironbird/messages"

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

const (
	defaultRuntime = time.Hour
	updateHandler  = "chain_update"
	shutdownSignal = "shutdown"
)

var (
	defaultWorkflowOptions = workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
)

func teardownProvider(ctx workflow.Context, runnerType testnettypes.RunnerType, providerState []byte) error {
	workflow.GetLogger(ctx).Info("tearing down provider")
	err := workflow.ExecuteActivity(ctx, testnetActivities.TeardownProvider, messages.TeardownProviderRequest{
		RunnerType:    runnerType,
		ProviderState: providerState,
	}).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("failed to teardown provider", zap.Error(err))
		return err
	}
	return nil
}

func waitForTestnetCompletion(ctx workflow.Context, req messages.TestnetWorkflowRequest, testnetRuntime time.Duration, providerState []byte) error {
	if req.LongRunningTestnet {
		signalChan := workflow.GetSignalChannel(ctx, shutdownSignal)
		workflow.GetLogger(ctx).Info("testnet is in long-running mode, waiting for shutdown signal")
		signalChan.Receive(ctx, nil)

		workflow.GetLogger(ctx).Info("received shutdown signal for long running testnet, no resources will be deleted")
		return nil
	}

	workflow.GetLogger(ctx).Info("setting testnet timer", zap.Duration("duration", testnetRuntime))
	if err := workflow.NewTimer(ctx, testnetRuntime).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("timer error", zap.Error(err))
		return err
	}

	return teardownProvider(ctx, req.RunnerType, providerState)
}

func Workflow(ctx workflow.Context, req messages.TestnetWorkflowRequest) (messages.TestnetWorkflowResponse, error) {
	if err := req.Validate(); err != nil {
		return "", temporal.NewApplicationErrorWithOptions("invalid workflow options", err.Error(),
			temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	name := fmt.Sprintf("Testnet (%s) bake", req.ChainConfig.Name)

	if req.LoadTestSpec != nil {
		name = fmt.Sprintf("%s/loadtest-%s", req.ChainConfig.Name, req.LoadTestSpec.Name)
	}

	checkName := fmt.Sprintf("Testnet (%s) bake", name)
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runName := fmt.Sprintf("ib-%s-%s", req.ChainConfig.Name, runID[:6])
	ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

	report, err := NewReport(ctx, checkName, "Launching testnet", "", req)

	if err != nil {
		return "", err
	}

	buildResult, err := buildImageAndReport(ctx, req, report)
	if err != nil {
		return "", err
	}

	if err := runTestnet(ctx, req, runName, buildResult, report); err != nil {
		return "", err
	}

	return "", nil
}

func buildImageAndReport(ctx workflow.Context, req messages.TestnetWorkflowRequest, report *Report) (messages.BuildDockerImageResponse, error) {
	buildResult, err := buildImage(ctx, req)
	if err != nil {
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("image build failed: %s", err.Error()))
		return messages.BuildDockerImageResponse{}, err
	}

	if err := report.SetBuildResult(ctx, buildResult); err != nil {
		workflow.GetLogger(ctx).Error("failed to set build result in report", zap.Error(err))
	}

	return buildResult, nil
}

func launchTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse, report *Report) ([]byte, []byte, error) {
	var providerState, chainState []byte
	providerSpecificOptions := determineProviderOptions(req.RunnerType)

	var createProviderResp messages.CreateProviderResponse
	if err := workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, messages.CreateProviderRequest{
		RunnerType: req.RunnerType,
		Name:       runName,
	}).Get(ctx, &createProviderResp); err != nil {
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("provider creation failed: %s", err.Error()))
		return nil, providerState, err
	}

	providerState = createProviderResp.ProviderState

	var testnetResp messages.LaunchTestnetResponse
	activityOptions := workflow.ActivityOptions{
		HeartbeatTimeout:    time.Second * 10,
		StartToCloseTimeout: time.Hour * 24 * 365,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}

	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, activityOptions), testnetActivities.LaunchTestnet,
		messages.LaunchTestnetRequest{
			Name:                    req.ChainConfig.Name,
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
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("testnet launch failed: %s", err.Error()))
		return nil, providerState, err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	if err := report.SetNodes(ctx, testnetResp.Nodes); err != nil {
		workflow.GetLogger(ctx).Error("failed to set nodes in report", zap.Error(err))
	}

	if err := report.SetDashboards(ctx, req.GrafanaConfig, testnetResp.ChainID); err != nil {
		workflow.GetLogger(ctx).Error("failed to set dashboards in report", zap.Error(err))
	}

	return chainState, providerState, nil
}

func runLoadTest(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte, report *Report) (time.Duration, error) {
	var loadTestTimeout time.Duration
	if req.LoadTestSpec == nil {
		return 0, nil
	}

	workflow.Go(ctx, func(ctx workflow.Context) {
		updateErr := report.UpdateLoadTest(ctx, "Load test in progress", "", nil)
		if updateErr != nil {
			workflow.GetLogger(ctx).Error("failed to update load test status", zap.Error(updateErr))
		}

		loadTestTimeout = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second
		loadTestTimeout = loadTestTimeout + 30*time.Minute

		var loadTestResp messages.RunLoadTestResponse
		activityErr := workflow.ExecuteActivity(
			workflow.WithStartToCloseTimeout(ctx, loadTestTimeout),
			loadTestActivities.RunLoadTest,
			messages.RunLoadTestRequest{
				ChainState:    chainState,
				ProviderState: providerState,
				LoadTestSpec:  *req.LoadTestSpec,
				RunnerType:    req.RunnerType,
			},
		).Get(ctx, &loadTestResp)

		if activityErr != nil {
			workflow.GetLogger(ctx).Error("load test activity failed", zap.Error(activityErr))
			updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+activityErr.Error(), "", nil)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("failed to update load test failure status", zap.Error(updateErr))
			}
			return
		}

		if loadTestResp.Result.Error != "" {
			workflow.GetLogger(ctx).Error("load test reported an error", zap.String("error", loadTestResp.Result.Error))
			updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+loadTestResp.Result.Error, "", &loadTestResp.Result)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("failed to update load test result error status", zap.Error(updateErr))
			}
		} else {
			updateErr := report.UpdateLoadTest(ctx, "✅ Load test completed successfully!", "", &loadTestResp.Result)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("failed to update load test success status", zap.Error(updateErr))
			}
		}
	})

	return loadTestTimeout, nil
}

func determineProviderOptions(runnerType testnettypes.RunnerType) map[string]string {
	if runnerType == testnettypes.DigitalOcean {
		return map[string]string{
			"region":   "ams3",
			"image_id": "185517855",
			"size":     "s-4vcpu-8gb",
		}
	}
	return nil
}

func runTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse, report *Report) error {
	chainState, providerState, err := launchTestnet(ctx, req, runName, buildResult, report)
	if err != nil {
		return err
	}

	loadTestTimeout, err := runLoadTest(ctx, req, chainState, providerState, report)
	if err != nil {
		workflow.GetLogger(ctx).Error("load test initiation failed", zap.Error(err))
	}

	err = setUpdateHandler(ctx, &providerState, &chainState, report, buildResult)
	if err != nil {
		return err
	}

	testnetRuntime := max(defaultRuntime, req.TestnetDuration, loadTestTimeout) // default runtime to 1 hour

	if err := waitForTestnetCompletion(ctx, req, testnetRuntime, providerState); err != nil {
		reportErr := report.Conclude(ctx, "completed", "failed", fmt.Sprintf("Testnet bake failed: %v", err))
		if reportErr != nil {
			workflow.GetLogger(ctx).Error("failed to conclude report and testnet failed", zap.Error(err), zap.Error(reportErr))
		}
		return err
	}

	if reportErr := report.Conclude(ctx, "completed", "success", "Testnet bake completed"); reportErr != nil {
		workflow.GetLogger(ctx).Error("failed to conclude report but testnet was successful", zap.Error(reportErr))
	}

	return nil
}

func setUpdateHandler(ctx workflow.Context, providerState, chainState *[]byte, report *Report,
	buildResult messages.BuildDockerImageResponse) error {
	if err := workflow.SetUpdateHandler(
		ctx,
		updateHandler,
		func(ctx workflow.Context, updateReq messages.TestnetWorkflowRequest) error {
			logger, _ := zap.NewDevelopment()
			stdCtx := context.Background()
			ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

			var p provider.ProviderI
			var err error

			if updateReq.RunnerType == testnettypes.Docker {
				p, err = docker.RestoreProvider(
					stdCtx,
					logger,
					*providerState,
				)
			} else {
				p, err = digitalocean.RestoreProvider(
					stdCtx,
					*providerState,
					"",
					testnetActivities.TailscaleSettings,
					digitalocean.WithLogger(logger),
				)
			}

			if err != nil {
				return fmt.Errorf("failed to restore provider: %w", err)
			}

			chain, err := chain.RestoreChain(stdCtx, logger, p, *chainState, node.RestoreNode,
				testnet.CosmosWalletConfig)

			if err != nil {
				return fmt.Errorf("failed to create chain: %w", err)
			}

			err = chain.Teardown(stdCtx)
			if err != nil {
				return fmt.Errorf("failed to teardown chain: %w", err)
			}

			// update provider and chain state here in case LaunchTestnet activity fails
			*chainState = []byte{}
			*providerState, err = p.SerializeProvider(stdCtx)
			if err != nil {
				return fmt.Errorf("failed to serialize provider: %w", err)
			}

			runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
			runName := fmt.Sprintf("ib-%s-%s", updateReq.ChainConfig.Name, runID[:6])

			return runTestnet(ctx, updateReq, runName, buildResult, report)
		},
	); err != nil {
		return temporal.NewApplicationErrorWithOptions(
			"failed to register update handler",
			err.Error(),
			temporal.ApplicationErrorOptions{NonRetryable: true},
		)
	}

	return nil
}
