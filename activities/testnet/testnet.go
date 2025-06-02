package testnet

import (
	"context"
	"fmt"
	"github.com/skip-mev/ironbird/db"

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
	ChainImages       types.ChainImages
	DatabaseService   *db.DatabaseService
	DashboardsConfig  *types.DashboardsConfig
}

var (
	CosmosWalletConfig = petritypes.WalletConfig{
		SigningAlgorithm: "secp256k1",
		Bech32Prefix:     "cosmos",
		HDPath:           hd.CreateHDPath(118, 0, 0),
		DerivationFn:     hd.Secp256k1.Derive(),
		GenerationFn:     hd.Secp256k1.Generate(),
	}
	EVMCosmosWalletConfig = petritypes.WalletConfig{
		SigningAlgorithm: "eth_secp256k1",
		Bech32Prefix:     "cosmos",
		HDPath:           hd.CreateHDPath(60, 0, 0),
		DerivationFn:     evmhd.EthSecp256k1.Derive(),
		GenerationFn:     evmhd.EthSecp256k1.Generate(),
	}
)

const (
	cosmosDenom    = "stake"
	gaiaEvmDenom   = "atest"
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

func (a *Activity) constructChainConfig(req messages.LaunchTestnetRequest) (petritypes.ChainConfig, petritypes.WalletConfig) {
	chainImage := a.ChainImages[req.Repo]

	denom := cosmosDenom
	chainID := req.Name
	gasPrice := chainImage.GasPrices
	walletConfig := CosmosWalletConfig
	coinType := "118"

	if req.GaiaEVM {
		denom = gaiaEvmDenom
		chainID = "cosmos_22222-1"
		gasPrice = "0.0005atest"
		walletConfig = EVMCosmosWalletConfig
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

func (a *Activity) processNode(ctx context.Context, nodeProvider petritypes.NodeI) (messages.Node, error) {
	cosmosIp, err := nodeProvider.GetExternalAddress(ctx, "1317")
	if err != nil {
		return messages.Node{}, err
	}

	cometIp, err := nodeProvider.GetExternalAddress(ctx, "26657")
	if err != nil {
		return messages.Node{}, err
	}

	ip, err := nodeProvider.GetIP(ctx)
	if err != nil {
		return messages.Node{}, err
	}

	return messages.Node{
		Name:    nodeProvider.GetDefinition().Name,
		RPC:     fmt.Sprintf("http://%s", cometIp),
		LCD:     fmt.Sprintf("http://%s", cosmosIp),
		Address: ip,
	}, nil
}

func (a *Activity) updateDatabase(workflowID string, nodes []messages.Node, validators []messages.Node, chainID string, startTime time.Time, logger *zap.Logger) {
	if a.DatabaseService != nil {
		if err := a.DatabaseService.UpdateWorkflowNodes(workflowID, nodes, validators); err != nil {
			logger.Error("Failed to update workflow nodes", zap.Error(err))
		}

		if a.DashboardsConfig != nil {
			monitoringLinks := a.DashboardsConfig.GenerateMonitoringLinks(chainID, startTime)
			logger.Info("monitoring links", zap.String("chainID", chainID),
				zap.Any("monitoringLinks", monitoringLinks))

			if err := a.DatabaseService.UpdateWorkflowMonitoring(workflowID, monitoringLinks); err != nil {
				logger.Error("Failed to update workflow monitoring links", zap.Error(err))
			} else {
				logger.Info("Successfully updated monitoring links in database")
			}
		} else {
			logger.Warn("DashboardsConfig is nil, skipping monitoring links generation")
		}
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

	chainConfig, walletConfig := a.constructChainConfig(req)
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

	testnetValidators := make([]messages.Node, len(chain.GetValidators()))
	testnetNodes := make([]messages.Node, len(chain.GetNodes()))

	for _, validator := range chain.GetValidators() {
		node, err := a.processNode(ctx, validator)
		if err != nil {
			return resp, err
		}
		testnetValidators = append(testnetValidators, node)
	}

	for _, node := range chain.GetNodes() {
		node, err := a.processNode(ctx, node)
		if err != nil {
			return resp, err
		}
		testnetNodes = append(testnetNodes, node)
	}

	resp.Nodes = testnetNodes
	resp.Validators = testnetValidators

	a.updateDatabase(workflowID, testnetNodes, testnetValidators, chainConfig.ChainId, startTime, logger)

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
