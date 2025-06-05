package db

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/skip-mev/ironbird/messages"
	pb "github.com/skip-mev/ironbird/server/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func TestSQLiteDB(t *testing.T) {
	dbPath := "/tmp/test_ironbird.db"
	defer os.Remove(dbPath)

	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	err = db.RunMigrations("../../migrations")
	require.NoError(t, err)

	workflow := &Workflow{
		WorkflowID:      "test-workflow-123",
		Nodes:           []*pb.Node{},
		Validators:      []*pb.Node{},
		LoadBalancers:   []*pb.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED, // Pending
		Config:          messages.TestnetWorkflowRequest{},
		LoadTestSpec:    json.RawMessage("{}"),
	}

	err = db.CreateWorkflow(workflow)
	require.NoError(t, err)
	assert.NotZero(t, workflow.ID)
	assert.NotZero(t, workflow.CreatedAt)
	assert.NotZero(t, workflow.UpdatedAt)

	retrieved, err := db.GetWorkflow("test-workflow-123")
	require.NoError(t, err)
	assert.Equal(t, workflow.WorkflowID, retrieved.WorkflowID)
	assert.Equal(t, workflow.Status, retrieved.Status)
	assert.NotNil(t, retrieved.LoadBalancers)
	assert.Equal(t, 0, len(retrieved.LoadBalancers))

	newStatus := enums.WORKFLOW_EXECUTION_STATUS_RUNNING
	testLoadBalancer := pb.Node{
		Name:    "test-lb",
		Address: "192.168.1.100",
		Rpc:     "http://192.168.1.100:26657",
		Lcd:     "http://192.168.1.100:1317",
	}
	loadBalancers := []pb.Node{testLoadBalancer}

	update := WorkflowUpdate{
		Status:        &newStatus,
		LoadBalancers: &loadBalancers,
	}
	err = db.UpdateWorkflow("test-workflow-123", update)
	require.NoError(t, err)

	updated, err := db.GetWorkflow("test-workflow-123")
	require.NoError(t, err)
	assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, updated.Status)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
	assert.Len(t, updated.LoadBalancers, 1)
	assert.Equal(t, "test-lb", updated.LoadBalancers[0].Name)

	workflows, err := db.ListWorkflows(10, 0)
	require.NoError(t, err)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "test-workflow-123", workflows[0].WorkflowID)
	assert.NotNil(t, workflows[0].LoadBalancers)

	err = db.DeleteWorkflow("test-workflow-123")
	require.NoError(t, err)

	_, err = db.GetWorkflow("test-workflow-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

func TestSQLiteDB_Interface(t *testing.T) {
	dbPath := "/tmp/test_interface.db"
	defer os.Remove(dbPath)

	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var dbInterface DB = db
	err = dbInterface.Ping()
	require.NoError(t, err)
}
