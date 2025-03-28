package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/provider/docker"

	"github.com/skip-mev/petri/core/v3/types"
	petriutil "github.com/skip-mev/petri/core/v3/util"

	"github.com/skip-mev/ironbird/activities/testnet"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type MsgType string

// LoadTestResult represents the results of a load test
type LoadTestResult struct {
	Overall   OverallStats
	ByMessage map[MsgType]MessageStats
	ByNode    map[string]NodeStats
	ByBlock   []BlockStat
	Error     string `json:"error,omitempty"`
}

// OverallStats represents the overall statistics of the load test
type OverallStats struct {
	TotalTransactions      int
	SuccessfulTransactions int
	FailedTransactions     int
	AvgGasPerTransaction   int64
	AvgBlockGasUtilization float64
	Runtime                time.Duration
	StartTime              time.Time
	EndTime                time.Time
	BlocksProcessed        int
}

// MessageStats represents statistics for a specific message type
type MessageStats struct {
	Transactions TransactionStats
	Gas          GasStats
	Errors       ErrorStats
}

// TransactionStats represents transaction-related statistics
type TransactionStats struct {
	Total      int
	Successful int
	Failed     int
}

// GasStats represents gas-related statistics
type GasStats struct {
	Average int64
	Min     int64
	Max     int64
	Total   int64
}

// ErrorStats represents error-related statistics
type ErrorStats struct {
	BroadcastErrors []BroadcastError
	ErrorCounts     map[string]int // Error type to count
}

// NodeStats represents statistics for a specific node
type NodeStats struct {
	Address          string
	TransactionStats TransactionStats
	MessageCounts    map[MsgType]int
	GasStats         GasStats
}

// BlockStat represents statistics for a specific block
type BlockStat struct {
	BlockHeight    int64
	Timestamp      time.Time
	GasLimit       int
	TotalGasUsed   int64
	MessageStats   map[MsgType]MessageBlockStats
	GasUtilization float64
}

// MessageBlockStats represents message-specific statistics within a block
type MessageBlockStats struct {
	TransactionsSent int
	SuccessfulTxs    int
	FailedTxs        int
	GasUsed          int64
}

// BroadcastError represents errors during broadcasting transactions
type BroadcastError struct {
	BlockHeight int64   // Block height where the error occurred (0 indicates tx did not make it to a block)
	TxHash      string  // Hash of the transaction that failed
	Error       string  // Error message
	MsgType     MsgType // Type of message that failed
	NodeAddress string  // Address of the node that returned the error
}

type PackagedState struct {
	ProviderState []byte
	ChainState    []byte
	Result        LoadTestResult
}

type LoadTestConfig struct {
	ChainID             string    `yaml:"chain_id"`
	BlockGasLimitTarget float64   `yaml:"block_gas_limit_target"`
	NumOfBlocks         int       `yaml:"num_of_blocks"`
	NodesAddresses      []Node    `yaml:"nodes_addresses"`
	Mnemonics           []string  `yaml:"mnemonics"`
	GasDenom            string    `yaml:"gas_denom"`
	Bech32Prefix        string    `yaml:"bech32_prefix"`
	Msgs                []Message `yaml:"msgs"`
}

type Node struct {
	GRPC string `yaml:"grpc"`
	RPC  string `yaml:"rpc"`
}

type Message struct {
	Type   string  `yaml:"type"`
	Weight float64 `yaml:"weight"`
}

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
}

func generateLoadTestConfig(ctx context.Context, logger *zap.Logger, chain *chain.Chain, chainID string, loadTestConfig *LoadTestConfig) ([]byte, error) {
	validators := chain.GetValidators()
	var nodes []Node
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

		nodes = append(nodes, Node{
			GRPC: grpcAddr,
			RPC:  fmt.Sprintf("http://%s", rpcAddr),
		})
	}

	var mnemonics []string
	var wallets []types.WalletI
	var walletsMutex sync.Mutex
	var wg sync.WaitGroup

	faucetWallet := chain.GetFaucetWallet()

	numberOfCustomWallets := 75
	for i := 0; i < numberOfCustomWallets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := chain.CreateWallet(ctx, petriutil.RandomString(5), testnet.CosmosWalletConfig)
			if err != nil {
				logger.Error("failed to create wallet", zap.Error(err))
				return
			}
			logger.Debug("load test wallet created", zap.String("address", w.FormattedAddress()))

			walletsMutex.Lock()
			wallets = append(wallets, w)
			walletsMutex.Unlock()
		}()
	}

	wg.Wait()

	logger.Info("successfully created wallets ", zap.Int("count", len(wallets)))

	node := validators[len(validators)-1]
	err := node.RecoverKey(ctx, "faucet", faucetWallet.Mnemonic())
	if err != nil {
		logger.Fatal("failed to recover faucet wallet key", zap.Error(err))
	}
	time.Sleep(1 * time.Second)

	chainConfig := chain.GetConfig()
	for _, w := range wallets {
		command := []string{
			chain.GetConfig().BinaryName,
			"tx", "bank", "send",
			faucetWallet.FormattedAddress(),
			w.FormattedAddress(),
			"1000000000stake",
			"--chain-id", chainConfig.ChainId,
			"--keyring-backend", "test",
			"--fees", "400000stake",
			"--yes",
			"--home", chainConfig.HomeDir,
		}

		_, stderr, exitCode, err := node.RunCommand(ctx, command)
		if err != nil || exitCode != 0 {
			logger.Warn("failed to fund wallet", zap.Error(err), zap.String("stderr", stderr))
		}

		logger.Debug("load test wallet funded", zap.String("address", w.FormattedAddress()))
		mnemonics = append(mnemonics, w.Mnemonic())
		time.Sleep(5 * time.Second)
	}

	var msgs []Message
	for _, msg := range loadTestConfig.Msgs {
		msgs = append(msgs, Message{
			Type:   msg.Type,
			Weight: msg.Weight,
		})
	}

	config := LoadTestConfig{
		ChainID:             chainID,
		BlockGasLimitTarget: loadTestConfig.BlockGasLimitTarget,
		NumOfBlocks:         loadTestConfig.NumOfBlocks,
		NodesAddresses:      nodes,
		Mnemonics:           mnemonics,
		GasDenom:            chain.GetConfig().Denom,
		Bech32Prefix:        chain.GetConfig().Bech32Prefix,
		Msgs:                msgs,
	}
	logger.Info("Load test config constructed", zap.Any("config", config))

	return yaml.Marshal(&config)
}

func (a *Activity) RunLoadTest(ctx context.Context, chainState []byte,
	loadTestConfig *LoadTestConfig, runnerType string, providerState []byte) (PackagedState, error) {
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

	configBytes, err := generateLoadTestConfig(ctx, logger, chain, chain.GetConfig().ChainId, loadTestConfig)
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
			"size":     "s-1vcpu-1gb",
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

			var result LoadTestResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				return PackagedState{}, fmt.Errorf("failed to parse result file: %w", err)
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
