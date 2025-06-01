package testnet

import (
	"context"
	"fmt"

	evmhd "github.com/cosmos/evm/crypto/hd"
	"github.com/skip-mev/ironbird/database"
	types2 "github.com/skip-mev/ironbird/types"

	"github.com/skip-mev/ironbird/messages"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"

	"time"

	"github.com/skip-mev/petri/core/v3/types"
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
	ChainImages       types2.ChainImages
	DatabaseService   *database.DatabaseService
	DashboardsConfig  *types2.DashboardsConfig
}

var (
	CosmosWalletConfig = types.WalletConfig{
		SigningAlgorithm: "secp256k1",
		Bech32Prefix:     "cosmos",
		HDPath:           hd.CreateHDPath(118, 0, 0),
		DerivationFn:     hd.Secp256k1.Derive(),
		GenerationFn:     hd.Secp256k1.Generate(),
	}
	EVMCosmosWalletConfig = types.WalletConfig{
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

	if req.RunnerType == testnet.Docker {
		p, err = docker.CreateProvider(
			ctx,
			logger,
			req.Name,
		)
	} else {
		p, err = digitalocean.NewProvider(
			ctx,
			req.Name,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
			digitalocean.WithTelemetry(a.TelemetrySettings),
		)
	}

	if err != nil {
		return messages.CreateProviderResponse{}, err
	}

	state, err := p.SerializeProvider(ctx)

	return messages.CreateProviderResponse{ProviderState: state}, err
}

func (a *Activity) TeardownProvider(ctx context.Context, req messages.TeardownProviderRequest) (messages.TeardownProviderResponse, error) {
	logger, _ := zap.NewDevelopment()

	var p provider.ProviderI
	var err error

	if req.RunnerType == testnet.Docker {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			req.ProviderState,
		)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			req.ProviderState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
			digitalocean.WithTelemetry(a.TelemetrySettings),
		)
	}

	if err != nil {
		return messages.TeardownProviderResponse{}, err
	}

	err = p.Teardown(ctx)
	return messages.TeardownProviderResponse{}, err
}

func (a *Activity) LaunchTestnet(ctx context.Context, req messages.LaunchTestnetRequest) (resp messages.LaunchTestnetResponse, err error) {
	logger, _ := zap.NewDevelopment()

	workflowID := activity.GetInfo(ctx).WorkflowExecution.ID
	startTime := time.Now()

	var p provider.ProviderI

	if req.RunnerType == testnet.Docker {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			req.ProviderState)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			req.ProviderState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
			digitalocean.WithTelemetry(a.TelemetrySettings),
		)
	}

	if err != nil {
		return
	}

	nodeOptions := types.NodeOptions{}

	if req.RunnerType == testnet.DigitalOcean {
		nodeOptions.NodeDefinitionModifier = func(definition provider.TaskDefinition, config types.NodeConfig) provider.TaskDefinition {
			definition.ProviderSpecificConfig = req.ProviderSpecificOptions
			return definition
		}
	}

	chainImage := a.ChainImages[req.Repo]
	logger.Info("chainImage", zap.Any("chainImage", chainImage), zap.Any("req", req))

	denom := cosmosDenom
	chainID := req.Name
	gasPrice := chainImage.GasPrices
	walletConfig := CosmosWalletConfig

	if req.GaiaEVM {
		denom = gaiaEvmDenom
		chainID = "cosmos_22222-1"
		gasPrice = "0.0005atest"
		walletConfig = EVMCosmosWalletConfig
	}
	logger.Info("chainid debuglog", zap.Any("", chainID))

	chain, chainErr := petrichain.CreateChain(
		ctx,
		logger,
		p,
		types.ChainConfig{
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
			CoinType:             "60",
			ChainId:              chainID,
			UseGenesisSubCommand: true,
		},
		types.ChainOptions{
			NodeCreator: node.CreateNode,
			NodeOptions: types.NodeOptions{
				NodeDefinitionModifier: func(definition provider.TaskDefinition, config types.NodeConfig) provider.TaskDefinition {
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

	resp.ChainID = chainID

	initErr := chain.Init(ctx, types.ChainOptions{
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

	var testnetValidators []testnet.Node
	var testnetNodes []testnet.Node

	for _, validator := range chain.GetValidators() {
		cosmosIp, err := validator.GetExternalAddress(ctx, "1317")
		if err != nil {
			return resp, err
		}

		cometIp, err := validator.GetExternalAddress(ctx, "26657")
		if err != nil {
			return resp, err
		}

		ip, err := validator.GetIP(ctx)
		if err != nil {
			return resp, err
		}

		testnetValidators = append(testnetValidators, testnet.Node{
			Name:    validator.GetDefinition().Name,
			Rpc:     fmt.Sprintf("http://%s", cometIp),
			Lcd:     fmt.Sprintf("http://%s", cosmosIp),
			Address: ip,
		})
	}

	for _, node := range chain.GetNodes() {
		cosmosIp, err := node.GetExternalAddress(ctx, "1317")
		if err != nil {
			return resp, err
		}

		cometIp, err := node.GetExternalAddress(ctx, "26657")
		if err != nil {
			return resp, err
		}

		ip, err := node.GetIP(ctx)
		if err != nil {
			return resp, err
		}

		testnetNodes = append(testnetNodes, testnet.Node{
			Name:    node.GetDefinition().Name,
			Rpc:     fmt.Sprintf("http://%s", cometIp),
			Lcd:     fmt.Sprintf("http://%s", cosmosIp),
			Address: ip,
		})
	}

	resp.Nodes = testnetNodes

	resp.Validators = testnetValidators

	if a.DatabaseService != nil {
		if err := a.DatabaseService.UpdateWorkflowNodes(workflowID, testnetNodes, testnetValidators); err != nil {
			logger.Error("Failed to update workflow nodes", zap.Error(err))
		}

		if a.DashboardsConfig != nil {
			monitoringLinks := a.DashboardsConfig.GenerateMonitoringLinks(chainID, startTime)
			logger.Info("Generated monitoring links",
				zap.String("chainID", chainID),
				zap.Time("startTime", startTime),
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
