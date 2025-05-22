package server

// LoadTestSpec represents the configuration for a load test
type LoadTestSpec struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ChainID      string    `json:"chain_id"`
	NumOfTxs     int       `json:"num_of_txs"`
	NumOfBlocks  int       `json:"num_of_blocks"`
	Msgs         []Message `json:"msgs"`
	UnorderedTxs bool      `json:"unordered_txs"`
	TxTimeout    string    `json:"tx_timeout"`
}

// Message represents a transaction message configuration
type Message struct {
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

// ChainConfig represents the configuration for a chain
type ChainConfig struct {
	Name                 string      `json:"name"`
	Image                string      `json:"image"`
	GenesisModifications []GenesisKV `json:"genesis_modifications"`
	NumOfNodes           int         `json:"num_of_nodes"`
	NumOfValidators      int         `json:"num_of_validators"`
}

// GenesisKV represents a key-value pair for genesis modifications
type GenesisKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TestnetWorkflowRequest represents the request to create or update a testnet workflow
type TestnetWorkflowRequest struct {
	Repo               string        `json:"repo"`
	SHA                string        `json:"sha"`
	ChainConfig        ChainConfig   `json:"chain_config"`
	LoadTestSpec       *LoadTestSpec `json:"load_test_spec,omitempty"`
	LongRunningTestnet bool          `json:"long_running_testnet"`
	TestnetDuration    string        `json:"testnet_duration"`
	RunLoadTest        bool          `json:"run_load_test"`
}

// WorkflowResponse represents the response for workflow operations
type WorkflowResponse struct {
	WorkflowID string                 `json:"workflow_id"`
	Status     string                 `json:"status"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// Node represents a validator or node in the testnet
type Node struct {
	Name    string `json:"name"`
	RPC     string `json:"rpc"`
	LCD     string `json:"lcd"`
	Metrics string `json:"metrics"`
}

// WorkflowStatus represents the detailed status of a workflow
type WorkflowStatus struct {
	WorkflowID string            `json:"workflow_id"`
	Status     string            `json:"status"`
	Nodes      []Node            `json:"nodes"`
	Monitoring map[string]string `json:"monitoring"`
}

type TemporalConfig struct {
	Host      string `json:"host"`
	Namespace string `json:"namespace,omitempty"`
}
