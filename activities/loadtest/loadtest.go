package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ctltcosmos "github.com/skip-mev/catalyst/chains/cosmos/types"
	ctlteth "github.com/skip-mev/catalyst/chains/ethereum/types"
	ctltypes "github.com/skip-mev/catalyst/chains/types"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/petri/core/types"
	"github.com/skip-mev/ironbird/util"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
}

func handleLoadTestError(ctx context.Context, logger *zap.Logger, p provider.ProviderI, chain *chain.Chain, originalErr error, errMsg string) (messages.RunLoadTestResponse, error) {
	res := messages.RunLoadTestResponse{}
	wrappedErr := fmt.Errorf("%s: %w", errMsg, originalErr)

	newProviderState, serializeErr := p.SerializeProvider(ctx)
	if serializeErr != nil {
		logger.Error("failed to serialize provider after error", zap.Error(wrappedErr), zap.Error(serializeErr))
		return res, fmt.Errorf("%w; also failed to serialize provider: %v", wrappedErr, serializeErr)
	}

	compressedProviderState, compressErr := util.CompressData(newProviderState)
	if compressErr != nil {
		logger.Error("failed to compress provider state after error", zap.Error(wrappedErr), zap.Error(compressErr))
		return res, fmt.Errorf("%w; also failed to compress provider state: %v", wrappedErr, compressErr)
	}
	res.ProviderState = compressedProviderState

	if chain != nil {
		newChainState, chainErr := chain.Serialize(ctx, p)
		if chainErr != nil {
			logger.Error("failed to serialize chain after error", zap.Error(wrappedErr), zap.Error(chainErr))
			return res, fmt.Errorf("%w; also failed to serialize chain: %v", wrappedErr, chainErr)
		}

		compressedChainState, chainCompressErr := util.CompressData(newChainState)
		if chainCompressErr != nil {
			logger.Error("failed to compress chain state after error", zap.Error(wrappedErr), zap.Error(chainCompressErr))
			return res, fmt.Errorf("%w; also failed to compress chain state: %v", wrappedErr, chainCompressErr)
		}
		res.ChainState = compressedChainState
	}

	return res, wrappedErr
}

type PetriChain interface {
	GetConfig() types.ChainConfig
	GetValidators() []types.NodeI
	GetNodes() []types.NodeI
}

// TODO: this function is kind of confusing because we give it a half-baked loadtest spec, and then fill in the rest.
// I'm not sure if there is a better/more clearer way to go about this, but it seems confusing as is.
func generateLoadTestSpec(ctx context.Context, logger *zap.Logger, chain PetriChain, chainID string,
	loadTestSpec ctltypes.LoadTestSpec, baseMnemonic string, numWallets int,
) ([]byte, error) {
	chainConfig := chain.GetConfig()

	var nodes []string
	for _, v := range chain.GetNodes() {
		ipAddr, err := v.GetIP(ctx)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, ipAddr)
	}

	// If no nodes are available, add validators to nodeAddresses so load test can still run
	if len(nodes) == 0 {
		for _, v := range chain.GetValidators() {
			ipAddr, err := v.GetIP(ctx)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, ipAddr)
		}
	}

	var catalystChainConfig ctltypes.ChainConfig
	switch loadTestSpec.Kind {
	case "eth":
		ethChainCfg := ctlteth.ChainConfig{}
		switch cfg := loadTestSpec.ChainCfg.(type) {
		case ctlteth.ChainConfig:
			ethChainCfg = cfg
		case *ctlteth.ChainConfig:
			ethChainCfg = *cfg
		default:
			ethChainCfg = ctlteth.ChainConfig{}
		}
		nodeAddresses := make([]ctlteth.NodeAddress, 0, len(nodes))
		for _, addr := range nodes {
			nodeAddresses = append(nodeAddresses, ctlteth.NodeAddress{
				RPC:       "http://" + addr + ":8545",
				Websocket: "ws://" + addr + ":8546",
			})
		}
		ethChainCfg.NodesAddresses = nodeAddresses
		catalystChainConfig = ethChainCfg
	case "cosmos":
		nodeAddresses := make([]ctltcosmos.NodeAddress, 0, len(nodes))
		for _, addr := range nodes {
			nodeAddresses = append(nodeAddresses, ctltcosmos.NodeAddress{
				GRPC: addr + ":9090",
				RPC:  "http://" + addr + ":26657",
			})
		}
		catalystChainConfig = ctltcosmos.ChainConfig{
			GasDenom:       chainConfig.Denom,
			Bech32Prefix:   chainConfig.Bech32Prefix,
			NodesAddresses: nodeAddresses,
		}
	default:
		return nil, fmt.Errorf("unknown load test spec kind: %v", loadTestSpec.Kind)
	}
	loadTestSpec.ChainCfg = catalystChainConfig
	loadTestSpec.ChainID = chainID

	loadTestSpec.BaseMnemonic = baseMnemonic
	if loadTestSpec.NumWallets == 0 {
		loadTestSpec.NumWallets = numWallets
	}

	err := loadTestSpec.Validate()
	if err != nil {
		logger.Error("failed to validate custom load test config", zap.Error(err), zap.Any("spec", loadTestSpec))
		return nil, fmt.Errorf("failed to validate custom load test config: %w", err)
	}

	logger.Info("Load test spec constructed", zap.Any("spec", loadTestSpec))
	return yaml.Marshal(&loadTestSpec)
}

func (a *Activity) RunLoadTest(ctx context.Context, req messages.RunLoadTestRequest) (messages.RunLoadTestResponse, error) {
	logger, _ := zap.NewDevelopment()

	decompressedProviderState, err := util.DecompressData(req.ProviderState)
	if err != nil {
		return messages.RunLoadTestResponse{}, fmt.Errorf("failed to decompress provider state: %w", err)
	}

	p, err := util.RestoreProvider(ctx, logger, req.RunnerType, decompressedProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return messages.RunLoadTestResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	decompressedChainState, err := util.DecompressData(req.ChainState)
	if err != nil {
		return messages.RunLoadTestResponse{}, fmt.Errorf("failed to decompress chain state: %w", err)
	}

	walletConfig := testnet.CosmosWalletConfig
	if req.IsEvmChain {
		walletConfig = testnet.EvmCosmosWalletConfig
		logger.Info("updated load test to evm walletconfig")
	}

	chain, err := chain.RestoreChain(ctx, logger, p, decompressedChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, nil, err, "failed to restore chain")
	}

	configBytes, err := generateLoadTestSpec(ctx, logger, chain, chain.GetConfig().ChainId, req.LoadTestSpec, req.BaseMnemonic, req.NumWallets)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to generate load test config")
	}

	catalystImage := "ghcr.io/skip-mev/catalyst"
	if req.CatalystVersion != "" {
		catalystImage = fmt.Sprintf("ghcr.io/skip-mev/catalyst:%s", req.CatalystVersion)
	}
	logger.Info("using catalyst image", zap.String("image", catalystImage))

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "catalyst",
		Image: provider.ImageDefinition{
			Image: catalystImage,
			UID:   "100",
			GID:   "100",
		},
		ProviderSpecificConfig: messages.DigitalOceanDefaultOpts,
		Command:                []string{"/tmp/catalyst/loadtest.yml"},
		DataDir:                "/tmp/catalyst",
		Environment: map[string]string{
			"DEV_LOGGING": "true",
		},
	})
	if err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to create task")
	}

	if err := task.WriteFile(ctx, "loadtest.yml", configBytes); err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to write config file to task")
	}

	logger.Info("starting load test")
	if err := task.Start(ctx); err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to start task")
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Warn("context cancelled during load test execution")
			return handleLoadTestError(ctx, logger, p, chain, ctx.Err(), "context cancelled")
		case <-ticker.C:
			status, err := task.GetStatus(ctx)
			if err != nil {
				continue
			}

			if status != provider.TASK_STOPPED {
				continue
			}

			logger.Info("load test task finished, reading results")
			resultBytes, err := task.ReadFile(ctx, "load_test.json")
			if err != nil {
				return handleLoadTestError(ctx, logger, p, chain, err, "failed to read result file")
			}

			var result ctltypes.LoadTestResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				return handleLoadTestError(ctx, logger, p, chain, err, "failed to parse result file")
			}
			logger.Info("load test completed successfully", zap.Any("result", result))

			if err := task.Destroy(ctx); err != nil {
				logger.Error("failed to destroy task after successful completion", zap.Error(err))
			}

			newProviderState, err := p.SerializeProvider(ctx)
			if err != nil {
				logger.Error("failed to serialize provider after successful run", zap.Error(err))
				return messages.RunLoadTestResponse{Result: result}, fmt.Errorf("load test succeeded, but failed to serialize provider: %w", err)
			}

			compressedProviderState, err := util.CompressData(newProviderState)
			if err != nil {
				logger.Error("failed to compress provider state after successful run", zap.Error(err))
				return messages.RunLoadTestResponse{Result: result}, fmt.Errorf("load test succeeded, but failed to compress provider state: %w", err)
			}

			newChainState, err := chain.Serialize(ctx, p)
			if err != nil {
				logger.Error("failed to serialize chain after successful run", zap.Error(err))
				return messages.RunLoadTestResponse{ProviderState: compressedProviderState, Result: result}, fmt.Errorf("load test succeeded, but failed to serialize chain: %w", err)
			}

			compressedChainState, err := util.CompressData(newChainState)
			if err != nil {
				logger.Error("failed to compress chain state after successful run", zap.Error(err))
				return messages.RunLoadTestResponse{ProviderState: compressedProviderState, Result: result}, fmt.Errorf("load test succeeded, but failed to compress chain state: %w", err)
			}

			return messages.RunLoadTestResponse{
				ProviderState: compressedProviderState,
				ChainState:    compressedChainState,
				Result:        result,
			}, nil
		}
	}
}
