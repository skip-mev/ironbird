package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	types2 "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/db"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	"github.com/uber-go/tally/v4/prometheus"
	temporalclient "go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
)

// WorkflowResponse represents the response from workflow operations
type WorkflowResponse struct {
	WorkflowID string `json:"WorkflowID"`
	Status     string `json:"Status"`
}

// Node represents a testnet node with its endpoints
type Node struct {
	Name    string `json:"Name"`
	RPC     string `json:"RPC"`
	LCD     string `json:"LCD"`
	Address string `json:"Address"`
}

// WorkflowStatus represents the complete status of a workflow
type WorkflowStatus struct {
	WorkflowID    string                          `json:"WorkflowID"`
	Status        string                          `json:"Status"`
	Nodes         []Node                          `json:"Nodes"`
	Validators    []Node                          `json:"Validators"`
	LoadBalancers []Node                          `json:"LoadBalancers"`
	Monitoring    map[string]string               `json:"Monitoring"`
	Config        messages.TestnetWorkflowRequest `json:"Config,omitempty"`

	// Individual fields from the database
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
}

func NewIronbirdServer(config types.TemporalConfig, database db.DB) (*IronbirdServer, error) {
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

	return &IronbirdServer{
		temporalClient: temporalClient,
		config:         config,
		db:             database,
	}, nil
}

func (s *IronbirdServer) HandleCreateWorkflow(w http.ResponseWriter, r *http.Request) error {
	var req messages.TestnetWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return nil
	}

	prettyJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
	} else {
		fmt.Printf("Received workflow request:\n%s\n", string(prettyJSON))
	}

	options := temporalclient.StartWorkflowOptions{
		TaskQueue: testnet.TaskQueue,
	}

	workflowRun, err := s.temporalClient.ExecuteWorkflow(context.TODO(), options, testnet.Workflow, req)
	if err != nil {
		fmt.Printf("Error executing workflow: %+v\n", err)
		http.Error(w, fmt.Sprintf("failed to execute workflow: %v", err), http.StatusInternalServerError)
		return err
	}
	fmt.Println("workflowrun.GetID", workflowRun.GetID())

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

//func (s *IronbirdServer) HandleUpdateWorkflow(w http.ResponseWriter, r *http.Request) error {
//	workflowID := strings.TrimPrefix(r.URL.Path, "/ironbird/workflow/")
//
//	var req messages.TestnetWorkflowRequest
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
//		return nil
//	}
//
//	// TODO: Implement actual workflow update using temporal client
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

	fmt.Printf("HandleGetWorkflow called with URL path: %s\n", r.URL.Path)
	fmt.Printf("Extracted workflowID: %s\n", workflowID)

	if workflowID == "" {
		fmt.Printf("Empty workflow ID, returning 400\n")
		http.Error(w, "workflow ID is required", http.StatusBadRequest)
		return nil
	}

	// Get workflow from database
	workflow, err := s.db.GetWorkflow(workflowID)
	if err != nil {
		fmt.Printf("Error getting workflow from database %s: %v\n", workflowID, err)
		http.Error(w, fmt.Sprintf("workflow not found: %v", err), http.StatusNotFound)
		return nil
	}

	// Convert database status to response format
	var status string
	switch workflow.Status {
	case "pending":
		status = "pending"
	case "running":
		status = "running"
	case "completed":
		status = "completed"
	case "failed":
		status = "failed"
	case "canceled":
		status = "canceled"
	case "terminated":
		status = "terminated"
	default:
		status = "unknown"
	}

	// Convert nodes from database format
	var nodes []Node
	for _, node := range workflow.Nodes {
		nodes = append(nodes, Node{
			Name:    node.Name,
			RPC:     node.Rpc,
			LCD:     node.Lcd,
			Address: node.Address,
		})
	}

	// Convert validators from database format
	var validators []Node
	for _, validator := range workflow.Validators {
		validators = append(validators, Node{
			Name:    validator.Name,
			RPC:     validator.Rpc,
			LCD:     validator.Lcd,
			Address: validator.Address,
		})
	}

	// Convert loadbalancers from database format
	var loadBalancers []Node
	for _, lb := range workflow.LoadBalancers {
		loadBalancers = append(loadBalancers, Node{
			Name:    lb.Name,
			RPC:     lb.Rpc,
			LCD:     lb.Lcd,
			Address: lb.Address,
		})
	}

	// Create monitoring links
	monitoring := map[string]string{}
	if workflow.MonitoringLinks != nil {
		monitoring = workflow.MonitoringLinks
	}

	response := WorkflowStatus{
		WorkflowID:         workflowID,
		Status:             status,
		Nodes:              nodes,
		Validators:         validators,
		LoadBalancers:      loadBalancers,
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

	fmt.Printf("Sending response from database: %+v\n", response)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	fmt.Printf("Successfully sent response from database\n")
	return nil
}

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

// HandleCancelWorkflow cancels a workflow using the Temporal client
func (s *IronbirdServer) HandleCancelWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Extract workflow ID from the URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid URL path format", http.StatusBadRequest)
		return nil
	}
	workflowID := parts[3]

	fmt.Printf("Canceling workflow: %s\n", workflowID)

	// Cancel the workflow using the Temporal client
	err := s.temporalClient.CancelWorkflow(context.Background(), workflowID, "")
	if err != nil {
		fmt.Printf("Error canceling workflow %s: %v\n", workflowID, err)
		http.Error(w, fmt.Sprintf("failed to cancel workflow: %v", err), http.StatusInternalServerError)
		return err
	}

	// No need to update the database for now

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

// HandleSignalWorkflow sends a signal to a workflow using the Temporal client
func (s *IronbirdServer) HandleSignalWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Extract workflow ID and signal name from the URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid URL path format", http.StatusBadRequest)
		return nil
	}
	workflowID := parts[3]
	signalName := parts[5]

	fmt.Printf("Sending signal '%s' to workflow: %s\n", signalName, workflowID)

	// Send the signal to the workflow using the Temporal client
	err := s.temporalClient.SignalWorkflow(context.Background(), workflowID, "", signalName, nil)
	if err != nil {
		fmt.Printf("Error sending signal to workflow %s: %v\n", workflowID, err)
		http.Error(w, fmt.Sprintf("failed to send signal to workflow: %v", err), http.StatusInternalServerError)
		return err
	}

	response := WorkflowResponse{
		WorkflowID: workflowID,
		Status:     "signaled",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *IronbirdServer) HandleListWorkflows(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("HandleListWorkflows called\n")

	// Get workflows from database
	dbWorkflows, err := s.db.ListWorkflows(100, 0) // Limit to 100 workflows for now
	if err != nil {
		fmt.Printf("Error listing workflows from database: %v\n", err)
		http.Error(w, fmt.Sprintf("failed to list workflows: %v", err), http.StatusInternalServerError)
		return nil
	}

	var workflows []WorkflowSummary
	for _, workflow := range dbWorkflows {
		// Convert database status to response format
		var status string
		switch workflow.Status {
		case "pending":
			status = "pending"
		case "running":
			status = "running"
		case "completed":
			status = "completed"
		case "failed":
			status = "failed"
		case "canceled":
			status = "canceled"
		case "terminated":
			status = "terminated"
		default:
			status = "unknown"
		}

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

	fmt.Printf("Returning %d workflows\n", len(workflows))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *IronbirdServer) Close() error {
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}
	return nil
}
