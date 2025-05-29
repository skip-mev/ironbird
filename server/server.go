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
	"go.temporal.io/api/workflowservice/v1"
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
	Metrics string `json:"Metrics"`
}

// WorkflowStatus represents the complete status of a workflow
type WorkflowStatus struct {
	WorkflowID string            `json:"WorkflowID"`
	Status     string            `json:"Status"`
	Nodes      []Node            `json:"Nodes"`
	Monitoring map[string]string `json:"Monitoring"`
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
		// Fallback to Temporal for backward compatibility
		return s.getWorkflowFromTemporal(w, workflowID)
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
			Metrics: node.Metrics,
		})
	}

	// Create monitoring links
	monitoring := map[string]string{}
	if workflow.MonitoringLinks != nil {
		monitoring = workflow.MonitoringLinks
	}

	response := WorkflowStatus{
		WorkflowID: workflowID,
		Status:     status,
		Nodes:      nodes,
		Monitoring: monitoring,
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

// getWorkflowFromTemporal is a fallback method for backward compatibility
func (s *IronbirdServer) getWorkflowFromTemporal(w http.ResponseWriter, workflowID string) error {
	// Get workflow status from Temporal
	fmt.Printf("Attempting to describe workflow: %s\n", workflowID)
	describe, err := s.temporalClient.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		fmt.Printf("Error describing workflow %s: %v\n", workflowID, err)
		http.Error(w, fmt.Sprintf("workflow not found: %v", err), http.StatusNotFound)
		return nil
	}

	fmt.Printf("Successfully described workflow. Status: %v\n", describe.WorkflowExecutionInfo.Status)

	var status string
	switch describe.WorkflowExecutionInfo.Status {
	case 1: // WORKFLOW_EXECUTION_STATUS_RUNNING
		status = "running"
	case 2: // WORKFLOW_EXECUTION_STATUS_COMPLETED
		status = "completed"
	case 3: // WORKFLOW_EXECUTION_STATUS_FAILED
		status = "failed"
	case 4: // WORKFLOW_EXECUTION_STATUS_CANCELED
		status = "canceled"
	case 5: // WORKFLOW_EXECUTION_STATUS_TERMINATED
		status = "terminated"
	case 6: // WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW
		status = "continued_as_new"
	case 7: // WORKFLOW_EXECUTION_STATUS_TIMED_OUT
		status = "timed_out"
	default:
		status = "unknown"
	}

	// For now, return mock data for nodes and monitoring
	// In a real implementation, you would query the workflow result or use signals to get the actual node information
	var nodes []Node
	var monitoring map[string]string

	if status == "running" || status == "completed" {
		// Mock data - in reality this would come from the workflow result or stored state
		nodes = []Node{
			{
				Name:    "validator-0",
				RPC:     "http://validator-0:26657",
				LCD:     "http://validator-0:1317",
				Metrics: "http://validator-0:26660",
			},
			{
				Name:    "validator-1",
				RPC:     "http://validator-1:26657",
				LCD:     "http://validator-1:1317",
				Metrics: "http://validator-1:26660",
			},
			{
				Name:    "validator-2",
				RPC:     "http://validator-2:26657",
				LCD:     "http://validator-2:1317",
				Metrics: "http://validator-2:26660",
			},
		}

		monitoring = map[string]string{
			"grafana":    "https://grafana.example.com/d/testnet-dashboard",
			"prometheus": "https://prometheus.example.com",
		}
	} else {
		nodes = []Node{}
		monitoring = map[string]string{}
	}

	response := WorkflowStatus{
		WorkflowID: workflowID,
		Status:     status,
		Nodes:      nodes,
		Monitoring: monitoring,
	}

	fmt.Printf("Sending response: %+v\n", response)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return nil
	}

	fmt.Printf("Successfully sent response\n")
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

func (s *IronbirdServer) HandleListWorkflows(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("HandleListWorkflows called\n")

	// List workflows from Temporal
	listRequest := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: s.config.Namespace,
		PageSize:  100, // Limit to 100 workflows for now
	}

	ctx := context.Background()
	listResponse, err := s.temporalClient.ListWorkflow(ctx, listRequest)
	if err != nil {
		fmt.Printf("Error listing workflows: %v\n", err)
		http.Error(w, fmt.Sprintf("failed to list workflows: %v", err), http.StatusInternalServerError)
		return nil
	}

	var workflows []WorkflowSummary
	for _, execution := range listResponse.Executions {
		var status string
		switch execution.Status {
		case 1: // WORKFLOW_EXECUTION_STATUS_RUNNING
			status = "running"
		case 2: // WORKFLOW_EXECUTION_STATUS_COMPLETED
			status = "completed"
		case 3: // WORKFLOW_EXECUTION_STATUS_FAILED
			status = "failed"
		case 4: // WORKFLOW_EXECUTION_STATUS_CANCELED
			status = "canceled"
		case 5: // WORKFLOW_EXECUTION_STATUS_TERMINATED
			status = "terminated"
		case 6: // WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW
			status = "continued_as_new"
		case 7: // WORKFLOW_EXECUTION_STATUS_TIMED_OUT
			status = "timed_out"
		default:
			status = "unknown"
		}

		startTime := ""
		if execution.StartTime != nil {
			startTime = execution.StartTime.AsTime().Format("2006-01-02 15:04:05")
		}

		// Extract repo and SHA from workflow ID if it follows the pattern "testnet-{repo}-{sha}"
		workflowID := execution.Execution.WorkflowId
		var repo, sha string
		if strings.HasPrefix(workflowID, "testnet-") {
			parts := strings.Split(workflowID, "-")
			if len(parts) >= 3 {
				repo = parts[1]
				sha = strings.Join(parts[2:], "-") // In case SHA contains dashes
			}
		}

		workflows = append(workflows, WorkflowSummary{
			WorkflowID: workflowID,
			Status:     status,
			StartTime:  startTime,
			Repo:       repo,
			SHA:        sha,
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
