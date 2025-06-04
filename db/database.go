package db

import (
	"fmt"

	"github.com/skip-mev/ironbird/core/messages"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap"
)

func StringToWorkflowStatus(status string) WorkflowStatus {
	switch status {
	case "pending":
		return enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
	case "running":
		return enums.WORKFLOW_EXECUTION_STATUS_RUNNING
	case "completed":
		return enums.WORKFLOW_EXECUTION_STATUS_COMPLETED
	case "failed":
		return enums.WORKFLOW_EXECUTION_STATUS_FAILED
	case "canceled":
		return enums.WORKFLOW_EXECUTION_STATUS_CANCELED
	case "terminated":
		return enums.WORKFLOW_EXECUTION_STATUS_TERMINATED
	default:
		return enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
	}
}

// DatabaseService provides simple database operations (not activities)
type DatabaseService struct {
	DB     DB
	Logger *zap.Logger
}

// NewDatabaseService creates a new database service
func NewDatabaseService(database DB, logger *zap.Logger) *DatabaseService {
	return &DatabaseService{
		DB:     database,
		Logger: logger,
	}
}

// CreateWorkflow creates a new workflow record in the database
func (s *DatabaseService) CreateWorkflow(workflowID string, config messages.TestnetWorkflowRequest, status string) error {
	s.Logger.Info("Creating workflow record", zap.String("workflow_id", workflowID))

	dbStatus := StringToWorkflowStatus(status)

	workflow := &Workflow{
		WorkflowID:      workflowID,
		Nodes:           []messages.Node{},
		Validators:      []messages.Node{},
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

	dbStatus := StringToWorkflowStatus(status)
	update := WorkflowUpdate{
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
func (s *DatabaseService) UpdateWorkflowNodes(workflowID string, nodes []messages.Node, validators []messages.Node) error {
	s.Logger.Info("Updating workflow nodes",
		zap.String("workflow_id", workflowID),
		zap.Int("nodes_count", len(nodes)),
		zap.Int("validators_count", len(validators)))

	update := WorkflowUpdate{
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

	update := WorkflowUpdate{
		MonitoringLinks: &monitoringLinks,
	}

	if err := s.DB.UpdateWorkflow(workflowID, update); err != nil {
		s.Logger.Error("Failed to update workflow monitoring", zap.Error(err))
		return fmt.Errorf("failed to update workflow monitoring: %w", err)
	}

	s.Logger.Info("Successfully updated workflow monitoring", zap.String("workflow_id", workflowID))
	return nil
}

// UpdateWorkflowLoadBalancers updates the loadbalancers for a workflow
func (s *DatabaseService) UpdateWorkflowLoadBalancers(workflowID string, loadbalancers []messages.Node) error {
	s.Logger.Info("Updating workflow loadbalancers",
		zap.String("workflow_id", workflowID),
		zap.Int("loadbalancers_count", len(loadbalancers)))

	update := WorkflowUpdate{
		LoadBalancers: &loadbalancers,
	}

	if err := s.DB.UpdateWorkflow(workflowID, update); err != nil {
		s.Logger.Error("Failed to update workflow loadbalancers", zap.Error(err))
		return fmt.Errorf("failed to update workflow loadbalancers: %w", err)
	}

	s.Logger.Info("Successfully updated workflow loadbalancers", zap.String("workflow_id", workflowID))
	return nil
}
