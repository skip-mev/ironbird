package testnet

import (
	"fmt"
	"time"

	pb "github.com/skip-mev/ironbird/server/proto"

	"github.com/skip-mev/ironbird/petri/core/apps"
	"github.com/skip-mev/ironbird/petri/core/util"

	"github.com/skip-mev/ironbird/activities/loadbalancer"
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
		// Use a very long timer so that cancellation unblocks the selector.
		// Timer futures are canceled when the workflow context is canceled,
		// which allows shutdownSelector.Select(ctx) to return deterministically
		f := workflow.NewTimer(ctx, time.Hour*24*365*10)
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
		errOpts := temporal.ApplicationErrorOptions{
			Cause:        err,
			NonRetryable: true,
		}

		return "", temporal.NewApplicationErrorWithOptions("invalid workflow options", "error_validation", errOpts)
	}

	var (
		workflowInfo = workflow.GetInfo(ctx)
		logger       = workflow.GetLogger(ctx)

		// todo: this is not deterministic according to Temporal logic - replace.
		runName = fmt.Sprintf("ib-%s", util.RandomString(6))
		runID   = workflowInfo.WorkflowExecution.RunID
	)

	logger.Info(
		"Workflow info",
		zap.String("run_id", runID),
		zap.String("run_name", runName),
		zap.Any("request", req),
	)

	ctx = workflow.WithActivityOptions(ctx, defaultWorkflowOptions)

	buildDockerReq := messages.BuildDockerImageRequest{
		Repo:         req.Repo,
		SHA:          req.SHA,
		CosmosSdkSha: req.CosmosSdkSha,
		CometBFTSha:  req.CometBFTSha,
		ImageConfig: messages.ImageConfig{
			Name:    req.ChainConfig.Name,
			Image:   req.ChainConfig.Image,
			Version: req.ChainConfig.Version,
		},
	}

	var buildDockerRes messages.BuildDockerImageResponse

	// exec blocking
	err := workflow.ExecuteActivity(ctx, builderActivities.BuildDockerImage, buildDockerReq).Get(ctx, &buildDockerRes)
	if err != nil {
		return "", err
	}

	err = startWorkflow(ctx, req, runName, buildDockerRes)

	switch {
	case temporal.IsCanceledError(err):
		logger.Info("testnet workflow was cancelled, completing gracefully")
		return "", nil
	case err != nil:
		logger.Error("testnet workflow failed", zap.Error(err))
		return "", err
	default:
		logger.Info("testnet workflow completed successfully")
		return "", nil
	}
}

type testnetResult struct {
	ChainState    []byte
	ProviderState []byte
	Nodes         []*pb.Node
	Validators    []*pb.Node
}

func launchTestnet(
	ctx workflow.Context,
	req messages.TestnetWorkflowRequest,
	runName string,
	buildResult messages.BuildDockerImageResponse,
) (*testnetResult, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("launching testnet", zap.Any("req", req))

	var (
		providerState      []byte
		createProviderResp messages.CreateProviderResponse
		createProviderReq  = messages.CreateProviderRequest{
			RunnerType: req.RunnerType,
			Name:       runName,
		}
	)

	err := workflow.ExecuteActivity(
		ctx,
		testnetActivities.CreateProvider,
		createProviderReq,
	).Get(ctx, &createProviderResp)

	if err != nil {
		return nil, err
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

	launchTestnetReq := messages.LaunchTestnetRequest{
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
		NumWallets:            req.NumWallets,
		BaseMnemonic:          req.BaseMnemonic,
	}

	err = workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, activityOptions),
		testnetActivities.LaunchTestnet,
		launchTestnetReq,
	).Get(ctx, &testnetResp)

	if err != nil {
		compressedProviderState, compressErr := ironbirdutil.CompressData(providerState)
		if compressErr != nil {
			logger.Error("failed to compress provider state for cleanup", zap.Error(compressErr))
			return &testnetResult{ProviderState: providerState}, err
		}

		return &testnetResult{ChainState: compressedProviderState}, err
	}

	return &testnetResult{
		ChainState:    testnetResp.ChainState,
		ProviderState: testnetResp.ProviderState,
		Nodes:         testnetResp.Nodes,
		Validators:    testnetResp.Validators,
	}, nil
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
			IsEvmChain:    req.IsEvmChain,
		},
	).Get(ctx, &loadBalancerResp); err != nil {
		logger.Error("Failed to launch loadbalancer", zap.Error(err))
		return providerState, err
	}

	return loadBalancerResp.ProviderState, nil
}

func runLoadTest(ctx workflow.Context, req messages.TestnetWorkflowRequest, chainState, providerState []byte,
	selector workflow.Selector) (workflow.Future, error) {
	if req.EthereumLoadTestSpec != nil {
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
				BaseMnemonic:    req.BaseMnemonic,
				NumWallets:      req.NumWallets,
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

		return f, nil
	} else if req.CosmosLoadTestSpec != nil {
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
				BaseMnemonic:    req.BaseMnemonic,
				NumWallets:      req.NumWallets,
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

		return f, nil
	}

	return nil, nil
}

func startWorkflow(
	ctx workflow.Context,
	req messages.TestnetWorkflowRequest,
	runName string,
	buildResult messages.BuildDockerImageResponse,
) error {
	logger := workflow.GetLogger(ctx)

	testnet, err := launchTestnet(ctx, req, runName, buildResult)

	// we might cleanup even if err != nil
	if testnet != nil && len(testnet.ProviderState) > 0 {
		cleanupCtx, cancelCleanupCtx := workflow.NewDisconnectedContext(ctx)
		defer func() {
			if len(testnet.ProviderState) > 0 {
				teardownProvider(cleanupCtx, req.RunnerType, testnet.ProviderState)
			}
			cancelCleanupCtx()
		}()
	}

	if err != nil {
		return err
	}

	if req.LaunchLoadBalancer {
		alteredState, err := launchLoadBalancer(ctx, req, testnet.ProviderState, testnet.Nodes, testnet.Validators)
		if err != nil {
			return err
		}

		testnet.ProviderState = alteredState
	}

	shutdownSelector := workflow.NewSelector(ctx)

	// 1. load test selector
	loadTestFuture, err := runLoadTest(
		ctx,
		req,
		testnet.ChainState,
		testnet.ProviderState,
		shutdownSelector,
	)
	if err != nil {
		logger.Error("load test initiation failed", zap.Error(err))
	}

	waitForTestnetCompletion(ctx, req, shutdownSelector)

	shutdownSelector.Select(ctx)

	// If we have a loadtest running and the duration timer expired (not cancelled),
	// wait for the loadtest to complete before allowing teardown
	if loadTestFuture != nil && !temporal.IsCanceledError(ctx.Err()) {
		logger.Info("testnet duration expired but loadtest is still running, waiting for completion")
		err = loadTestFuture.Get(ctx, nil) // Wait for loadtest to complete
		if err != nil {
			logger.Error("failed to wait for load test future", zap.Error(err))
			return err
		}

		logger.Info("loadtest completed, proceeding with teardown")
	}

	if ctx.Err() != nil && temporal.IsCanceledError(ctx.Err()) {
		logger.Info("workflow was cancelled, completing gracefully")
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
