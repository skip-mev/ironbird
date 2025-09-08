package testnet

import (
	"fmt"
	"time"

	pb "github.com/skip-mev/ironbird/server/proto"

	"github.com/skip-mev/ironbird/petri/core/apps"
	"github.com/skip-mev/ironbird/petri/core/util"

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
)

var (
	defaultWorkflowOptions = workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 1,
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
	selector workflow.Selector) {
	logger := workflow.GetLogger(ctx)
	// 2. Long-running testnet does not end
	if req.LongRunningTestnet {
		logger.Info("testnet is in long-running mode - will run until workflow is cancelled")
		f, _ := workflow.NewFuture(ctx)
		selector.AddFuture(f, func(_ workflow.Future) {})
	} else {
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

		networkTimeout := max(testnetDuration, defaultRuntime)
		logger.Info("network timeout", zap.Duration("timeout", networkTimeout))
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
		case "evm":
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
		if temporal.IsCanceledError(err) {
			workflow.GetLogger(ctx).Info("testnet workflow was cancelled, completing gracefully")
			return "", nil
		}
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
		//HeartbeatTimeout:    time.Hour * 1,
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
		compressedProviderState, compressErr := ironbirdutil.CompressData(providerState)
		if compressErr != nil {
			workflow.GetLogger(ctx).Error("failed to compress provider state for cleanup", zap.Error(compressErr))
			return nil, providerState, nil, nil, err
		}
		return nil, compressedProviderState, nil, nil, err
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
					ChainState:      chainState,
					ProviderState:   providerState,
					LoadTestSpec:    *req.EthereumLoadTestSpec,
					RunnerType:      req.RunnerType,
					IsEvmChain:      req.IsEvmChain,
					Mnemonics:       mnemonics,
					CatalystVersion: req.CatalystVersion,
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
					ChainState:      chainState,
					ProviderState:   providerState,
					LoadTestSpec:    *req.CosmosLoadTestSpec,
					RunnerType:      req.RunnerType,
					IsEvmChain:      req.IsEvmChain,
					Mnemonics:       mnemonics,
					CatalystVersion: req.CatalystVersion,
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
	var providerState []byte
	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	defer func() {
		if len(providerState) != 0 {
			teardownProvider(cleanupCtx, req.RunnerType, providerState)
		}
	}()

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

	shutdownSelector := workflow.NewSelector(ctx)
	// 1. load test selector
	err = runLoadTest(ctx, req, chainState, providerState, mnemonics, shutdownSelector)
	if err != nil {
		workflow.GetLogger(ctx).Error("load test initiation failed", zap.Error(err))
	}

	waitForTestnetCompletion(ctx, req, shutdownSelector)

	shutdownSelector.Select(ctx)

	if ctx.Err() != nil && temporal.IsCanceledError(ctx.Err()) {
		workflow.GetLogger(ctx).Info("workflow was cancelled, completing gracefully")
		return nil
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
