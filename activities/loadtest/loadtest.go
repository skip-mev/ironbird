package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/skip-mev/catalyst/pkg/types"
	"sync"
	"time"

	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/provider/docker"

	"github.com/skip-mev/ironbird/activities/testnet"
	petriutil "github.com/skip-mev/petri/core/v3/util"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"github.com/skip-mev/petri/cosmos/v3/wallet"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type PackagedState struct {
	ProviderState []byte
	ChainState    []byte
	Result        types.LoadTestResult
}

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
}

func generateLoadTestSpec(ctx context.Context, logger *zap.Logger, chain *chain.Chain, chainID string,
	loadTestSpec *types.LoadTestSpec) ([]byte, error) {
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
			w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), testnet.CosmosWalletConfig)
			if err != nil {
				logger.Error("failed to create wallet", zap.Error(err))
				return
			}
			logger.Debug("load test wallet created", zap.String("address", w.FormattedAddress()))

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
		logger.Fatal("failed to recover faucet wallet key", zap.Error(err))
	}
	time.Sleep(1 * time.Second)

	command := []string{
		chainConfig.BinaryName,
		"tx", "bank", "multi-send",
		faucetWallet.FormattedAddress(),
	}

	command = append(command, addresses...)
	command = append(command, "1000000000stake",
		"--chain-id", chainConfig.ChainId,
		"--keyring-backend", "test",
		"--fees", "80000stake",
		"--gas", "auto",
		"--yes",
		"--home", chainConfig.HomeDir,
	)

	_, stderr, exitCode, err := node.RunCommand(ctx, command)
	if err != nil || exitCode != 0 {
		logger.Warn("failed to fund wallet", zap.Error(err), zap.String("stderr", stderr))
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

func (a *Activity) RunLoadTest(ctx context.Context, chainState []byte,
	loadTestSpec *types.LoadTestSpec, runnerType string, providerState []byte) (PackagedState, error) {
	logger, _ := zap.NewDevelopment()

	var p provider.ProviderI
	var err error
	if runnerType == string(testnettypes.Docker) {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			providerState,
		)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			providerState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
		)
	}

	if err != nil {
		return PackagedState{}, err
	}

	chain, err := chain.RestoreChain(ctx, logger, p, chainState, node.RestoreNode, testnet.CosmosWalletConfig)
	if err != nil {
		return PackagedState{}, err
	}

	configBytes, err := generateLoadTestSpec(ctx, logger, chain, chain.GetConfig().ChainId, loadTestSpec)
	if err != nil {
		return PackagedState{}, err
	}

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name:          "catalyst",
		ContainerName: "catalyst",
		Image: provider.ImageDefinition{
			Image: "ghcr.io/skip-mev/catalyst:latest",
			UID:   "100",
			GID:   "100",
		},
		ProviderSpecificConfig: map[string]string{
			"region":   "ams3",
			"image_id": "177032231",
			"size":     "s-4vcpu-8gb",
		},
		Command: []string{"/tmp/catalyst/loadtest.yml"},
		DataDir: "/tmp/catalyst",
		Environment: map[string]string{
			"DEV_LOGGING": "true",
		},
	})

	if err != nil {
		return PackagedState{}, err
	}

	if err := task.WriteFile(ctx, "loadtest.yml", configBytes); err != nil {
		return PackagedState{}, fmt.Errorf("failed to write config file to task: %w", err)
	}

	logger.Info("starting load test")
	if err := task.Start(ctx); err != nil {
		return PackagedState{}, err
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return PackagedState{}, ctx.Err()
		case <-ticker.C:
			status, err := task.GetStatus(ctx)
			if err != nil {
				continue
			}

			if status != provider.TASK_STOPPED {
				continue
			}

			resultBytes, err := task.ReadFile(ctx, "load_test.json")
			if err != nil {
				return PackagedState{}, fmt.Errorf("failed to read result file: %w", err)
			}

			var result types.LoadTestResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				return PackagedState{}, fmt.Errorf("failed to parse result file: %w", err)
			}
			logger.Info("load test result", zap.Any("result", result))

			if err := task.Destroy(ctx); err != nil {
				return PackagedState{}, fmt.Errorf("failed to destroy task: %w", err)
			}

			newProviderState, err := p.SerializeProvider(ctx)
			if err != nil {
				return PackagedState{}, fmt.Errorf("failed to serialize provider: %w", err)
			}

			newChainState, err := chain.Serialize(ctx, p)
			if err != nil {
				return PackagedState{}, fmt.Errorf("failed to serialize chain: %w", err)
			}

			return PackagedState{
				ProviderState: newProviderState,
				ChainState:    newChainState,
				Result:        result,
			}, nil
		}
	}
}
