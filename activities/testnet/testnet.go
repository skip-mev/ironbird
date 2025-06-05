package testnet

import (
	"context"
	"fmt"

	pb "github.com/skip-mev/ironbird/server/proto"

	evmhd "github.com/cosmos/evm/crypto/hd"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"

	"github.com/skip-mev/ironbird/messages"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"

	"time"

	petritypes "github.com/skip-mev/petri/core/v3/types"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
	Chains            types.Chains
	GrafanaConfig     types.GrafanaConfig
	GRPCClient        pb.IronbirdServiceClient
}

var (
	CosmosWalletConfig = petritypes.WalletConfig{
		SigningAlgorithm: "secp256k1",
		Bech32Prefix:     "cosmos",
		HDPath:           hd.CreateHDPath(118, 0, 0),
		DerivationFn:     hd.Secp256k1.Derive(),
		GenerationFn:     hd.Secp256k1.Generate(),
	}
	EvmCosmosWalletConfig = petritypes.WalletConfig{
		SigningAlgorithm: "eth_secp256k1",
		Bech32Prefix:     "cosmos",
		HDPath:           hd.CreateHDPath(60, 0, 0),
		DerivationFn:     evmhd.EthSecp256k1.Derive(),
		GenerationFn:     evmhd.EthSecp256k1.Generate(),
	}
)

const (
	cosmosDenom    = "stake"
	evmDenom       = "uatom"
	cosmosDecimals = 6
)

func (a *Activity) CreateProvider(ctx context.Context, req messages.CreateProviderRequest) (messages.CreateProviderResponse, error) {
	logger, _ := zap.NewDevelopment()

	var p provider.ProviderI
	var err error

	if req.RunnerType == messages.Docker {
		p, err = docker.CreateProvider(ctx, logger, req.Name)
	} else {
		p, err = digitalocean.NewProvider(ctx, req.Name, a.DOToken, a.TailscaleSettings,
			digitalocean.WithLogger(logger), digitalocean.WithTelemetry(a.TelemetrySettings))
	}

	if err != nil {
		return messages.CreateProviderResponse{}, err
	}

	state, err := p.SerializeProvider(ctx)

	return messages.CreateProviderResponse{ProviderState: state}, err
}

func (a *Activity) TeardownProvider(ctx context.Context, req messages.TeardownProviderRequest) (messages.TeardownProviderResponse, error) {
	logger, _ := zap.NewDevelopment()

	p, err := util.RestoreProvider(ctx, logger, req.RunnerType, req.ProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return messages.TeardownProviderResponse{}, err
	}

	err = p.Teardown(ctx)
	return messages.TeardownProviderResponse{}, err
}

func (a *Activity) updateWorkflowData(ctx context.Context, workflowID string, nodes []*pb.Node, validators []*pb.Node, chainID string, startTime time.Time, logger *zap.Logger) {
	if a.GRPCClient == nil {
		logger.Warn("GRPCClient is nil, skipping workflow data update")
		return
	}

	monitoringLinks := types.GenerateMonitoringLinks(chainID, startTime, a.GrafanaConfig)
	logger.Info("monitoring links", zap.String("chainID", chainID),
		zap.Any("monitoringLinks", monitoringLinks))

	updateReq := &pb.UpdateWorkflowDataRequest{
		WorkflowId: workflowID,
		Nodes:      nodes,
		Validators: validators,
		Monitoring: monitoringLinks,
	}

	_, err := a.GRPCClient.UpdateWorkflowData(ctx, updateReq)
	if err != nil {
		logger.Error("Failed to update workflow data", zap.Error(err))
	} else {
		logger.Info("Successfully updated workflow data")
	}
}

func (a *Activity) LaunchTestnet(ctx context.Context, req messages.LaunchTestnetRequest) (resp messages.LaunchTestnetResponse, err error) {
	logger, _ := zap.NewDevelopment()

	workflowID := activity.GetInfo(ctx).WorkflowExecution.ID
	startTime := time.Now()

	p, err := util.RestoreProvider(ctx, logger, req.RunnerType, req.ProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return
	}

	nodeOptions := petritypes.NodeOptions{}

	if req.RunnerType == messages.DigitalOcean {
		nodeOptions.NodeDefinitionModifier = func(definition provider.TaskDefinition, config petritypes.NodeConfig) provider.TaskDefinition {
			definition.ProviderSpecificConfig = req.ProviderSpecificOptions
			return definition
		}
	}

	chainConfig, walletConfig := constructChainConfig(req, a.Chains)
	logger.Info("creating chain", zap.Any("chain_config", chainConfig))
	chain, chainErr := petrichain.CreateChain(
		ctx, logger, p, chainConfig,
		petritypes.ChainOptions{
			NodeCreator: node.CreateNode,
			NodeOptions: petritypes.NodeOptions{
				NodeDefinitionModifier: func(definition provider.TaskDefinition, config petritypes.NodeConfig) provider.TaskDefinition {
					definition.ProviderSpecificConfig = req.ProviderSpecificOptions
					return definition
				},
			},
			WalletConfig: walletConfig,
		},
	)

	if chainErr != nil {
		providerState, serializeErr := p.SerializeProvider(ctx)
		if serializeErr != nil {
			return resp, temporal.NewApplicationErrorWithOptions("failed to serialize provider", serializeErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
		}

		resp.ProviderState = providerState

		return resp, temporal.NewApplicationErrorWithOptions("failed to create chain", chainErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	resp.ChainID = chainConfig.ChainId

	initErr := chain.Init(ctx, petritypes.ChainOptions{
		ModifyGenesis: petrichain.ModifyGenesis(req.GenesisModifications),
		NodeCreator:   node.CreateNode,
		WalletConfig:  walletConfig,
	})
	if initErr != nil {
		providerState, serializeErr := p.SerializeProvider(ctx)
		if serializeErr != nil {
			return resp, temporal.NewApplicationErrorWithOptions("failed to serialize provider", serializeErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
		}

		resp.ProviderState = providerState

		return resp, temporal.NewApplicationErrorWithOptions("failed to init chain", initErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	err = chain.WaitForStartup(ctx)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to wait for chain startup", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	providerState, err := p.SerializeProvider(ctx)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to serialize provider", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	resp.ProviderState = providerState

	chainState, err := chain.Serialize(ctx, p)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to serialize chain", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	resp.ChainState = chainState

	testnetValidators := make([]*pb.Node, 0, len(chain.GetValidators()))
	testnetNodes := make([]*pb.Node, 0, len(chain.GetNodes()))

	for _, validator := range chain.GetValidators() {
		validatorInfo, err := processNodeInfo(ctx, validator)
		if err != nil {
			return resp, err
		}
		testnetValidators = append(testnetValidators, validatorInfo)
	}

	for _, node := range chain.GetNodes() {
		nodeInfo, err := processNodeInfo(ctx, node)
		if err != nil {
			return resp, err
		}
		testnetNodes = append(testnetNodes, nodeInfo)
	}

	resp.Nodes = testnetNodes
	resp.Validators = testnetValidators

	if a.GRPCClient != nil {
		a.updateWorkflowData(ctx, workflowID, testnetNodes, testnetValidators, chainConfig.ChainId, startTime, logger)
	}

	go func() {
		emitHeartbeats(ctx, chain, logger)
	}()

	return resp, nil
}

func emitHeartbeats(ctx context.Context, chain *petrichain.Chain, logger *zap.Logger) {
	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-heartbeatCtx.Done():
			return
		case <-ticker.C:
			validators := chain.GetValidators()

			// attempts to get a heartbeat from up to 3 validators
			success := false
			maxValidators := 3
			if len(validators) < maxValidators {
				maxValidators = len(validators)
			}

			for i := 0; i < maxValidators; i++ {
				tmClient, err := validators[i].GetTMClient(ctx)
				if err != nil {
					logger.Error("Failed to get TM client", zap.Error(err), zap.Int("validator", i))
					continue
				}

				_, err = tmClient.Status(ctx)
				if err != nil {
					logger.Error("Chain status check failed", zap.Error(err), zap.Int("validator", i))
					continue
				}

				success = true
				break
			}

			if !success {
				logger.Error("All validator checks failed", zap.Int("validators_attempted", maxValidators))
				continue
			}

			activity.RecordHeartbeat(ctx, "Chain status: healthy")
		}
	}
}

func constructChainConfig(req messages.LaunchTestnetRequest,
	chains types.Chains) (petritypes.ChainConfig, petritypes.WalletConfig) {
	chainImage := chains[req.Image]
	fmt.Println("chain image + req", chainImage, req, chains)

	denom := cosmosDenom
	chainID := req.Name
	gasPrice := chainImage.GasPrices
	walletConfig := CosmosWalletConfig
	coinType := "118"

	if req.Evm {
		denom = evmDenom
		chainID = "cosmos_22222-1"
		gasPrice = "0.0005uatom"
		walletConfig = EvmCosmosWalletConfig
		coinType = "60"
	}

	chainConfig := petritypes.ChainConfig{
		Name:          req.Name,
		Denom:         denom,
		Decimals:      cosmosDecimals,
		NumValidators: int(req.NumOfValidators),
		NumNodes:      int(req.NumOfNodes),
		BinaryName:    chainImage.BinaryName,
		Image: provider.ImageDefinition{
			Image: req.Image,
			UID:   chainImage.UID,
			GID:   chainImage.GID,
		},
		GasPrices:            gasPrice,
		Bech32Prefix:         "cosmos",
		HomeDir:              chainImage.HomeDir,
		CoinType:             coinType,
		ChainId:              chainID,
		UseGenesisSubCommand: true,
	}

	return chainConfig, walletConfig
}

func processNodeInfo(ctx context.Context, nodeProvider petritypes.NodeI) (*pb.Node, error) {
	cosmosIp, err := nodeProvider.GetExternalAddress(ctx, "1317")
	if err != nil {
		return &pb.Node{}, err
	}

	cometIp, err := nodeProvider.GetExternalAddress(ctx, "26657")
	if err != nil {
		return &pb.Node{}, err
	}

	ip, err := nodeProvider.GetIP(ctx)
	if err != nil {
		return &pb.Node{}, err
	}

	return &pb.Node{
		Name:    nodeProvider.GetDefinition().Name,
		Rpc:     fmt.Sprintf("http://%s", cometIp),
		Lcd:     fmt.Sprintf("http://%s", cosmosIp),
		Address: ip,
	}, nil
}
