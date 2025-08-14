package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cosmos "github.com/skip-mev/catalyst/chains/cosmos"
	types2 "github.com/skip-mev/catalyst/chains/cosmos/types"
	"github.com/skip-mev/catalyst/chains/ethereum"
	"github.com/skip-mev/catalyst/chains/ethereum/types"
	catchaintypes "github.com/skip-mev/catalyst/chains/types"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/util"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
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
	res.ProviderState = newProviderState

	if chain != nil {
		newChainState, chainErr := chain.Serialize(ctx, p)
		if chainErr != nil {
			logger.Error("failed to serialize chain after error", zap.Error(wrappedErr), zap.Error(chainErr))
			return res, fmt.Errorf("%w; also failed to serialize chain: %v", wrappedErr, chainErr)
		}
		res.ChainState = newChainState
	}

	return res, wrappedErr
}

func generateLoadTestSpec(ctx context.Context, logger *zap.Logger, chain *chain.Chain, chainID string,
	loadTestSpec catchaintypes.LoadTestSpec, mnemonics []string,
) ([]byte, error) {
	chainConfig := chain.GetConfig()

	var nodes []string
	for _, v := range chain.GetValidators() {
		ipAddr, err := v.GetIP(ctx)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, ipAddr)
	}

	var catalystChainConfig catchaintypes.ChainConfig
	switch loadTestSpec.Kind {
	case ethereum.Kind:
		nodeAddresses := make([]types.NodeAddress, 0, len(nodes))
		for _, addr := range nodes {
			nodeAddresses = append(nodeAddresses, types.NodeAddress{
				RPC:       "http://" + addr + ":8545",
				Websocket: "ws://" + addr + ":8546",
			})
		}
		catalystChainConfig = types.ChainConfig{
			NodesAddresses: nodeAddresses,
		}
	case cosmos.Kind:
		nodeAddresses := make([]types2.NodeAddress, 0, len(nodes))
		for _, addr := range nodes {
			nodeAddresses = append(nodeAddresses, types2.NodeAddress{
				GRPC: addr + ":9090",
				RPC:  "http://" + addr + ":26657",
			})
		}
		catalystChainConfig = types2.ChainConfig{
			GasDenom:       chainConfig.Denom,
			Bech32Prefix:   chainConfig.Bech32Prefix,
			NodesAddresses: nodeAddresses,
		}
	default:
		return nil, fmt.Errorf("unknown load test spec kind: %v", loadTestSpec.Kind)
	}
	loadTestSpec.ChainCfg = catalystChainConfig
	loadTestSpec.ChainID = chainID

	loadTestSpec.Mnemonics = mnemonics

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

	p, err := util.RestoreProvider(ctx, logger, req.RunnerType, req.ProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return messages.RunLoadTestResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	walletConfig := testnet.CosmosWalletConfig
	if req.IsEvmChain {
		walletConfig = testnet.EvmCosmosWalletConfig
		logger.Info("updated load test to evm walletconfig")
	}

	chain, err := chain.RestoreChain(ctx, logger, p, req.ChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, nil, err, "failed to restore chain")
	}

	configBytes, err := generateLoadTestSpec(ctx, logger, chain, chain.GetConfig().ChainId, req.LoadTestSpec, req.Mnemonics)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to generate load test config")
	}

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "catalyst",
		Image: provider.ImageDefinition{
			Image: "ghcr.io/skip-mev/catalyst:0.0.0-beta.4",
			UID:   "100",
			GID:   "100",
		},
		ProviderSpecificConfig: messages.DigitalOceanDefaultOpts[0],
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

			var result catchaintypes.LoadTestResult
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

			newChainState, err := chain.Serialize(ctx, p)
			if err != nil {
				logger.Error("failed to serialize chain after successful run", zap.Error(err))
				return messages.RunLoadTestResponse{ProviderState: newProviderState, Result: result}, fmt.Errorf("load test succeeded, but failed to serialize chain: %w", err)
			}

			return messages.RunLoadTestResponse{
				ProviderState: newProviderState,
				ChainState:    newChainState,
				Result:        result,
			}, nil
		}
	}
}
