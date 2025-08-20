package db

import (
	"encoding/json"
	"time"

	"github.com/skip-mev/ironbird/messages"
	pb "github.com/skip-mev/ironbird/server/proto"

	"go.temporal.io/api/enums/v1"
)

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
		return "running"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "failed"
	default:
		return "unknown"
	}
}

type Workflow struct {
	ID              int                             `json:"id" db:"id"`
	WorkflowID      string                          `json:"workflow_id" db:"workflow_id"`
	Nodes           []*pb.Node                      `json:"nodes" db:"nodes"`
	Validators      []*pb.Node                      `json:"validators" db:"validators"`
	LoadBalancers   []*pb.Node                      `json:"loadbalancers" db:"loadbalancers"`
	Wallets         *pb.WalletInfo                  `json:"wallets" db:"wallets"`
	MonitoringLinks map[string]string               `json:"monitoring_links" db:"monitoring_links"`
	Status          WorkflowStatus                  `json:"status" db:"status"`
	Config          messages.TestnetWorkflowRequest `json:"config" db:"config"`
	LoadTestSpec    json.RawMessage                 `json:"load_test_spec" db:"load_test_spec"`
	Provider        string                          `json:"provider" db:"provider"`
	CreatedAt       time.Time                       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at" db:"updated_at"`
}

type WorkflowUpdate struct {
	Nodes           *[]pb.Node         `json:"nodes,omitempty"`
	Validators      *[]pb.Node         `json:"validators,omitempty"`
	LoadBalancers   *[]pb.Node         `json:"loadbalancers,omitempty"`
	Wallets         *pb.WalletInfo     `json:"wallets,omitempty"`
	MonitoringLinks *map[string]string `json:"monitoring_links,omitempty"`
	Status          *WorkflowStatus    `json:"status,omitempty"`
	Provider        *string            `json:"provider,omitempty"`
}

func (w *Workflow) NodesJSON() ([]byte, error) {
	return json.Marshal(w.Nodes)
}

func (w *Workflow) ValidatorsJSON() ([]byte, error) {
	return json.Marshal(w.Validators)
}

func (w *Workflow) LoadBalancersJSON() ([]byte, error) {
	return json.Marshal(w.LoadBalancers)
}

func (w *Workflow) WalletsJSON() ([]byte, error) {
	if w.Wallets == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(w.Wallets)
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
