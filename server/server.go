package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	types2 "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	"github.com/uber-go/tally/v4/prometheus"
	temporalclient "go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
)

type IronbirdServer struct {
	temporalClient temporalclient.Client
	config         types.TemporalConfig
}

func NewIronbirdServer(config types.TemporalConfig) (*IronbirdServer, error) {
	temporalClient, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  config.Host,
		Namespace: config.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9090",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		return nil, err
	}

	return &IronbirdServer{
		temporalClient: temporalClient,
		config:         config,
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

	workflowID := fmt.Sprintf("testnet-%s-%s", req.Repo, req.SHA)
	fmt.Printf("workflowID:\n%s\n", string(workflowID))

	options := temporalclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: testnet.TaskQueue,
	}
	workflowRun, err := s.temporalClient.ExecuteWorkflow(context.Background(), options, testnet.Workflow, req)
	if err != nil {
		fmt.Println("Error executing workflow", err)
		return fmt.Errorf("failed to execute workflow: %w", err)
	}
	fmt.Println("workflowrun.GetID", workflowRun.GetID())

	response := workflowRun.GetID()

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
	//workflowID := strings.TrimPrefix(r.URL.Path, "/ironbird/workflow/")

	// TODO: Implement actual workflow status retrieval using temporal client
	// Example:
	// workflow := s.temporalClient.GetWorkflow(context.Background(), workflowID, "")
	// var result WorkflowStatus
	// err := workflow.GetResult(context.Background(), &result)

	//response := WorkflowStatus{
	//	WorkflowID: workflowID,
	//	Status:     "running",
	//	Nodes: []Node{
	//		{
	//			Name:    "validator-1",
	//			RPC:     "http://validator-1:26657",
	//			LCD:     "http://validator-1:1317",
	//			Metrics: "http://validator-1:26660",
	//		},
	//	},
	//	Monitoring: map[string]string{
	//		"grafana": "https://grafana.example.com/d/xyz",
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

func (s *IronbirdServer) Close() error {
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}
	return nil
}
