package testnet

import (
	"context"
	"fmt"
	"time"

	pb "github.com/skip-mev/ironbird/server/proto"

	"github.com/skip-mev/petri/core/v3/apps"
	"github.com/skip-mev/petri/core/v3/util"

	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"

	"github.com/skip-mev/ironbird/activities/loadbalancer"
	"github.com/skip-mev/ironbird/activities/walletcreator"
	"github.com/skip-mev/ironbird/messages"
	ironbirdutil "github.com/skip-mev/ironbird/util"

	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/testnet"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var testnetActivities *testnet.Activity
var loadTestActivities *loadtest.Activity
var builderActivities *builder.Activity
var loadBalancerActivities *loadbalancer.Activity
var walletCreatorActivities *walletcreator.Activity

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

func teardownProvider(ctx workflow.Context, runnerType messages.RunnerType, providerState []byte) error {
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
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	runName := fmt.Sprintf("ib-%s-%s", req.ChainConfig.Name, util.RandomString(6))
	workflow.GetLogger(ctx).Info("run info", zap.String("run_id", runID), zap.String("run_name", runName), zap.Any("req", req))
	ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

	// If the image is set (e.g. when running a testnet for CometBFT testing)
	chainImageKey := req.ChainConfig.Image
	if chainImageKey == "" {
		switch req.Repo {
		case "gaia":
			chainImageKey = req.Repo
		default:
			// for SDK testing default to simapp
			// todo(nadim-az): keep just one generic simapp image, and cleanup this logic
			chainImageKey = "simapp-v53"
		}
	}

	var buildResult messages.BuildDockerImageResponse
	err := workflow.ExecuteActivity(ctx, builderActivities.BuildDockerImage, messages.BuildDockerImageRequest{
		Repo: req.Repo,
		SHA:  req.SHA,
		ChainConfig: messages.ChainConfig{
			Name:  req.ChainConfig.Name,
			Image: chainImageKey,
		},
	}).Get(ctx, &buildResult)
	if err != nil {
		return "", err
	}

	if err := runTestnet(ctx, req, runName, buildResult, workflowID); err != nil {
		workflow.GetLogger(ctx).Error("testnet workflow failed", zap.Error(err))
		return "", err
	}

	return "", nil
}

func launchTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse) ([]byte, []byte, []*pb.Node, []*pb.Node, error) {
	var providerState, chainState []byte
	providerSpecificOptions := determineProviderOptions(req.RunnerType)

	var createProviderResp messages.CreateProviderResponse
	if err := workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, messages.CreateProviderRequest{
		RunnerType: req.RunnerType,
		Name:       runName,
	}).Get(ctx, &createProviderResp); err != nil {
		return nil, nil, nil, nil, err
	}

	providerState = createProviderResp.ProviderState

	var testnetResp messages.LaunchTestnetResponse
	activityOptions := workflow.ActivityOptions{
		HeartbeatTimeout:    time.Minute * 4,
		StartToCloseTimeout: time.Hour * 24 * 365,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, activityOptions), testnetActivities.LaunchTestnet,
		messages.LaunchTestnetRequest{
			Name:                    req.ChainConfig.Name,
			Repo:                    req.Repo,
			SHA:                     req.SHA,
			Evm:                     req.Evm,
			Image:                   buildResult.FQDNTag,
			GenesisModifications:    req.ChainConfig.GenesisModifications,
			RunnerType:              req.RunnerType,
			NumOfValidators:         req.ChainConfig.NumOfValidators,
			NumOfNodes:              req.ChainConfig.NumOfNodes,
			ProviderSpecificOptions: providerSpecificOptions,
			ProviderState:           providerState,
		}).Get(ctx, &testnetResp); err != nil {
		return nil, providerState, nil, nil, err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	return chainState, providerState, testnetResp.Nodes, testnetResp.Validators, nil
}

func launchLoadBalancer(ctx workflow.Context, req messages.TestnetWorkflowRequest, providerState []byte,
	nodes []*pb.Node) ([]byte, error) {
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	if req.RunnerType != messages.DigitalOcean {
		logger.Info("Skipping loadbalancer creation for non-DigitalOcean runner",
			zap.String("runnerType", string(req.RunnerType)))
		return providerState, nil
	}

	logger.Info("Creating loadbalancer domains for nodes",
		zap.Int("nodeCount", len(nodes)),
		zap.String("workflowID", workflowID))

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

	logger.Info("Executing LaunchLoadBalancer activity",
		zap.Int("domainCount", len(domains)),
		zap.String("workflowID", workflowID))

	if err := workflow.ExecuteActivity(
		ctx,
		loadBalancerActivities.LaunchLoadBalancer,
		messages.LaunchLoadBalancerRequest{
			ProviderState: providerState,
			RunnerType:    req.RunnerType,
			Domains:       domains,
			WorkflowID:    workflowID,
		},
	).Get(ctx, &loadBalancerResp); err != nil {
		logger.Error("Failed to launch loadbalancer", zap.Error(err))
		return providerState, err
	}

	return loadBalancerResp.ProviderState, nil
}

func createWallets(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte) ([]string, error) {
	if req.NumWallets <= 0 {
		workflow.GetLogger(ctx).Info("no wallets to create, using default value of 2500")
		req.NumWallets = 2500
	}

	workflow.GetLogger(ctx).Info("creating wallets", zap.Int("numWallets", req.NumWallets))

	var createWalletsResp messages.CreateWalletsResponse
	err := workflow.ExecuteActivity(
		ctx,
		walletCreatorActivities.CreateWallets,
		messages.CreateWalletsRequest{
			NumWallets:    req.NumWallets,
			Evm:           req.Evm,
			RunnerType:    string(req.RunnerType),
			ChainState:    chainState,
			ProviderState: providerState,
		},
	).Get(ctx, &createWalletsResp)

	if err != nil {
		workflow.GetLogger(ctx).Error("wallet creation activity failed", zap.Error(err))
		return nil, err
	}

	workflow.GetLogger(ctx).Info("wallets created successfully", zap.Int("count", len(createWalletsResp.Mnemonics)))
	return createWalletsResp.Mnemonics, nil
}

func runLoadTest(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte, mnemonics []string) (time.Duration, error) {
	var loadTestTimeout time.Duration
	if req.LoadTestSpec == nil {
		return 0, nil
	}
	workflow.GetLogger(ctx).Info("TestnetWorkflowRequest", zap.Any("req", req))

	workflow.Go(ctx, func(ctx workflow.Context) {
		loadTestTimeout = time.Duration(req.LoadTestSpec.NumOfBlocks*2) * time.Second
		loadTestTimeout = loadTestTimeout + 1*time.Hour

		var loadTestResp messages.RunLoadTestResponse
		req.LoadTestSpec.Evm = req.Evm
		activityErr := workflow.ExecuteActivity(
			workflow.WithStartToCloseTimeout(ctx, loadTestTimeout),
			loadTestActivities.RunLoadTest,
			messages.RunLoadTestRequest{
				ChainState:    chainState,
				ProviderState: providerState,
				LoadTestSpec:  *req.LoadTestSpec,
				RunnerType:    req.RunnerType,
				Evm:           req.Evm,
				Mnemonics:     mnemonics,
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

func determineProviderOptions(runnerType messages.RunnerType) map[string]string {
	if runnerType == messages.DigitalOcean {
		return messages.DigitalOceanDefaultOpts
	}
	return nil
}

func runTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse, workflowID string) error {
	chainState, providerState, nodes, _, err := launchTestnet(ctx, req, runName, buildResult)
	if err != nil {
		return err
	}

	providerState, err = launchLoadBalancer(ctx, req, providerState, nodes)
	if err != nil {
		return err
	}

	mnemonics, err := createWallets(ctx, req, chainState, providerState)
	if err != nil {
		workflow.GetLogger(ctx).Error("failed to create wallets", zap.Error(err))
	}

	loadTestTimeout, err := runLoadTest(ctx, req, chainState, providerState, mnemonics)
	if err != nil {
		workflow.GetLogger(ctx).Error("load test initiation failed", zap.Error(err))
	}

	err = setUpdateHandler(ctx, &providerState, &chainState, buildResult, req.Evm, workflowID)
	if err != nil {
		return err
	}

	testnetRuntime := max(defaultRuntime, req.TestnetDuration, loadTestTimeout)

	if err := waitForTestnetCompletion(ctx, req, testnetRuntime, providerState); err != nil {
		return err
	}

	return nil
}

func setUpdateHandler(ctx workflow.Context, providerState, chainState *[]byte, buildResult messages.BuildDockerImageResponse, Evm bool, workflowID string) error {
	if err := workflow.SetUpdateHandler(
		ctx,
		updateHandler,
		func(ctx workflow.Context, updateReq messages.TestnetWorkflowRequest) error {
			workflow.GetLogger(ctx).Info("received update", zap.Any("updateReq", updateReq))

			stdCtx := context.Background()
			logger, _ := zap.NewDevelopment()

			p, err := ironbirdutil.RestoreProvider(stdCtx, logger, updateReq.RunnerType, *providerState, ironbirdutil.ProviderOptions{
				DOToken: testnetActivities.DOToken, TailscaleSettings: testnetActivities.TailscaleSettings, TelemetrySettings: testnetActivities.TelemetrySettings})

			if err != nil {
				return fmt.Errorf("failed to restore provider: %w", err)
			}

			walletConfig := testnet.CosmosWalletConfig
			if Evm {
				walletConfig = testnet.EvmCosmosWalletConfig
			}
			chain, err := chain.RestoreChain(stdCtx, logger, p, *chainState, node.RestoreNode, walletConfig)

			if err != nil {
				return fmt.Errorf("failed to restore chain: %w", err)
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

			return runTestnet(ctx, updateReq, runName, buildResult, workflowID)
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
