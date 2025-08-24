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
	defaultRuntime  = time.Minute * 2
	loadTestTimeout = time.Hour
	updateHandler   = "chain_update"
	shutdownSignal  = "shutdown"
)

var (
	defaultWorkflowOptions = workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
)

func teardownProvider(ctx workflow.Context, runnerType messages.RunnerType, providerState []byte) {
	workflow.GetLogger(ctx).Info("tearing down provider")
	err := workflow.ExecuteActivity(ctx, testnetActivities.TeardownProvider, messages.TeardownProviderRequest{
		RunnerType:    runnerType,
		ProviderState: providerState,
	}).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("failed to teardown provider", zap.Error(err))
	}
}

func waitForTestnetCompletion(ctx workflow.Context, req messages.TestnetWorkflowRequest,
	selector workflow.Selector, providerState []byte) {
	logger := workflow.GetLogger(ctx)
	// 2. Long-running testnet does not end
	if req.LongRunningTestnet {
		logger.Info("testnet is in long-running mode")
		f, setter := workflow.NewFuture(ctx)
		workflow.Go(ctx, func(ctx workflow.Context) {
			logger.Info("waiting for shutdown signal")
			signalChan := workflow.GetSignalChannel(ctx, shutdownSignal)
			signalChan.Receive(ctx, nil)
			logger.Info("received shutdown signal for testnet")
			setter.SetError(nil)
		})
		selector.AddFuture(f, func(_ workflow.Future) {})
	} else if req.CosmosLoadTestSpec == nil && req.EthereumLoadTestSpec == nil {
		testnetDuration := defaultRuntime
		if req.TestnetDuration != "" {
			var err error
			testnetDuration, err = time.ParseDuration(req.TestnetDuration)
			if err != nil {
				logger.Error("failed to parse testnet duration, falling back to default runtime",
					zap.String("duration", req.TestnetDuration))
				testnetDuration = defaultRuntime
			}
		}

		// 3. No load test and not long-running will end after the timeout timer
		networkTimeout := max(testnetDuration, defaultRuntime)
		f := workflow.NewTimer(ctx, networkTimeout)
		selector.AddFuture(f, func(_ workflow.Future) {})
	}
}

func Workflow(ctx workflow.Context, req messages.TestnetWorkflowRequest) (messages.TestnetWorkflowResponse, error) {
	if err := req.Validate(); err != nil {
		return "", temporal.NewApplicationErrorWithOptions("invalid workflow options", err.Error(),
			temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	runName := fmt.Sprintf("ib-%s", util.RandomString(6))
	workflow.GetLogger(ctx).Info("run info", zap.String("run_id", runID), zap.String("run_name", runName), zap.Any("req", req))
	ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

	if req.ChainConfig.Image == "" {
		switch req.Repo {
		case "gaia":
			req.ChainConfig.Image = req.Repo
		default:
			// for SDK testing default to simapp
			// todo(nadim-az): keep just one generic simapp image, and cleanup this logic
			req.ChainConfig.Image = "simapp-v53"
		}
	}

	var buildResult messages.BuildDockerImageResponse
	err := workflow.ExecuteActivity(ctx, builderActivities.BuildDockerImage, messages.BuildDockerImageRequest{
		Repo: req.Repo,
		SHA:  req.SHA,
		ChainConfig: messages.ChainConfig{
			Name:  req.ChainConfig.Name,
			Image: req.ChainConfig.Image,
		},
	}).Get(ctx, &buildResult)
	if err != nil {
		return "", err
	}

	if err := startWorkflow(ctx, req, runName, buildResult, workflowID); err != nil {
		workflow.GetLogger(ctx).Error("testnet workflow failed", zap.Error(err))
		return "", err
	}

	return "", nil
}

func launchTestnet(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string,
	buildResult messages.BuildDockerImageResponse) ([]byte, []byte, []*pb.Node, []*pb.Node, error) {
	var providerState, chainState []byte
	workflow.GetLogger(ctx).Info("launching testnet", zap.Any("req", req))

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
		HeartbeatTimeout:    time.Minute * 10,
		StartToCloseTimeout: time.Hour * 24 * 365,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, activityOptions), testnetActivities.LaunchTestnet,
		messages.LaunchTestnetRequest{
			Name:                  req.ChainConfig.Name,
			Repo:                  req.Repo,
			SHA:                   req.SHA,
			IsEvmChain:            req.IsEvmChain,
			Image:                 buildResult.FQDNTag,
			BaseImage:             req.ChainConfig.Image,
			GenesisModifications:  req.ChainConfig.GenesisModifications,
			RunnerType:            req.RunnerType,
			NumOfValidators:       req.ChainConfig.NumOfValidators,
			NumOfNodes:            req.ChainConfig.NumOfNodes,
			RegionConfigs:         req.ChainConfig.RegionConfigs,
			CustomAppConfig:       req.ChainConfig.CustomAppConfig,
			CustomConsensusConfig: req.ChainConfig.CustomConsensusConfig,
			CustomClientConfig:    req.ChainConfig.CustomClientConfig,
			SetSeedNode:           req.ChainConfig.SetSeedNode,
			SetPersistentPeers:    req.ChainConfig.SetPersistentPeers,
			ProviderState:         providerState,
		}).Get(ctx, &testnetResp); err != nil {
		return nil, providerState, nil, nil, err
	}

	chainState = testnetResp.ChainState
	providerState = testnetResp.ProviderState

	return chainState, providerState, testnetResp.Nodes, testnetResp.Validators, nil
}

func launchLoadBalancer(ctx workflow.Context, req messages.TestnetWorkflowRequest, providerState []byte,
	nodes []*pb.Node, validators []*pb.Node) ([]byte, error) {
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
	domains := processDomainInfo(req.ChainConfig.Name, nodes, validators, req.IsEvmChain)

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

func createWallets(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte, workflowID string) ([]string, error) {
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
			WorkflowID:    workflowID,
			NumWallets:    req.NumWallets,
			IsEvmChain:    req.IsEvmChain,
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

func runLoadTest(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte,
	mnemonics []string, selector workflow.Selector) error {
	if req.EthereumLoadTestSpec != nil {
		workflow.Go(ctx, func(ctx workflow.Context) {
			var loadTestResp messages.RunLoadTestResponse
			f := workflow.ExecuteActivity(
				workflow.WithStartToCloseTimeout(ctx, loadTestTimeout),
				loadTestActivities.RunLoadTest,
				messages.RunLoadTestRequest{
					ChainState:    chainState,
					ProviderState: providerState,
					LoadTestSpec:  *req.EthereumLoadTestSpec,
					RunnerType:    req.RunnerType,
					IsEvmChain:    req.IsEvmChain,
					Mnemonics:     mnemonics,
				},
			)

			selector.AddFuture(f, func(f workflow.Future) {
				activityErr := f.Get(ctx, &loadTestResp)
				if activityErr != nil {
					workflow.GetLogger(ctx).Error("ethereum load test failed", zap.Error(activityErr))
				} else if loadTestResp.Result.Error != "" {
					workflow.GetLogger(ctx).Error("ethereum load test reported an error", zap.String("error", loadTestResp.Result.Error))
				}
			})

		})
	} else if req.CosmosLoadTestSpec != nil {
		workflow.Go(ctx, func(ctx workflow.Context) {
			var loadTestResp messages.RunLoadTestResponse
			f := workflow.ExecuteActivity(
				workflow.WithStartToCloseTimeout(ctx, loadTestTimeout),
				loadTestActivities.RunLoadTest,
				messages.RunLoadTestRequest{
					ChainState:    chainState,
					ProviderState: providerState,
					LoadTestSpec:  *req.CosmosLoadTestSpec,
					RunnerType:    req.RunnerType,
					IsEvmChain:    req.IsEvmChain,
					Mnemonics:     mnemonics,
				},
			)

			selector.AddFuture(f, func(f workflow.Future) {
				activityErr := f.Get(ctx, &loadTestResp)
				if activityErr != nil {
					workflow.GetLogger(ctx).Error("cosmos load test failed", zap.Error(activityErr))
				} else if loadTestResp.Result.Error != "" {
					workflow.GetLogger(ctx).Error("cosmos load test reported an error", zap.String("error", loadTestResp.Result.Error))
				}
			})

		})
	}

	return nil
}

func startWorkflow(ctx workflow.Context, req messages.TestnetWorkflowRequest, runName string, buildResult messages.BuildDockerImageResponse, workflowID string) error {
	chainState, providerState, nodes, validators, err := launchTestnet(ctx, req, runName, buildResult)
	if err != nil {
		return err
	}

	if req.LaunchLoadBalancer {
		providerState, err = launchLoadBalancer(ctx, req, providerState, nodes, validators)
		if err != nil {
			return err
		}
	}

	mnemonics, err := createWallets(ctx, req, chainState, providerState, workflowID)
	if err != nil {
		workflow.GetLogger(ctx).Error("failed to create wallets", zap.Error(err))
	}

	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	defer func() {
		teardownProvider(cleanupCtx, req.RunnerType, providerState)
	}()

	shutdownSelector := workflow.NewSelector(ctx)
	// 1. load test selector
	err = runLoadTest(ctx, req, chainState, providerState, mnemonics, shutdownSelector)
	if err != nil {
		workflow.GetLogger(ctx).Error("load test initiation failed", zap.Error(err))
	}

	err = setUpdateHandler(ctx, &providerState, &chainState, req.IsEvmChain, workflowID)
	if err != nil {
		return err
	}

	waitForTestnetCompletion(ctx, req, shutdownSelector, providerState)

	// Wait for shutdown
	shutdownSelector.Select(ctx)

	return nil
}

func setUpdateHandler(ctx workflow.Context, providerState, chainState *[]byte, isEvmChain bool, workflowID string) error {
	if err := workflow.SetUpdateHandler(
		ctx,
		updateHandler,
		func(ctx workflow.Context, updateReq messages.TestnetWorkflowRequest) error {
			workflow.GetLogger(ctx).Info("received update", zap.Any("updateReq", updateReq))

			ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

			stdCtx := context.Background()
			logger, _ := zap.NewDevelopment()

			p, err := ironbirdutil.RestoreProvider(stdCtx, logger, updateReq.RunnerType, *providerState, ironbirdutil.ProviderOptions{
				DOToken: testnetActivities.DOToken, TailscaleSettings: testnetActivities.TailscaleSettings, TelemetrySettings: testnetActivities.TelemetrySettings})

			if err != nil {
				return fmt.Errorf("failed to restore provider: %w", err)
			}

			walletConfig := testnet.CosmosWalletConfig
			if isEvmChain {
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

			var buildResult messages.BuildDockerImageResponse
			err = workflow.ExecuteActivity(ctx, builderActivities.BuildDockerImage, messages.BuildDockerImageRequest{
				Repo: updateReq.Repo,
				SHA:  updateReq.SHA,
				ChainConfig: messages.ChainConfig{
					Name:  updateReq.ChainConfig.Name,
					Image: updateReq.ChainConfig.Image,
				},
			}).Get(ctx, &buildResult)
			if err != nil {
				return fmt.Errorf("failed to build docker image for update request: %w", err)
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

			if updateReq.ChainConfig.Image == "" {
				switch updateReq.Repo {
				case "gaia":
					updateReq.ChainConfig.Image = updateReq.Repo
				default:
					// for SDK testing default to simapp
					// todo(nadim-az): keep just one generic simapp image, and cleanup this logic
					updateReq.ChainConfig.Image = "simapp-v53"
				}
			}

			return startWorkflow(ctx, updateReq, runName, buildResult, workflowID)
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

func processDomainInfo(chainName string, nodes []*pb.Node, validators []*pb.Node, isEvmChain bool) []apps.LoadBalancerDomain {
	var domains []apps.LoadBalancerDomain

	domainTypes := map[string]string{
		"grpc": "9090",
		"rpc":  "26657",
		"lcd":  "1317",
	}

	if isEvmChain {
		domainTypes["evmrpc"] = "8545"
		domainTypes["evmws"] = "8546"
	}

	for domainType, port := range domainTypes {
		domain := apps.LoadBalancerDomain{Domain: fmt.Sprintf("%s-%s", chainName, domainType)}

		var ips []string
		for _, node := range append(nodes, validators...) {
			ips = append(ips, fmt.Sprintf("%s:%s", node.Address, port))
		}

		if domainType == "grpc" {
			domain.Protocol = "grpc"
		} else {
			domain.Protocol = "http"
		}

		domain.IPs = ips

		domains = append(domains, domain)
	}

	return domains
}
