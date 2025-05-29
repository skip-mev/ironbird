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
	ID              int                             `json:"id" db:"id"`
	WorkflowID      string                          `json:"workflow_id" db:"workflow_id"`
	Nodes           []testnet.Node                  `json:"nodes" db:"nodes"`
	Validators      []testnet.Node                  `json:"validators" db:"validators"`
	MonitoringLinks map[string]string               `json:"monitoring_links" db:"monitoring_links"`
	Status          WorkflowStatus                  `json:"status" db:"status"`
	Config          messages.TestnetWorkflowRequest `json:"config" db:"config"`
	CreatedAt       time.Time                       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at" db:"updated_at"`
}

// WorkflowUpdate represents fields that can be updated
type WorkflowUpdate struct {
	Nodes           *[]testnet.Node    `json:"nodes,omitempty"`
	Validators      *[]testnet.Node    `json:"validators,omitempty"`
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

// ToJSON converts config to JSON for database storage
func (w *Workflow) ConfigJSON() ([]byte, error) {
	return json.Marshal(w.Config)
}
