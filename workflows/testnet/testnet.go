package testnet

import (
	"fmt"
	"time"

	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/observability"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/petri/core/v3/monitoring"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var testnetActivities *testnet.Activity
var githubActivities *github.NotifierActivity
var observabilityActivities *observability.Activity
var loadTestActivities *loadtest.Activity

func Workflow(ctx workflow.Context, opts WorkflowOptions) (string, error) {
	name := fmt.Sprintf("Testnet (%s) bake", opts.ChainConfig.Name)

	runName := fmt.Sprintf("ib-%s-%s", opts.ChainConfig.Name, opts.SHA[:6])
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	report, err := NewReport(
		ctx,
		name,
		"Launching testnet",
		"",
		&opts,
	)

	if err != nil {
		return "", err
	}

	builtTag, err := buildImage(ctx, opts)

	if err != nil {
		return "", err
	}

	testnetOptions := testnet.TestnetOptions{
		Name:                 runName,
		Image:                builtTag,
		UID:                  opts.ChainConfig.Image.UID,
		GID:                  opts.ChainConfig.Image.GID,
		BinaryName:           opts.ChainConfig.Image.BinaryName,
		HomeDir:              opts.ChainConfig.Image.HomeDir,
		GenesisModifications: opts.ChainConfig.GenesisModifications,
		ProviderSpecificOptions: map[string]string{
			"region":   "ams3",
			"image_id": "177869680",
			"size":     "s-1vcpu-1gb",
		},
		ValidatorCount: 1,
		NodeCount:      0,
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
		metricsIps = append(metricsIps, node.Metrics)
	}

	if err := workflow.ExecuteActivity(
		ctx,
		observabilityActivities.LaunchObservabilityStack,
		observability.Options{
			PrometheusTargets:      metricsIps,
			ProviderState:          testnetOptions.ProviderState,
			ProviderSpecificConfig: testnetOptions.ProviderSpecificOptions,
		},
	).Get(ctx, &observabilityPackagedState); err != nil {
		return "", err
	}

	testnetOptions.ProviderState = observabilityPackagedState.ProviderState

	if err := report.SetObservabilityURL(ctx, observabilityPackagedState.ExternalGrafanaURL); err != nil {
		return "", err
	}

	if opts.LoadTestConfig != nil {
		workflow.Go(ctx, func(ctx workflow.Context) {
			duration, err := time.ParseDuration(opts.LoadTestConfig.Runtime)
			if err != nil {
				workflow.GetLogger(ctx).Error("Load test failed with error", zap.Error(err))
				updateErr := report.UpdateLoadTest(ctx, "❌ Failed to begin load test: "+err.Error(), "", nil)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
				return
			}

			// Add 5 minute buffer to the duration
			duration = duration + 5*time.Minute

			configStr := fmt.Sprintf(
				"- Runtime: %s\n"+
					"- Block Gas Limit Target: %.2f%%\n"+
					"- Number of Blocks: %d\n"+
					"- Message Types: %v\n",
				opts.LoadTestConfig.Runtime,
				opts.LoadTestConfig.BlockGasLimitTarget*100,
				opts.LoadTestConfig.NumOfBlocks,
				opts.LoadTestConfig.Msgs)

			err = report.UpdateLoadTest(ctx, "Load test in progress", configStr, nil)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(err))
				return
			}

			var state loadtest.PackagedState
			if err := workflow.ExecuteActivity(
				workflow.WithStartToCloseTimeout(ctx, duration),
				loadTestActivities.RunLoadTest,
				testnetOptions.ChainState,
				opts.LoadTestConfig,
				testnetOptions.ProviderState,
			).Get(ctx, &state); err != nil {
				workflow.GetLogger(ctx).Error("Load test failed with error", zap.Error(err))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+err.Error(), configStr, nil)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
				return
			}

			if state.Result.Error != "" {
				workflow.GetLogger(ctx).Error("Load test reported an error", zap.String("error", state.Result.Error))
				updateErr := report.UpdateLoadTest(ctx, "❌ Load test failed: "+state.Result.Error, configStr, &state.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			} else {
				updateErr := report.UpdateLoadTest(ctx, "✅ Load test completed successfully!", configStr, &state.Result)
				if updateErr != nil {
					workflow.GetLogger(ctx).Error("Failed to update load test status", zap.Error(updateErr))
				}
			}
		})
	}

	if err := monitorTestnet(ctx, testnetOptions, opts.LoadTestConfig, report, observabilityPackagedState.ExternalGrafanaURL); err != nil {
		return "", err
	}

	if err := report.Conclude(ctx, "completed", "success", "Testnet bake completed"); err != nil {
		return "", err
	}

	return "", nil
}

func monitorTestnet(ctx workflow.Context, testnetOptions testnet.TestnetOptions, loadTestConfig *LoadTestConfig, report *Report, grafanaUrl string) error {
	// Calculate number of iterations (each iteration is 10 seconds)
	iterations := 360 // default to 1 hour (360 * 10 seconds)
	if loadTestConfig != nil {
		duration, err := time.ParseDuration(loadTestConfig.Runtime)
		if err != nil {
			return fmt.Errorf("failed to parse load test runtime: %w", err)
		}
		// Add 2 minute buffer and convert to iterations
		duration = duration + 2*time.Minute
		iterations = int(duration.Seconds() / 10)
		if iterations < 360 { // ensure we run for at least the default time
			iterations = 360
		}
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

		var screenshot []byte
		err = workflow.ExecuteActivity(ctx, observabilityActivities.GrabGraphScreenshot, grafanaUrl, monitoring.DefaultDashboardUID, "comet-performance", "18", "now-5m").Get(ctx, &screenshot)

		if err != nil {
			return err
		}

		var screenShotUrl string

		err = workflow.ExecuteActivity(ctx, observabilityActivities.UploadScreenshot, testnetOptions.Name, fmt.Sprintf("testnet-%d", time.Now().Unix()), screenshot).Get(ctx, &screenShotUrl)

		if err != nil {
			return err
		}

		if err := report.SetScreenshots(ctx, map[string]string{
			"Average block latency": screenShotUrl,
		}); err != nil {
			return err
		}

		if err := report.SetStatus(ctx, "in_progress", "Monitoring the testnet", fmt.Sprintf("Monitoring the testnet - %d", i)); err != nil {
			return err
		}
	}

	return nil
}
