package db

import (
	"encoding/json"
	"time"

	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types/testnet"
)

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusPending    WorkflowStatus = "pending"
	WorkflowStatusRunning    WorkflowStatus = "running"
	WorkflowStatusCompleted  WorkflowStatus = "completed"
	WorkflowStatusFailed     WorkflowStatus = "failed"
	WorkflowStatusCanceled   WorkflowStatus = "canceled"
	WorkflowStatusTerminated WorkflowStatus = "terminated"
)

// Workflow represents a workflow record in the database
type Workflow struct {
	ID                 int                             `json:"id" db:"id"`
	WorkflowID         string                          `json:"workflow_id" db:"workflow_id"`
	Nodes              []testnet.Node                  `json:"nodes" db:"nodes"`
	Validators         []testnet.Node                  `json:"validators" db:"validators"`
	LoadBalancers      []testnet.Node                  `json:"loadbalancers" db:"loadbalancers"`
	MonitoringLinks    map[string]string               `json:"monitoring_links" db:"monitoring_links"`
	Status             WorkflowStatus                  `json:"status" db:"status"`
	Config             messages.TestnetWorkflowRequest `json:"config" db:"config"`
	Repo               string                          `json:"repo" db:"repo"`
	SHA                string                          `json:"sha" db:"sha"`
	ChainName          string                          `json:"chain_name" db:"chain_name"`
	RunnerType         string                          `json:"runner_type" db:"runner_type"`
	NumOfNodes         int                             `json:"num_of_nodes" db:"num_of_nodes"`
	NumOfValidators    int                             `json:"num_of_validators" db:"num_of_validators"`
	NumWallets         int                             `json:"num_wallets" db:"num_wallets"`
	LongRunningTestnet bool                            `json:"long_running_testnet" db:"long_running_testnet"`
	TestnetDuration    int64                           `json:"testnet_duration" db:"testnet_duration"`
	LoadTestSpec       json.RawMessage                 `json:"load_test_spec" db:"load_test_spec"`
	CreatedAt          time.Time                       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time                       `json:"updated_at" db:"updated_at"`
}

// WorkflowUpdate represents fields that can be updated
type WorkflowUpdate struct {
	Nodes           *[]testnet.Node    `json:"nodes,omitempty"`
	Validators      *[]testnet.Node    `json:"validators,omitempty"`
	LoadBalancers   *[]testnet.Node    `json:"loadbalancers,omitempty"`
	MonitoringLinks *map[string]string `json:"monitoring_links,omitempty"`
	Status          *WorkflowStatus    `json:"status,omitempty"`
}

// ToJSON converts nodes to JSON for database storage
func (w *Workflow) NodesJSON() ([]byte, error) {
	return json.Marshal(w.Nodes)
}

// ToJSON converts validators to JSON for database storage
func (w *Workflow) ValidatorsJSON() ([]byte, error) {
	return json.Marshal(w.Validators)
}

// ToJSON converts loadbalancers to JSON for database storage
func (w *Workflow) LoadBalancersJSON() ([]byte, error) {
	return json.Marshal(w.LoadBalancers)
}

// ToJSON converts config to JSON for database storage
func (w *Workflow) ConfigJSON() ([]byte, error) {
	return json.Marshal(w.Config)
}

// LoadTestSpecJSON converts load test spec to JSON for database storage
func (w *Workflow) LoadTestSpecJSON() ([]byte, error) {
	if w.LoadTestSpec == nil {
		return []byte("{}"), nil
	}
	return w.LoadTestSpec, nil
}
