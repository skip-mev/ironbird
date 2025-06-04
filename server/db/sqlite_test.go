package db

import (
	"os"
	"testing"

	"github.com/skip-mev/ironbird/messages"
	pb "github.com/skip-mev/ironbird/server/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func TestSQLiteDB(t *testing.T) {
	// Create a temporary database file
	dbPath := "/tmp/test_ironbird.db"
	defer os.Remove(dbPath)

	// Create database connection
	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	err = db.RunMigrations("../migrations")
	require.NoError(t, err)

	// Test creating a workflow
	workflow := &Workflow{
		WorkflowID:      "test-workflow-123",
		Nodes:           []pb.Node{},
		Validators:      []pb.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED, // Pending
		Config:          messages.TestnetWorkflowRequest{},
	}

	err = db.CreateWorkflow(workflow)
	require.NoError(t, err)
	assert.NotZero(t, workflow.ID)
	assert.NotZero(t, workflow.CreatedAt)
	assert.NotZero(t, workflow.UpdatedAt)

	// Test getting the workflow
	retrieved, err := db.GetWorkflow("test-workflow-123")
	require.NoError(t, err)
	assert.Equal(t, workflow.WorkflowID, retrieved.WorkflowID)
	assert.Equal(t, workflow.Status, retrieved.Status)

	// Test updating the workflow
	newStatus := enums.WORKFLOW_EXECUTION_STATUS_RUNNING
	update := WorkflowUpdate{
		Status: &newStatus,
	}
	err = db.UpdateWorkflow("test-workflow-123", update)
	require.NoError(t, err)

	// Verify the update
	updated, err := db.GetWorkflow("test-workflow-123")
	require.NoError(t, err)
	assert.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, updated.Status)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))

	// Test listing workflows
	workflows, err := db.ListWorkflows(10, 0)
	require.NoError(t, err)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "test-workflow-123", workflows[0].WorkflowID)

	// Test deleting the workflow
	err = db.DeleteWorkflow("test-workflow-123")
	require.NoError(t, err)

	// Verify deletion
	_, err = db.GetWorkflow("test-workflow-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

func TestSQLiteDB_Interface(t *testing.T) {
	// Test that SQLiteDB implements the DB interface
	dbPath := "/tmp/test_interface.db"
	defer os.Remove(dbPath)

	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify it implements the DB interface
	var dbInterface DB = db
	err = dbInterface.Ping()
	require.NoError(t, err)
}
