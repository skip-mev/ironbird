package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	petritypes "github.com/skip-mev/petri/core/v3/types"

	"github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"

	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/provider/docker"

	petriutil "github.com/skip-mev/petri/core/v3/util"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"github.com/skip-mev/petri/cosmos/v3/wallet"
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
	walletConfig petritypes.WalletConfig, loadTestSpec types.LoadTestSpec) ([]byte, error) {

	chainConfig := chain.GetConfig()
	loadTestSpec.GasDenom = chainConfig.Denom
	loadTestSpec.Bech32Prefix = chainConfig.Bech32Prefix
	loadTestSpec.ChainID = chainID

	validators := chain.GetValidators()
	var nodes []types.NodeAddress
	for _, v := range validators {
		grpcAddr, err := v.GetIP(ctx)
		grpcAddr = grpcAddr + ":9090"
		if err != nil {
			return nil, err
		}

		rpcAddr, err := v.GetIP(ctx)
		rpcAddr = rpcAddr + ":26657"
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, types.NodeAddress{
			GRPC: grpcAddr,
			RPC:  fmt.Sprintf("http://%s", rpcAddr),
		})
	}

	loadTestSpec.NodesAddresses = nodes

	var mnemonics []string
	var addresses []string
	var walletsMutex sync.Mutex

	faucetWallet := chain.GetFaucetWallet()

	totalWallets := 2500
	var wg sync.WaitGroup

	for i := 0; i < totalWallets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), walletConfig)
			if err != nil {
				logger.Error("failed to create wallet", zap.Error(err))
				return
			}

			walletsMutex.Lock()
			mnemonics = append(mnemonics, w.Mnemonic())
			addresses = append(addresses, w.FormattedAddress())
			walletsMutex.Unlock()
		}()
	}

	wg.Wait()
	logger.Info("successfully created all wallets", zap.Int("count", len(mnemonics)))

	node := validators[len(validators)-1]
	err := node.RecoverKey(ctx, "faucet", faucetWallet.Mnemonic())
	if err != nil {
		logger.Error("failed to recover faucet wallet key", zap.Error(err))
		return nil, fmt.Errorf("failed to recover faucet wallet key: %w", err)
	}
	time.Sleep(1 * time.Second)

	command := []string{
		chainConfig.BinaryName,
		"tx", "bank", "multi-send",
		faucetWallet.FormattedAddress(),
	}

	command = append(command, addresses...)
	command = append(command, fmt.Sprintf("1000000000%s", chainConfig.Denom),
		"--chain-id", chainConfig.ChainId,
		"--keyring-backend", "test",
		"--from", "faucet",
		"--fees", fmt.Sprintf("80000%s", chainConfig.Denom),
		"--gas", "auto",
		"--yes",
		"--home", chainConfig.HomeDir,
	)

	_, stderr, exitCode, err := node.RunCommand(ctx, command)
	if err != nil || exitCode != 0 {
		logger.Warn("failed to fund wallets", zap.Error(err), zap.String("stderr", stderr))
	}
	time.Sleep(5 * time.Second)
	loadTestSpec.Mnemonics = mnemonics

	err = loadTestSpec.Validate()
	if err != nil {
		logger.Error("failed to validate custom load test config", zap.Error(err), zap.Any("spec", loadTestSpec))
		return nil, fmt.Errorf("failed to validate custom load test config: %w", err)
	}

	logger.Info("Load test spec constructed", zap.Any("spec", loadTestSpec))
	return yaml.Marshal(&loadTestSpec)
}

func (a *Activity) RunLoadTest(ctx context.Context, req messages.RunLoadTestRequest) (messages.RunLoadTestResponse, error) {
	logger, _ := zap.NewDevelopment()
	logger.Info("req", zap.Any("req", req))

	var p provider.ProviderI
	var err error
	if req.RunnerType == testnettypes.Docker {
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
		return messages.RunLoadTestResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	walletConfig := testnet.CosmosWalletConfig
	if req.GaiaEVM {
		walletConfig = testnet.EVMCosmosWalletConfig
		logger.Info("updated load test to evm walletconfig")
	}

	chain, err := chain.RestoreChain(ctx, logger, p, req.ChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, nil, err, "failed to restore chain")
	}

	configBytes, err := generateLoadTestSpec(ctx, logger, chain, chain.GetConfig().ChainId, walletConfig, req.LoadTestSpec)
	if err != nil {
		return handleLoadTestError(ctx, logger, p, chain, err, "failed to generate load test config")
	}

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "catalyst",
		Image: provider.ImageDefinition{
			Image: "ghcr.io/skip-mev/catalyst:latest",
			UID:   "100",
			GID:   "100",
		},
		ProviderSpecificConfig: map[string]string{
			"region":   "ams3",
			"image_id": "185517855",
			"size":     "s-4vcpu-8gb",
		},
		Command: []string{"/tmp/catalyst/loadtest.yml"},
		DataDir: "/tmp/catalyst",
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

			var result types.LoadTestResult
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
