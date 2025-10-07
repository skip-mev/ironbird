package testnet

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	pb "github.com/skip-mev/ironbird/server/proto"

	evmhd "github.com/cosmos/evm/crypto/hd"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"

	"github.com/skip-mev/ironbird/messages"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"

	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	petritypes "github.com/skip-mev/ironbird/petri/core/types"
	petrichain "github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
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

// convertECRTokenToDockerAuth converts an ECR authorization token to Docker API RegistryAuth format
func convertECRTokenToDockerAuth(ecrToken string) (string, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(ecrToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode ECR token: %w", err)
	}

	parts := strings.Split(string(decodedToken), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid ECR token format")
	}

	authConfig := map[string]string{
		"username": parts[0],
		"password": parts[1],
	}

	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth config: %w", err)
	}

	return base64.StdEncoding.EncodeToString(authJSON), nil
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

	if len(req.ProviderState) == 0 {
		logger.Info("provider state is empty, skipping teardown")
		return messages.TeardownProviderResponse{}, nil
	}

	decompressedProviderState, err := util.DecompressData(req.ProviderState)
	if err != nil {
		return messages.TeardownProviderResponse{}, fmt.Errorf("failed to decompress provider state: %w", err)
	}

	p, err := util.RestoreProvider(ctx, logger, req.RunnerType, decompressedProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return messages.TeardownProviderResponse{}, err
	}

	err = p.Teardown(ctx)
	return messages.TeardownProviderResponse{}, err
}

func (a *Activity) updateWorkflowData(ctx context.Context, workflowID string, nodes []*pb.Node, validators []*pb.Node, chainID string, startTime time.Time, provider string, logger *zap.Logger) {
	if a.GRPCClient == nil {
		logger.Warn("GRPCClient is nil, skipping workflow data update")
		return
	}

	monitoringLinks := types.GenerateMonitoringLinks(chainID, startTime, nil, provider, a.GrafanaConfig)
	logger.Info("monitoring links", zap.String("chainID", chainID),
		zap.Any("monitoringLinks", monitoringLinks))

	updateReq := &pb.UpdateWorkflowDataRequest{
		WorkflowId: workflowID,
		Nodes:      nodes,
		Validators: validators,
		Monitoring: monitoringLinks,
		Provider:   provider,
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

	var dockerAuth string
	if a.AwsConfig != nil {
		token, err := util.FetchDockerRepoToken(ctx, *a.AwsConfig)
		if err != nil {
			logger.Error("Failed to fetch docker repo token", zap.Error(err))
		} else {
			dockerAuth, err = convertECRTokenToDockerAuth(token)
			if err != nil {
				logger.Error("Failed to convert ECR token to Docker auth format", zap.Error(err))
			}
		}
	}

	nodeOptions.NodeDefinitionModifier = func(definition provider.TaskDefinition, config petritypes.NodeConfig) provider.TaskDefinition {
		if definition.ProviderSpecificConfig == nil {
			definition.ProviderSpecificConfig = make(map[string]string)
		}
		if dockerAuth != "" {
			definition.ProviderSpecificConfig["docker_auth"] = dockerAuth
		}
		return definition
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

		compressedProviderState, compressErr := util.CompressData(providerState)
		if compressErr != nil {
			return resp, temporal.NewApplicationErrorWithOptions("failed to compress provider state", compressErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
		}
		resp.ProviderState = compressedProviderState

		return resp, temporal.NewApplicationErrorWithOptions("failed to create chain", chainErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	resp.ChainID = chainConfig.ChainId

	initErr := chain.Init(ctx, petritypes.ChainOptions{
		ModifyGenesis:      petrichain.ModifyGenesis(req.GenesisModifications),
		NodeCreator:        node.CreateNode,
		WalletConfig:       walletConfig,
		NodeOptions:        nodeOptions,
		BaseMnemonic:       req.BaseMnemonic,
		AdditionalAccounts: req.NumWallets,
	})
	if initErr != nil {
		providerState, serializeErr := p.SerializeProvider(ctx)
		if serializeErr != nil {
			return resp, temporal.NewApplicationErrorWithOptions("failed to serialize provider", serializeErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
		}

		compressedProviderState, compressErr := util.CompressData(providerState)
		if compressErr != nil {
			return resp, temporal.NewApplicationErrorWithOptions("failed to compress provider state", compressErr.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
		}
		resp.ProviderState = compressedProviderState

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

	compressedProviderState, err := util.CompressData(providerState)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to compress provider state", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}
	resp.ProviderState = compressedProviderState

	chainState, err := chain.Serialize(ctx, p)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to serialize chain", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	compressedChainState, err := util.CompressData(chainState)
	if err != nil {
		return resp, temporal.NewApplicationErrorWithOptions("failed to compress chain state", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}
	resp.ChainState = compressedChainState

	testnetValidators := make([]*pb.Node, 0, len(chain.GetValidators()))
	testnetNodes := make([]*pb.Node, 0, len(chain.GetNodes()))

	for _, validator := range chain.GetValidators() {
		validatorInfo, err := getNodeExternalAddresses(ctx, validator, req.IsEvmChain)
		if err != nil {
			return resp, err
		}
		testnetValidators = append(testnetValidators, validatorInfo)
	}

	for _, node := range chain.GetNodes() {
		nodeInfo, err := getNodeExternalAddresses(ctx, node, req.IsEvmChain)
		if err != nil {
			return resp, err
		}
		testnetNodes = append(testnetNodes, nodeInfo)
	}

	resp.Nodes = testnetNodes
	resp.Validators = testnetValidators

	if a.GRPCClient != nil {
		a.updateWorkflowData(ctx, workflowID, testnetNodes, testnetValidators, chainConfig.ChainId, startTime, p.GetName(), logger)
	}

	//go func() {
	//	emitHeartbeats(ctx, chain, logger)
	//}()

	return resp, nil
}

func constructChainConfig(req messages.LaunchTestnetRequest,
	chains types.Chains) (petritypes.ChainConfig, petritypes.WalletConfig) {
	chainImage := chains[req.BaseImage]

	config := petritypes.ChainConfig{
		Name:          req.Name,
		Denom:         cosmosDenom,
		Decimals:      cosmosDecimals,
		NumValidators: int(req.NumOfValidators),
		NumNodes:      int(req.NumOfNodes),
		BinaryName:    chainImage.BinaryName,
		Entrypoint:    chainImage.Entrypoint,
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
		RegionConfig:          req.RegionConfigs,
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
		config.AdditionalPorts = []string{"8545", "8546", "8100"} // geth rpc, geth ws rpc, evmd geth metrics
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

		deleg := new(big.Int)
		deleg.SetString("10000000000000000000000000", 10)
		genBal := deleg.Mul(deleg, big.NewInt(int64(req.NumOfValidators+2)))
		config.GenesisDelegation = deleg
		config.GenesisBalance = genBal
	}

	return config, walletConfig
}

func getNodeExternalAddresses(ctx context.Context, nodeProvider petritypes.NodeI, isEvmChain bool) (*pb.Node, error) {
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

	node := &pb.Node{
		Name:    nodeProvider.GetDefinition().Name,
		Rpc:     fmt.Sprintf("http://%s", cometIp),
		Lcd:     fmt.Sprintf("http://%s", lcdIp),
		Grpc:    grpcIp,
		Address: ip,
	}

	if isEvmChain {
		evmRpcIp, err := nodeProvider.GetExternalAddress(ctx, "8545")
		if err != nil {
			return &pb.Node{}, err
		}
		node.Evmrpc = fmt.Sprintf("http://%s", evmRpcIp)

		evmWsIp, err := nodeProvider.GetExternalAddress(ctx, "8546")
		if err != nil {
			return &pb.Node{}, err
		}
		node.Evmws = fmt.Sprintf("ws://%s", evmWsIp)
	}

	return node, nil
}
