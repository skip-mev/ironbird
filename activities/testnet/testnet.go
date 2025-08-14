package testnet

import (
	"context"
	"fmt"
	"math/big"

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

	"github.com/aws/aws-sdk-go-v2/aws"
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
	AwsConfig         *aws.Config
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
	cosmosDenom       = "stake"
	evmDenom          = "atest"
	cosmosDecimals    = 6
	DefaultEvmChainID = "262144"
)

var launchedNodes = 0

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
		token, err := util.FetchDockerRepoToken(ctx, *a.AwsConfig)
		if err != nil {
			logger.Error("Failed to fetch docker repo token", zap.Error(err))
		}
		nodeOptions.NodeDefinitionModifier = func(definition provider.TaskDefinition, config petritypes.NodeConfig) provider.TaskDefinition {
			definition.ProviderSpecificConfig = messages.DigitalOceanDefaultOpts[launchedNodes%5]
			launchedNodes++
			definition.ProviderSpecificConfig = req.ProviderSpecificOptions
			definition.ProviderSpecificConfig["docker_auth"] = token
			return definition
		}
	}

	chainConfig, walletConfig := constructChainConfig(req, a.Chains)
	logger.Info("creating chain", zap.Any("chain_config", chainConfig))
	chain, chainErr := petrichain.CreateChain(
		ctx, logger, p, chainConfig,
		petritypes.ChainOptions{
			NodeCreator:  node.CreateNode,
			NodeOptions:  nodeOptions,
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
		NodeOptions:   nodeOptions,
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
		validatorInfo, err := getNodeExternalAddresses(ctx, validator)
		if err != nil {
			return resp, err
		}
		testnetValidators = append(testnetValidators, validatorInfo)
	}

	for _, node := range chain.GetNodes() {
		nodeInfo, err := getNodeExternalAddresses(ctx, node)
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
	chainImage := chains[req.BaseImage]
	deleg := new(big.Int)
	deleg.SetString("10000000000000000000000000", 10)
	genBal := deleg.Mul(deleg, big.NewInt(int64(req.NumOfValidators+2)))

	config := petritypes.ChainConfig{
		Name:          req.Name,
		Denom:         cosmosDenom,
		Decimals:      cosmosDecimals,
		NumValidators: int(req.NumOfValidators),
		NumNodes:      int(req.NumOfNodes),
		BinaryName:    chainImage.BinaryName,
		Image: provider.ImageDefinition{
			Image: req.Image,
			UID:   chainImage.UID,
			GID:   chainImage.GID,
		},
		GasPrices:             chainImage.GasPrices,
		Bech32Prefix:          "cosmos",
		HomeDir:               chainImage.HomeDir,
		CoinType:              "118",
		ChainId:               req.Name,
		UseGenesisSubCommand:  true,
		CustomAppConfig:       req.CustomAppConfig,
		CustomConsensusConfig: req.CustomConsensusConfig,
		CustomClientConfig:    req.CustomClientConfig,
		SetPersistentPeers:    req.SetPersistentPeers,
		SetSeedNode:           req.SetSeedNode,
		GenesisDelegation:     deleg,
		GenesisBalance:        genBal,
	}
	walletConfig := CosmosWalletConfig

	if req.IsEvmChain {
		config.Denom = evmDenom
		chainID := DefaultEvmChainID
		config.IsEVMChain = true
		config.ChainId = chainID
		config.CoinType = "60"
		config.AdditionalStartFlags = []string{
			"--json-rpc.api", "eth,net,web3,txpool,debug",
			"--json-rpc.address", "0.0.0.0:8545",
			"--json-rpc.ws-address", "0.0.0.0:8546",
			"--json-rpc.enable",
		}
		config.AdditionalPorts = []string{"8545", "8546"}
		walletConfig = EvmCosmosWalletConfig
		if config.CustomAppConfig == nil {
			config.CustomAppConfig = make(map[string]interface{})
		}
		if config.CustomAppConfig["evm"] == nil {
			config.CustomAppConfig["evm"] = make(map[string]interface{})
		}
		if evmConfig, ok := config.CustomAppConfig["evm"].(map[string]interface{}); ok {
			evmConfig["evm-chain-id"] = chainID
		}
	}

	return config, walletConfig
}

func getNodeExternalAddresses(ctx context.Context, nodeProvider petritypes.NodeI) (*pb.Node, error) {
	lcdIp, err := nodeProvider.GetExternalAddress(ctx, "1317")
	if err != nil {
		return &pb.Node{}, err
	}

	cometIp, err := nodeProvider.GetExternalAddress(ctx, "26657")
	if err != nil {
		return &pb.Node{}, err
	}

	grpcIp, err := nodeProvider.GetExternalAddress(ctx, "9090")
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
		Lcd:     fmt.Sprintf("http://%s", lcdIp),
		Grpc:    grpcIp,
		Address: ip,
	}, nil
}
