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

type monitoringState struct {
	cancelChan workflow.Channel
	errChan    workflow.Channel
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

	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	report, err := NewReport(ctx, checkName, "Launching testnet", "", req)

	if err != nil {
		return "", err
	}

	buildResult, err := buildImageAndReport(ctx, req, report)
	if err != nil {
		return "", err
	}

	chainState, providerState, err := launchTestnetInternal(ctx, req, runName, buildResult, report)
	if err != nil {
		return "", err
	}

	loadTestRuntime, err := runLoadTestInternal(ctx, req, chainState, providerState, report)
	if err != nil {
		workflow.GetLogger(ctx).Error("Load test initiation failed", zap.Error(err))
	}

	testnetRuntime := calculateTestnetRuntime(req.TestnetDuration, loadTestRuntime)

	monitorState := &monitoringState{
		cancelChan: workflow.NewChannel(ctx),
		errChan:    workflow.NewChannel(ctx),
	}

	startMonitoring(ctx, monitorState, chainState, providerState, req.RunnerType, report, testnetRuntime, req.LongRunningTestnet)

	err = setUpdateHandler(ctx, &providerState, &chainState, monitorState, report)
	if err != nil {
		return "", err
	}

	var receivedErr error
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(monitorState.errChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &receivedErr)
	})
	selector.Select(ctx)

	if receivedErr != nil {
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("Testnet monitoring failed: %s", receivedErr.Error()))
		return "", receivedErr
	}

	if err := report.Conclude(ctx, "completed", "success", "Testnet bake completed"); err != nil {
		return "", err
	}

	if !req.LongRunningTestnet {
		workflow.GetLogger(ctx).Info("tearing down provider")
		err := workflow.ExecuteActivity(ctx, testnetActivities.TeardownProvider, messages.TeardownProviderRequest{
			RunnerType:    req.RunnerType,
			ProviderState: providerState,
		}).Get(ctx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to teardown provider after monitoring", "error", err)
			return "", err
		}
	}

	return "", nil
}

func startMonitoring(ctx workflow.Context, state *monitoringState, chainState, providerState []byte, runnerType testnettypes.RunnerType, report *Report, testnetRuntime time.Duration, longRunningTestnet bool) {
	workflow.Go(ctx, func(ctx workflow.Context) {
		monitorCtx, cancel := workflow.WithCancel(ctx)
		defer cancel()

		selector := workflow.NewSelector(monitorCtx)
		selector.AddReceive(state.cancelChan, func(c workflow.ReceiveChannel, more bool) {
			workflow.GetLogger(ctx).Info("Monitoring cancelled")
		})
		selector.AddDefault(func() {
			err := monitorTestnet(monitorCtx, chainState, providerState, runnerType, report, testnetRuntime, longRunningTestnet)
			if err != nil {
				state.errChan.Send(ctx, err)
			}
		})
		selector.Select(ctx)
	})
}

func buildImageAndReport(ctx workflow.Context, req messages.TestnetWorkflowRequest, report *Report) (messages.BuildDockerImageResponse, error) {
	buildResult, err := buildImage(ctx, req)
	if err != nil {
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("Image build failed: %s", err.Error()))
		return messages.BuildDockerImageResponse{}, err
	}

	if err := report.SetBuildResult(ctx, buildResult); err != nil {
		workflow.GetLogger(ctx).Error("Failed to set build result in report", zap.Error(err))
	}

	return buildResult, nil
}

func launchTestnetInternal(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse, report *Report) ([]byte, []byte, error) {
	var providerState, chainState []byte
	providerSpecificOptions := determineProviderOptions(req.RunnerType)

	var createProviderResp messages.CreateProviderResponse
	if err := workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, messages.CreateProviderRequest{
		RunnerType: req.RunnerType,
		Name:       runName,
	}).Get(ctx, &createProviderResp); err != nil {
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("Provider creation failed: %s", err.Error()))
		return nil, nil, err
	}

	providerState = createProviderResp.ProviderState

	var testnetResp messages.LaunchTestnetResponse

	if err := workflow.ExecuteActivity(ctx, testnetActivities.LaunchTestnet, messages.LaunchTestnetRequest{
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
		_ = report.Conclude(ctx, "failed", "error", fmt.Sprintf("Testnet launch failed: %s", err.Error()))
		return nil, nil, err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	if err := report.SetNodes(ctx, testnetResp.Nodes); err != nil {
		workflow.GetLogger(ctx).Error("Failed to set nodes in report", zap.Error(err))
	}

	if err := report.SetDashboards(ctx, req.GrafanaConfig, testnetResp.ChainID); err != nil {
		workflow.GetLogger(ctx).Error("Failed to set dashboards in report", zap.Error(err))
	}

	return chainState, providerState, nil
}

func runLoadTestInternal(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte, report *Report) (time.Duration, error) {
	var loadTestRuntime time.Duration
	if req.LoadTestSpec == nil {
		return 0, nil
	}

	workflow.Go(ctx, func(ctx workflow.Context) {
		updateErr := report.UpdateLoadTest(ctx, "Load test in progress", "", nil)
		if updateErr != nil {
			workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
		}

		loadTestRuntime = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second
		loadTestRuntime = loadTestRuntime + 30*time.Minute

		var loadTestResp messages.RunLoadTestResponse
		activityErr := workflow.ExecuteActivity(
			workflow.WithStartToCloseTimeout(ctx, loadTestRuntime),
			loadTestActivities.RunLoadTest,
			messages.RunLoadTestRequest{
				ChainState:    chainState,
				ProviderState: providerState,
				LoadTestSpec:  *req.LoadTestSpec,
				RunnerType:    req.RunnerType,
			},
		).Get(ctx, &loadTestResp)

		if activityErr != nil {
			workflow.GetLogger(ctx).Error("Load test activity failed", zap.Error(activityErr))
			updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+activityErr.Error(), "", nil)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test failure status", zap.Error(updateErr))
			}
			return
		}

		if loadTestResp.Result.Error != "" {
			workflow.GetLogger(ctx).Error("Load test reported an error", zap.String("error", loadTestResp.Result.Error))
			updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+loadTestResp.Result.Error, "", &loadTestResp.Result)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test result error status", zap.Error(updateErr))
			}
		} else {
			updateErr := report.UpdateLoadTest(ctx, "✅ Load test completed successfully!", "", &loadTestResp.Result)
			if updateErr != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test success status", zap.Error(updateErr))
			}
		}
	})

	loadTestRuntime = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second

	return loadTestRuntime, nil
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

func monitorTestnet(ctx workflow.Context, chainState, providerState []byte, runnerType testnettypes.RunnerType, report *Report,
	runtime time.Duration, longRunningTestnet bool) error {
	startTime := workflow.Now(ctx)
	sleepDuration := 10 * time.Second

	for {
		if !longRunningTestnet && workflow.Now(ctx).Sub(startTime) >= runtime {
			break
		}

		if err := workflow.Sleep(ctx, sleepDuration); err != nil {
			if temporal.IsCanceledError(err) {
				workflow.GetLogger(ctx).Info("Monitoring loop cancelled.")
				return nil
			}
			workflow.GetLogger(ctx).Error("workflow failed to sleep", zap.Error(err))
			continue
		}

		var monitorTestnetResp messages.MonitorTestnetResponse
		err := workflow.ExecuteActivity(ctx, testnetActivities.MonitorTestnet, messages.MonitorTestnetRequest{
			RunnerType:    runnerType,
			ChainState:    chainState,
			ProviderState: providerState,
		}).Get(ctx, &monitorTestnetResp)

		if err != nil {
			if temporal.IsCanceledError(err) {
				workflow.GetLogger(ctx).Info("Monitoring activity cancelled.")
				return nil
			}
			workflow.GetLogger(ctx).Error("MonitorTestnet error", zap.Error(err))
		}

		elapsedMinutes := int(workflow.Now(ctx).Sub(startTime).Minutes())
		statusMsg := fmt.Sprintf("Monitoring the testnet - %d minutes elapsed", elapsedMinutes)
		if longRunningTestnet {
			statusMsg = fmt.Sprintf("Monitoring the testnet (long-running) - %d", elapsedMinutes)
		}

		if err := report.SetStatus(ctx, "in_progress", "Monitoring the testnet", statusMsg); err != nil {
			workflow.GetLogger(ctx).Error("Failed to update report status", zap.Error(err))
		}
	}

	workflow.GetLogger(ctx).Info("Monitoring finished.")
	return nil
}

func setUpdateHandler(ctx workflow.Context, providerState, chainState *[]byte, monitorState *monitoringState, report *Report) error {
	if err := workflow.SetUpdateHandler(
		ctx,
		"chain_update",
		func(ctx workflow.Context, updateReq messages.TestnetWorkflowRequest) error {
			logger, _ := zap.NewDevelopment()
			stdCtx := context.Background()

			monitorState.cancelChan.Send(ctx, struct{}{})
			monitorState.cancelChan = workflow.NewChannel(ctx)

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
			pState, err := p.SerializeProvider(stdCtx)
			if err != nil {
				return fmt.Errorf("failed to serialize provider: %w", err)
			}
			*providerState = pState

			var testnetResp messages.LaunchTestnetResponse
			if err := workflow.ExecuteActivity(
				ctx,
				testnetActivities.LaunchTestnet,
				updateReq,
			).Get(ctx, &testnetResp); err != nil {
				return err
			}

			*providerState = testnetResp.ProviderState
			*chainState = testnetResp.ChainState

			var loadTestRuntime time.Duration
			if updateReq.LoadTestSpec != nil {
				loadTestRuntime, err = runLoadTestInternal(ctx, updateReq, *chainState, *providerState, report)
				if err != nil {
					workflow.GetLogger(ctx).Error("Load test initiation failed during update", zap.Error(err))
				}
			}

			testnetRuntime := calculateTestnetRuntime(updateReq.TestnetDuration, loadTestRuntime)

			startMonitoring(ctx, monitorState, *chainState, *providerState, updateReq.RunnerType, report, testnetRuntime, updateReq.LongRunningTestnet)

			return nil
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

func calculateTestnetRuntime(requestedDuration time.Duration, loadTestRuntime time.Duration) time.Duration {
	defaultRuntime := 1 * time.Hour

	if requestedDuration == 0 {
		requestedDuration = defaultRuntime
	}

	if loadTestRuntime > requestedDuration {
		return loadTestRuntime
	}

	return requestedDuration
}
