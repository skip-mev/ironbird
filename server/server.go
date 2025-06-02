package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	types2 "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/db"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.uber.org/zap"
)

// WorkflowResponse represents the response from workflow operations
type WorkflowResponse struct {
	WorkflowID string `json:"WorkflowID"`
	Status     string `json:"Status"`
}

type Workflow struct {
	WorkflowID    string            `json:"WorkflowID"`
	Status        string            `json:"Status"`
	Nodes         []messages.Node   `json:"Nodes"`
	Validators    []messages.Node   `json:"Validators"`
	LoadBalancers []messages.Node   `json:"LoadBalancers"`
	Monitoring    map[string]string `json:"Monitoring"`
	// TODO(nadim-az): keep either config, or individual fields in the Workflow struct
	Config messages.TestnetWorkflowRequest `json:"Config,omitempty"`

	Repo               string          `json:"repo,omitempty"`
	SHA                string          `json:"sha,omitempty"`
	ChainName          string          `json:"chainName,omitempty"`
	RunnerType         string          `json:"runnerType,omitempty"`
	NumOfNodes         int             `json:"numOfNodes,omitempty"`
	NumOfValidators    int             `json:"numOfValidators,omitempty"`
	NumWallets         int             `json:"numWallets,omitempty"`
	LongRunningTestnet bool            `json:"longRunningTestnet,omitempty"`
	TestnetDuration    int64           `json:"testnetDuration,omitempty"`
	LoadTestSpec       json.RawMessage `json:"loadTestSpec,omitempty"`
}

// WorkflowSummary represents a summary of a workflow for listing
type WorkflowSummary struct {
	WorkflowID string `json:"WorkflowID"`
	Status     string `json:"Status"`
	StartTime  string `json:"StartTime"`
	Repo       string `json:"Repo,omitempty"`
	SHA        string `json:"SHA,omitempty"`
}

// WorkflowListResponse represents the response for listing workflows
type WorkflowListResponse struct {
	Workflows []WorkflowSummary `json:"Workflows"`
	Count     int               `json:"Count"`
}

type IronbirdServer struct {
	temporalClient temporalclient.Client
	config         types.TemporalConfig
	db             db.DB
	stopCh         chan struct{}
	logger         *zap.Logger
}

func NewIronbirdServer(config types.TemporalConfig, database db.DB, logger *zap.Logger) (*IronbirdServer, error) {
	temporalClient, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  config.Host,
		Namespace: config.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9091",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		return nil, err
	}

	server := &IronbirdServer{
		temporalClient: temporalClient,
		config:         config,
		db:             database,
		stopCh:         make(chan struct{}),
		logger:         logger,
	}

	go server.startWorkflowStatusUpdater()

	return server, nil
}

// startWorkflowStatusUpdater starts a background process that fetches workflow statuses
// from the temporal client and updates them in the database every 10 seconds
func (s *IronbirdServer) startWorkflowStatusUpdater() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	s.logger.Info("starting workflow status updater background process")

	for {
		select {
		case <-ticker.C:
			s.updateWorkflowStatuses()
		case <-s.stopCh:
			s.logger.Info("stopping workflow status updater background process")
			return
		}
	}
}

// updateWorkflowStatuses fetches all workflows from the database and updates their status
// by querying the temporal client
func (s *IronbirdServer) updateWorkflowStatuses() {
	workflows, err := s.db.ListWorkflows(1000, 0)
	if err != nil {
		s.logger.Error("Error listing workflows from database", zap.Error(err))
		return
	}

	for _, workflow := range workflows {
		// Skip workflows that are already in a terminal state
		if workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_FAILED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_CANCELED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_TERMINATED {
			continue
		}

		workflowID := workflow.WorkflowID
		desc, err := s.temporalClient.DescribeWorkflowExecution(
			context.Background(),
			workflowID,
			"", // Empty run ID to get the latest run
		)

		if err != nil {
			s.logger.Error("Error describing workflow",
				zap.String("workflowID", workflowID),
				zap.Error(err))
			continue
		}

		var newStatus db.WorkflowStatus
		newStatus = desc.WorkflowExecutionInfo.Status

		if newStatus != workflow.Status {
			s.logger.Info("updating workflow status",
				zap.String("workflowID", workflowID),
				zap.String("oldStatus", db.WorkflowStatusToString(workflow.Status)),
				zap.String("newStatus", db.WorkflowStatusToString(newStatus)))

			update := db.WorkflowUpdate{
				Status: &newStatus,
			}

			if err := s.db.UpdateWorkflow(workflowID, update); err != nil {
				s.logger.Error("updating workflow status",
					zap.String("workflowID", workflowID),
					zap.Error(err))
			}
		}
	}
}

func (s *IronbirdServer) HandleCreateWorkflow(w http.ResponseWriter, r *http.Request) error {
	var req messages.TestnetWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return nil
	}

	prettyJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		s.logger.Error("marshaling request", zap.Error(err))
	} else {
		s.logger.Info("received workflow request", zap.String("request", string(prettyJSON)))
	}

	options := temporalclient.StartWorkflowOptions{
		TaskQueue: messages.TaskQueue,
	}

	workflowRun, err := s.temporalClient.ExecuteWorkflow(context.TODO(), options, testnet.Workflow, req)
	if err != nil {
		s.logger.Error("executing workflow", zap.Error(err))
		http.Error(w, fmt.Sprintf("failed to execute workflow: %v", err), http.StatusInternalServerError)
		return err
	}
	s.logger.Info("workflow execution started", zap.String("workflowID", workflowRun.GetID()))

	workflowID := workflowRun.GetID()
	workflow := &db.Workflow{
		WorkflowID:      workflowID,
		Nodes:           []messages.Node{},
		Validators:      []messages.Node{},
		LoadBalancers:   []messages.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		Config:          req,
		Repo:            req.Repo,
		SHA:             req.SHA,
		ChainName:       req.ChainConfig.Name,
		RunnerType:      string(req.RunnerType),
		NumOfNodes:      int(req.ChainConfig.NumOfNodes),
		NumOfValidators: int(req.ChainConfig.NumOfValidators),
	}

	if err := s.db.CreateWorkflow(workflow); err != nil {
		s.logger.Error("creating workflow record", zap.Error(err))
	}

	response := WorkflowResponse{
		WorkflowID: workflowRun.GetID(),
		Status:     "in progress",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

// TODO(nadim-az): Implement update workflow
//func (s *IronbirdServer) HandleUpdateWorkflow(w http.ResponseWriter, r *http.Request) error {
//	workflowID := strings.TrimPrefix(r.URL.Path, "/ironbird/workflow/")
//
//	var req messages.TestnetWorkflowRequest
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
//		return nil
//	}
//
//	// Example:
//	// s.temporalClient.SignalWorkflow(context.Background(), workflowID, "", "update_signal", req)
//
//	response := WorkflowResponse{
//		WorkflowID: workflowID,
//		Status:     "updated",
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	if err := json.NewEncoder(w).Encode(response); err != nil {
//		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
//		return nil
//	}
//
//	return nil
//}

func (s *IronbirdServer) HandleGetWorkflow(w http.ResponseWriter, r *http.Request) error {
	workflowID := strings.TrimPrefix(r.URL.Path, "/ironbird/workflow/")

	s.logger.Debug("HandleGetWorkflow called",
		zap.String("path", r.URL.Path),
		zap.String("workflowID", workflowID))

	if workflowID == "" {
		s.logger.Warn("Empty workflow ID, returning 400")
		http.Error(w, "workflow ID is required", http.StatusBadRequest)
		return nil
	}

	workflow, err := s.db.GetWorkflow(workflowID)
	if err != nil {
		s.logger.Error("getting workflow from database",
			zap.String("workflowID", workflowID),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("workflow not found: %v", err), http.StatusNotFound)
		return nil
	}

	// Convert database status to response format
	status := db.WorkflowStatusToString(workflow.Status)

	monitoring := map[string]string{}
	if workflow.MonitoringLinks != nil {
		monitoring = workflow.MonitoringLinks
	}

	response := Workflow{
		WorkflowID:         workflowID,
		Status:             status,
		Nodes:              workflow.Nodes,
		Validators:         workflow.Validators,
		LoadBalancers:      workflow.LoadBalancers,
		Monitoring:         monitoring,
		Config:             workflow.Config,
		Repo:               workflow.Repo,
		SHA:                workflow.SHA,
		ChainName:          workflow.ChainName,
		RunnerType:         workflow.RunnerType,
		NumOfNodes:         workflow.NumOfNodes,
		NumOfValidators:    workflow.NumOfValidators,
		NumWallets:         workflow.NumWallets,
		LongRunningTestnet: workflow.LongRunningTestnet,
		TestnetDuration:    workflow.TestnetDuration,
		LoadTestSpec:       workflow.LoadTestSpec,
	}

	s.logger.Debug("Sending response from database", zap.Any("response", response))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("encoding response", zap.Error(err))
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	s.logger.Debug("Successfully sent response from database")
	return nil
}

// TODO(nadim-az): implement adhoc load test runs
func (s *IronbirdServer) HandleRunLoadTest(w http.ResponseWriter, r *http.Request) error {
	//workflowID := strings.TrimPrefix(r.URL.Path, "/ironbird/loadtest/")

	var req types2.LoadTestSpec
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return nil
	}

	// TODO: Implement actual load test execution using temporal client
	// Example:
	// Options for starting a load test workflow
	// options := temporalclient.StartWorkflowOptions{
	//     ID:        fmt.Sprintf("loadtest-%s", workflowID),
	//     TaskQueue: "loadtest-queue",
	// }
	// loadTestRun, err := s.temporalClient.ExecuteWorkflow(context.Background(), options, "LoadTestWorkflow", req)

	// Or signal an existing workflow to start load testing
	// s.temporalClient.SignalWorkflow(context.Background(), workflowID, "", "start_loadtest", req)

	//loadTestID := fmt.Sprintf("lt-%s-%s", workflowID, req.Name)
	//
	//response := WorkflowResponse{
	//	WorkflowID: workflowID,
	//	Status:     "started",
	//	Data: map[string]interface{}{
	//		"load_test_id": loadTestID,
	//	},
	//}
	//
	//w.Header().Set("Content-Type", "application/json")
	//if err := json.NewEncoder(w).Encode(response); err != nil {
	//	http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
	//	return nil
	//}

	return nil
}

func (s *IronbirdServer) HandleCancelWorkflow(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid URL path format", http.StatusBadRequest)
		return nil
	}
	workflowID := parts[3]
	s.logger.Info("canceling workflow", zap.String("workflowID", workflowID))

	err := s.temporalClient.CancelWorkflow(context.Background(), workflowID, "")
	if err != nil {
		s.logger.Error("canceling workflow",
			zap.String("workflowID", workflowID),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("failed to cancel workflow: %v", err), http.StatusInternalServerError)
		return err
	}

	response := WorkflowResponse{
		WorkflowID: workflowID,
		Status:     "canceled",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *IronbirdServer) HandleSignalWorkflow(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid URL path format", http.StatusBadRequest)
		return nil
	}
	workflowID := parts[3]
	signalName := parts[5]

	s.logger.Info("sending signal to workflow",
		zap.String("signal", signalName),
		zap.String("workflowID", workflowID))

	err := s.temporalClient.SignalWorkflow(context.Background(), workflowID, "", signalName, nil)
	if err != nil {
		s.logger.Error("sending signal to workflow",
			zap.String("workflowID", workflowID),
			zap.Error(err))
		http.Error(w, fmt.Sprintf("failed to send signal to workflow: %v", err), http.StatusInternalServerError)
		return err
	}

	response := WorkflowResponse{
		WorkflowID: workflowID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *IronbirdServer) HandleListWorkflows(w http.ResponseWriter, r *http.Request) error {
	dbWorkflows, err := s.db.ListWorkflows(100, 0)
	if err != nil {
		s.logger.Error("listing workflows from database", zap.Error(err))
		http.Error(w, fmt.Sprintf("failed to list workflows: %v", err), http.StatusInternalServerError)
		return nil
	}

	var workflows []WorkflowSummary
	for _, workflow := range dbWorkflows {
		status := db.WorkflowStatusToString(workflow.Status)

		startTime := workflow.CreatedAt.Format("2006-01-02 15:04:05")

		workflows = append(workflows, WorkflowSummary{
			WorkflowID: workflow.WorkflowID,
			Status:     status,
			StartTime:  startTime,
			Repo:       workflow.Repo,
			SHA:        workflow.SHA,
		})
	}

	response := WorkflowListResponse{
		Workflows: workflows,
		Count:     len(workflows),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("encoding response", zap.Error(err))
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *IronbirdServer) Close() error {
	close(s.stopCh)

	if s.temporalClient != nil {
		s.temporalClient.Close()
	}
	return nil
}
