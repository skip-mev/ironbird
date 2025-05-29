package database

import (
	"fmt"

	"github.com/skip-mev/ironbird/db"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types/testnet"
	"go.uber.org/zap"
)

// DatabaseService provides simple database operations (not activities)
type DatabaseService struct {
	DB     db.DB
	Logger *zap.Logger
}

// NewDatabaseService creates a new database service
func NewDatabaseService(database db.DB, logger *zap.Logger) *DatabaseService {
	return &DatabaseService{
		DB:     database,
		Logger: logger,
	}
}

// CreateWorkflow creates a new workflow record in the database
func (s *DatabaseService) CreateWorkflow(workflowID string, config messages.TestnetWorkflowRequest, status string) error {
	s.Logger.Info("Creating workflow record", zap.String("workflow_id", workflowID))

	dbStatus := db.WorkflowStatus(status)
	if dbStatus == "" {
		dbStatus = db.WorkflowStatusPending
	}

	workflow := &db.Workflow{
		WorkflowID:      workflowID,
		Nodes:           []testnet.Node{},
		Validators:      []testnet.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          dbStatus,
		Config:          config,
	}

	if err := s.DB.CreateWorkflow(workflow); err != nil {
		s.Logger.Error("Failed to create workflow record", zap.Error(err))
		return fmt.Errorf("failed to create workflow record: %w", err)
	}

	s.Logger.Info("Successfully created workflow record", zap.String("workflow_id", workflowID))
	return nil
}

// UpdateWorkflowStatus updates the status of a workflow
func (s *DatabaseService) UpdateWorkflowStatus(workflowID string, status string) error {
	s.Logger.Info("Updating workflow status",
		zap.String("workflow_id", workflowID),
		zap.String("status", status))

	dbStatus := db.WorkflowStatus(status)
	update := db.WorkflowUpdate{
		Status: &dbStatus,
	}

	if err := s.DB.UpdateWorkflow(workflowID, update); err != nil {
		s.Logger.Error("Failed to update workflow status", zap.Error(err))
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	s.Logger.Info("Successfully updated workflow status", zap.String("workflow_id", workflowID))
	return nil
}

// UpdateWorkflowNodes updates the nodes and validators for a workflow
func (s *DatabaseService) UpdateWorkflowNodes(workflowID string, nodes []testnet.Node, validators []testnet.Node) error {
	s.Logger.Info("Updating workflow nodes",
		zap.String("workflow_id", workflowID),
		zap.Int("nodes_count", len(nodes)),
		zap.Int("validators_count", len(validators)))

	update := db.WorkflowUpdate{
		Nodes:      &nodes,
		Validators: &validators,
	}

	if err := s.DB.UpdateWorkflow(workflowID, update); err != nil {
		s.Logger.Error("Failed to update workflow nodes", zap.Error(err))
		return fmt.Errorf("failed to update workflow nodes: %w", err)
	}

	s.Logger.Info("Successfully updated workflow nodes", zap.String("workflow_id", workflowID))
	return nil
}

// UpdateWorkflowMonitoring updates the monitoring links for a workflow
func (s *DatabaseService) UpdateWorkflowMonitoring(workflowID string, monitoringLinks map[string]string) error {
	s.Logger.Info("Updating workflow monitoring",
		zap.String("workflow_id", workflowID),
		zap.Any("monitoring_links", monitoringLinks))

	update := db.WorkflowUpdate{
		MonitoringLinks: &monitoringLinks,
	}

	if err := s.DB.UpdateWorkflow(workflowID, update); err != nil {
		s.Logger.Error("Failed to update workflow monitoring", zap.Error(err))
		return fmt.Errorf("failed to update workflow monitoring: %w", err)
	}

	s.Logger.Info("Successfully updated workflow monitoring", zap.String("workflow_id", workflowID))
	return nil
}
