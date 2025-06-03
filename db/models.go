package db

import (
	"encoding/json"
	"time"

	"github.com/skip-mev/ironbird/messages"
	"go.temporal.io/api/enums/v1"
)

// WorkflowStatus is an alias for Temporal's WorkflowExecutionStatus
type WorkflowStatus = enums.WorkflowExecutionStatus

func WorkflowStatusToString(status WorkflowStatus) string {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED:
		return "pending"
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "running"
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failed"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "canceled"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return "terminated"
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return "running" // Treat as still running
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "failed" // Treat as failed
	default:
		return "unknown"
	}
}

// Workflow represents a workflow record in the database
type Workflow struct {
	ID                 int                             `json:"id" db:"id"`
	WorkflowID         string                          `json:"workflow_id" db:"workflow_id"`
	Nodes              []messages.Node                 `json:"nodes" db:"nodes"`
	Validators         []messages.Node                 `json:"validators" db:"validators"`
	LoadBalancers      []messages.Node                 `json:"loadbalancers" db:"loadbalancers"`
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
	Nodes           *[]messages.Node   `json:"nodes,omitempty"`
	Validators      *[]messages.Node   `json:"validators,omitempty"`
	LoadBalancers   *[]messages.Node   `json:"loadbalancers,omitempty"`
	MonitoringLinks *map[string]string `json:"monitoring_links,omitempty"`
	Status          *WorkflowStatus    `json:"status,omitempty"`
}

// ToJSON converts nodes to JSON for database storage
func (w *Workflow) NodesJSON() ([]byte, error) {
	return json.Marshal(w.Nodes)
}

func (w *Workflow) ValidatorsJSON() ([]byte, error) {
	return json.Marshal(w.Validators)
}

func (w *Workflow) LoadBalancersJSON() ([]byte, error) {
	return json.Marshal(w.LoadBalancers)
}

func (w *Workflow) ConfigJSON() ([]byte, error) {
	return json.Marshal(w.Config)
}

func (w *Workflow) LoadTestSpecJSON() ([]byte, error) {
	if w.LoadTestSpec == nil {
		return []byte("{}"), nil
	}
	return w.LoadTestSpec, nil
}
