package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/skip-mev/ironbird/activities/testnet"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	LoadTestConfigFileName = "loadtest.yml"
	CatalystResultDir      = "/tmp/catalyst"
	LoadTestResultPrefix   = "load_test_"
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
	Runtime             string    `yaml:"runtime"`
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
	Type   int     `yaml:"type"`
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
		grpcAddr, err := v.GetExternalAddress(ctx, "9090")
		if err != nil {
			return nil, err
		}

		rpcAddr, err := v.GetExternalAddress(ctx, "26657")
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, Node{
			GRPC: grpcAddr,
			RPC:  fmt.Sprintf("http://%s", rpcAddr),
		})
	}

	var mnemonics []string
	for _, w := range chain.GetValidatorWallets() {
		mnemonics = append(mnemonics, w.Mnemonic())
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
		Runtime:             loadTestConfig.Runtime,
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

func (a *Activity) RunLoadTest(ctx context.Context, chainState []byte, chainID string, loadTestConfig *LoadTestConfig, providerState []byte) (PackagedState, error) {
	logger, _ := zap.NewDevelopment()

	p, err := digitalocean.RestoreProvider(
		ctx,
		providerState,
		a.DOToken,
		a.TailscaleSettings,
		digitalocean.WithLogger(logger),
	)
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
		Command: []string{"/catalyst/loadtest.yml"},
		DataDir: "/catalyst",
	})

	if err != nil {
		return PackagedState{}, err
	}

	if err := task.WriteFile(ctx, "loadtest.yml", configBytes); err != nil {
		return PackagedState{}, fmt.Errorf("failed to write config file to task: %w", err)
	}

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

			fmt.Println("CATALYST TASK STATUS", status)
			if status != provider.TASK_STOPPED {
				continue
			}

			if err := task.DownloadDir(ctx, CatalystResultDir, CatalystResultDir); err != nil {
				return PackagedState{}, fmt.Errorf("failed to download results: %w", err)
			}

			entries, err := os.ReadDir(CatalystResultDir)
			if err != nil {
				return PackagedState{}, fmt.Errorf("failed to read results directory: %w", err)
			}

			var resultFile string
			for _, entry := range entries {
				fmt.Println("ENTRIES length", len(entries))
				if !entry.IsDir() && path.Base(entry.Name())[:len(LoadTestResultPrefix)] == LoadTestResultPrefix {
					resultFile = filepath.Join(CatalystResultDir, entry.Name())
					break
				}
			}

			if resultFile == "" {
				return PackagedState{}, fmt.Errorf("task completed but no result file found in %s", CatalystResultDir)
			}

			resultBytes, err := os.ReadFile(resultFile)
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

// FormatResults returns a markdown formatted string of the load test results
func (r *LoadTestResult) FormatResults() string {
	var output string
	output += "=== Load Test Results ===\n\n"

	// Overall stats
	output += "🎯 Overall Statistics:\n"
	output += fmt.Sprintf("Total Transactions: %d\n", r.Overall.TotalTransactions)
	output += fmt.Sprintf("Successful Transactions: %d\n", r.Overall.SuccessfulTransactions)
	output += fmt.Sprintf("Failed Transactions: %d\n", r.Overall.FailedTransactions)
	output += fmt.Sprintf("Average Gas Per Transaction: %d\n", r.Overall.AvgGasPerTransaction)
	output += fmt.Sprintf("Average Block Gas Utilization: %.2f%%\n", r.Overall.AvgBlockGasUtilization*100)
	output += fmt.Sprintf("Runtime: %s\n", r.Overall.Runtime)
	output += fmt.Sprintf("Blocks Processed: %d\n\n", r.Overall.BlocksProcessed)

	// Message type stats
	output += "📊 Message Type Statistics:\n"
	for msgType, stats := range r.ByMessage {
		output += fmt.Sprintf("\n%s:\n", msgType)
		output += "  Transactions:\n"
		output += fmt.Sprintf("    Total: %d\n", stats.Transactions.Total)
		output += fmt.Sprintf("    Successful: %d\n", stats.Transactions.Successful)
		output += fmt.Sprintf("    Failed: %d\n", stats.Transactions.Failed)
		output += "  Gas Usage:\n"
		output += fmt.Sprintf("    Average: %d\n", stats.Gas.Average)
		output += fmt.Sprintf("    Min: %d\n", stats.Gas.Min)
		output += fmt.Sprintf("    Max: %d\n", stats.Gas.Max)
		output += fmt.Sprintf("    Total: %d\n", stats.Gas.Total)
		if len(stats.Errors.BroadcastErrors) > 0 {
			output += "  Errors:\n"
			for errType, count := range stats.Errors.ErrorCounts {
				output += fmt.Sprintf("    %s: %d occurrences\n", errType, count)
			}
		}
	}

	// Node stats
	output += "\n🖥️  Node Statistics:\n"
	for nodeAddr, stats := range r.ByNode {
		output += fmt.Sprintf("\n%s:\n", nodeAddr)
		output += "  Transactions:\n"
		output += fmt.Sprintf("    Total: %d\n", stats.TransactionStats.Total)
		output += fmt.Sprintf("    Successful: %d\n", stats.TransactionStats.Successful)
		output += fmt.Sprintf("    Failed: %d\n", stats.TransactionStats.Failed)
		output += "  Message Distribution:\n"
		for msgType, count := range stats.MessageCounts {
			output += fmt.Sprintf("    %s: %d\n", msgType, count)
		}
		output += "  Gas Usage:\n"
		output += fmt.Sprintf("    Average: %d\n", stats.GasStats.Average)
		output += fmt.Sprintf("    Min: %d\n", stats.GasStats.Min)
		output += fmt.Sprintf("    Max: %d\n", stats.GasStats.Max)
	}

	// Block stats summary
	output += "\n📦 Block Statistics Summary:\n"
	output += fmt.Sprintf("Total Blocks: %d\n", len(r.ByBlock))
	if len(r.ByBlock) > 0 {
		var totalGasUtilization float64
		var maxGasUtilization float64
		minGasUtilization := r.ByBlock[0].GasUtilization
		var maxGasBlock int64
		var minGasBlock int64
		for _, block := range r.ByBlock {
			totalGasUtilization += block.GasUtilization
			if block.GasUtilization > maxGasUtilization {
				maxGasUtilization = block.GasUtilization
				maxGasBlock = block.BlockHeight
			}
			if block.GasUtilization < minGasUtilization {
				minGasUtilization = block.GasUtilization
				minGasBlock = block.BlockHeight
			}
		}
		avgGasUtilization := totalGasUtilization / float64(len(r.ByBlock))
		output += fmt.Sprintf("Average Gas Utilization: %.2f%%\n", avgGasUtilization*100)
		output += fmt.Sprintf("Min Gas Utilization: %.2f%% (Block %d)\n", minGasUtilization*100, minGasBlock)
		output += fmt.Sprintf("Max Gas Utilization: %.2f%% (Block %d)\n", maxGasUtilization*100, maxGasBlock)
	}

	if r.Error != "" {
		output += fmt.Sprintf("\n❌ Errors\n%s\n", r.Error)
	}

	return output
}
