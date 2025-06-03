package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIronbirdServer_CreateWorkflow(t *testing.T) {
	server := &IronbirdServer{}

	tests := []struct {
		name           string
		request        TestnetWorkflowRequest
		expectedStatus int
	}{
		{
			name: "valid request",
			request: TestnetWorkflowRequest{
				Repo: "Cosmos SDK",
				SHA:  "test-sha",
				ChainConfig: ChainConfig{
					Name:  "test-chain",
					Image: "test-image:latest",
					GenesisModifications: []GenesisKV{
						{
							Key:   "consensus.params.block.max_gas",
							Value: "75000000",
						},
					},
					NumOfNodes:      4,
					NumOfValidators: 3,
				},
				LoadTestSpec: &LoadTestSpec{
					Name:        "test-load",
					Description: "Test load test",
					ChainID:     "test-chain",
					NumOfBlocks: 100,
					Msgs:        []Message{},
					TxTimeout:   time.Second * 30,
				},
				LongRunningTestnet: false,
				TestnetDuration:    "2h",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request - empty body",
			request:        TestnetWorkflowRequest{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/ironbird/workflow", bytes.NewReader(body))
			w := httptest.NewRecorder()

			err = server.ServeHTTP(w, req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response WorkflowResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.NotEmpty(t, response.WorkflowID)
			assert.Equal(t, "created", response.Status)
		})
	}
}

func TestIronbirdServer_UpdateWorkflow(t *testing.T) {
	server := &IronbirdServer{}

	tests := []struct {
		name           string
		workflowID     string
		request        TestnetWorkflowRequest
		expectedStatus int
	}{
		{
			name:       "valid update",
			workflowID: "test-workflow",
			request: TestnetWorkflowRequest{
				Repo: "Cosmos SDK",
				ChainConfig: ChainConfig{
					Name:            "updated-chain",
					NumOfNodes:      5,
					NumOfValidators: 4,
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPut, "/ironbird/workflow/"+tt.workflowID, bytes.NewReader(body))
			w := httptest.NewRecorder()

			err = server.ServeHTTP(w, req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response WorkflowResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, tt.workflowID, response.WorkflowID)
			assert.Equal(t, "updated", response.Status)
		})
	}
}

func TestIronbirdServer_GetWorkflow(t *testing.T) {
	server := &IronbirdServer{}

	tests := []struct {
		name           string
		workflowID     string
		expectedStatus int
	}{
		{
			name:           "valid workflow",
			workflowID:     "test-workflow",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ironbird/workflow/"+tt.workflowID, nil)
			w := httptest.NewRecorder()

			err := server.ServeHTTP(w, req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Workflow
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, tt.workflowID, response.WorkflowID)
			assert.Equal(t, "running", response.Status)
			assert.NotEmpty(t, response.Nodes)
			assert.NotEmpty(t, response.Monitoring)
		})
	}
}

func TestIronbirdServer_RunLoadTest(t *testing.T) {
	server := &IronbirdServer{}

	tests := []struct {
		name           string
		workflowID     string
		request        LoadTestSpec
		expectedStatus int
	}{
		{
			name:       "valid load test",
			workflowID: "test-workflow",
			request: LoadTestSpec{
				Name:        "test-load",
				Description: "Test load test",
				ChainID:     "test-chain",
				NumOfBlocks: 200,
				NumOfTxs:    1000,
				Msgs:        []Message{},
				TxTimeout:   time.Second * 30,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/ironbird/loadtest/"+tt.workflowID, bytes.NewReader(body))
			w := httptest.NewRecorder()

			err = server.ServeHTTP(w, req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response WorkflowResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, tt.workflowID, response.WorkflowID)
			assert.Equal(t, "started", response.Status)
			assert.Contains(t, response.Data, "load_test_id")
		})
	}
}

func TestIronbirdServer_InvalidEndpoint(t *testing.T) {
	server := &IronbirdServer{}

	req := httptest.NewRequest(http.MethodGet, "/invalid/endpoint", nil)
	w := httptest.NewRecorder()

	err := server.ServeHTTP(w, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint not found")
}
