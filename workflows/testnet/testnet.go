package testnet

import (
	"context"
	"fmt"

	"github.com/skip-mev/petri/core/v3/apps"
	"github.com/skip-mev/petri/core/v3/util"
	"time"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"

	"github.com/skip-mev/ironbird/activities/loadbalancer"
	"github.com/skip-mev/ironbird/messages"

	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/testnet"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var testnetActivities *testnet.Activity
var loadTestActivities *loadtest.Activity
var loadBalancerActivities *loadbalancer.Activity

type monitoringState struct {
	cancelChan workflow.Channel
	errChan    workflow.Channel
	doneChan   workflow.Channel
}

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

	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runName := fmt.Sprintf("ib-%s-%s", req.ChainConfig.Name, util.RandomString(6))
	workflow.GetLogger(ctx).Info("run info", zap.String("run_id", runID), zap.String("run_name", runName))
	ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

	buildResult, err := buildImage(ctx, req)
	if err != nil {
		return "", err
	}

	if err := runTestnet(ctx, req, runName, buildResult); err != nil {
		workflow.GetLogger(ctx).Error("testnet workflow failed", zap.Error(err))
		return "", err
	}

	//chainState, providerState, nodes, err := launchTestnet(ctx, req, runName, buildResult, report)
	//if err != nil {
	//	return "", err
	//}
	//
	//providerState, err = launchLoadBalancer(ctx, req, providerState, nodes, report)
	//if err != nil {
	//	return "", err
	//}
	//
	//loadTestTimeout, err := runLoadTest(ctx, req, chainState, providerState, report)
	//if err != nil {
	//	return "", err
	//}

	return "", nil
}

func launchTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse) ([]byte, []byte, []testnettypes.Node, error) {
	var providerState, chainState []byte
	providerSpecificOptions := determineProviderOptions(req.RunnerType)

	var createProviderResp messages.CreateProviderResponse
	if err := workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, messages.CreateProviderRequest{
		RunnerType: req.RunnerType,
		Name:       runName,
	}).Get(ctx, &createProviderResp); err != nil {
		return nil, nil, nil, err
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
		return nil, providerState, nil, err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	return chainState, providerState, testnetResp.Nodes, nil
}

func launchLoadBalancer(ctx workflow.Context, req messages.TestnetWorkflowRequest, providerState []byte, nodes []testnettypes.Node, report *Report) ([]byte, error) {
	if req.RunnerType != testnettypes.DigitalOcean {
		return providerState, nil
	}

	var loadBalancerResp messages.LaunchLoadBalancerResponse

	var domains []apps.LoadBalancerDomain
	for _, node := range nodes {
		domains = append(domains, apps.LoadBalancerDomain{
			Domain:   fmt.Sprintf("%s-grpc", node.Name),
			IP:       fmt.Sprintf("%s:9090", node.Address),
			Protocol: "grpc",
		})

		domains = append(domains, apps.LoadBalancerDomain{
			Domain:   fmt.Sprintf("%s-rpc", node.Name),
			IP:       fmt.Sprintf("%s:26657", node.Address),
			Protocol: "http",
		})

		domains = append(domains, apps.LoadBalancerDomain{
			Domain:   fmt.Sprintf("%s-lcd", node.Name),
			IP:       fmt.Sprintf("%s:1317", node.Address),
			Protocol: "http",
		})
	}

	if err := workflow.ExecuteActivity(
		ctx,
		loadBalancerActivities.LaunchLoadBalancer,
		messages.LaunchLoadBalancerRequest{
			ProviderState: providerState,
			RunnerType:    req.RunnerType,
			Domains:       domains,
		},
	).Get(ctx, &loadBalancerResp); err != nil {
		return providerState, err
	}

	var reformedNodes []testnettypes.Node

	for _, node := range nodes {
		reformedNodes = append(reformedNodes, testnettypes.Node{
			Name:    node.Name,
			Address: node.Address,
			Rpc:     fmt.Sprintf("https://%s-rpc.%s", node.Name, loadBalancerResp.RootDomain),
			Lcd:     fmt.Sprintf("https://%s-lcd.%s", node.Name, loadBalancerResp.RootDomain),
			Metrics: node.Address,
		})
	}

	return loadBalancerResp.ProviderState, nil
}

func runLoadTest(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte) (time.Duration, error) {
	var loadTestTimeout time.Duration
	if req.LoadTestSpec == nil {
		return 0, nil
	}

	workflow.Go(ctx, func(ctx workflow.Context) {

		loadTestTimeout = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second
		loadTestTimeout = loadTestTimeout + 1*time.Hour

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
			return
		}

		if loadTestResp.Result.Error != "" {
			workflow.GetLogger(ctx).Error("load test reported an error", zap.String("error", loadTestResp.Result.Error))
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
	chainState, providerState, nodes, err := launchTestnet(ctx, req, runName, buildResult, report)
func runTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse) error {
	chainState, providerState, err := launchTestnet(ctx, req, runName, buildResult)
	if err != nil {
		return err
	}

	loadTestTimeout, err := runLoadTest(ctx, req, chainState, providerState)
	if err != nil {
		workflow.GetLogger(ctx).Error("load test initiation failed", zap.Error(err))
	}

	providerState, err = launchLoadBalancer(ctx, req, providerState, nodes, report)
	if err != nil {
		return err
	}

	err = setUpdateHandler(ctx, &providerState, &chainState, report, buildResult)
	err = setUpdateHandler(ctx, &providerState, &chainState, buildResult)
	if err != nil {
		return err
	}

	testnetRuntime := max(defaultRuntime, req.TestnetDuration, loadTestTimeout) // default runtime to 1 hour

	if err := waitForTestnetCompletion(ctx, req, testnetRuntime, providerState); err != nil {
		return err
	}

	return nil
}

func setUpdateHandler(ctx workflow.Context, providerState, chainState *[]byte, buildResult messages.BuildDockerImageResponse) error {
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
			runName := fmt.Sprintf("ib-%s-%s", updateReq.ChainConfig.Name, util.RandomString(6))
			workflow.GetLogger(ctx).Info("run info", zap.String("run_id", runID),
				zap.String("run_name", runName))

			return runTestnet(ctx, updateReq, runName, buildResult)
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
