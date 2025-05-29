package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// DB interface defines the database operations
type DB interface {
	// Workflow operations
	CreateWorkflow(workflow *Workflow) error
	GetWorkflow(workflowID string) (*Workflow, error)
	UpdateWorkflow(workflowID string, update WorkflowUpdate) error
	ListWorkflows(limit, offset int) ([]Workflow, error)
	DeleteWorkflow(workflowID string) error

	// Health check
	Ping() error
	Close() error
}

// SQLiteDB implements the DB interface for SQLite
type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	return &SQLiteDB{db: db}, nil
}

// RunMigrations runs the database migrations
func (s *SQLiteDB) RunMigrations(migrationsPath string) error {
	driver, err := sqlite3.WithInstance(s.db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite3 driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// CreateWorkflow creates a new workflow record
func (s *SQLiteDB) CreateWorkflow(workflow *Workflow) error {
	nodesJSON, err := workflow.NodesJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	validatorsJSON, err := workflow.ValidatorsJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal validators: %w", err)
	}

	configJSON, err := workflow.ConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	monitoringLinksJSON, err := json.Marshal(workflow.MonitoringLinks)
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring links: %w", err)
	}

	now := time.Now()
	query := `
		INSERT INTO workflows (workflow_id, nodes, validators, monitoring_links, status, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`

	err = s.db.QueryRow(
		query,
		workflow.WorkflowID,
		string(nodesJSON),
		string(validatorsJSON),
		string(monitoringLinksJSON),
		workflow.Status,
		string(configJSON),
		now,
		now,
	).Scan(&workflow.ID)

	if err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	return nil
}

// GetWorkflow retrieves a workflow by workflow ID
func (s *SQLiteDB) GetWorkflow(workflowID string) (*Workflow, error) {
	query := `
		SELECT id, workflow_id, nodes, validators, monitoring_links, status, config, created_at, updated_at
		FROM workflows
		WHERE workflow_id = ?`

	var workflow Workflow
	var nodesJSON, validatorsJSON, configJSON, monitoringLinksJSON string

	err := s.db.QueryRow(query, workflowID).Scan(
		&workflow.ID,
		&workflow.WorkflowID,
		&nodesJSON,
		&validatorsJSON,
		&monitoringLinksJSON,
		&workflow.Status,
		&configJSON,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(nodesJSON), &workflow.Nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}

	if err := json.Unmarshal([]byte(validatorsJSON), &workflow.Validators); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validators: %w", err)
	}

	if err := json.Unmarshal([]byte(configJSON), &workflow.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := json.Unmarshal([]byte(monitoringLinksJSON), &workflow.MonitoringLinks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal monitoring links: %w", err)
	}

	return &workflow, nil
}

// UpdateWorkflow updates a workflow record
func (s *SQLiteDB) UpdateWorkflow(workflowID string, update WorkflowUpdate) error {
	setParts := []string{}
	args := []interface{}{}

	if update.Nodes != nil {
		nodesJSON, err := json.Marshal(*update.Nodes)
		if err != nil {
			return fmt.Errorf("failed to marshal nodes: %w", err)
		}
		setParts = append(setParts, "nodes = ?")
		args = append(args, string(nodesJSON))
	}

	if update.Validators != nil {
		validatorsJSON, err := json.Marshal(*update.Validators)
		if err != nil {
			return fmt.Errorf("failed to marshal validators: %w", err)
		}
		setParts = append(setParts, "validators = ?")
		args = append(args, string(validatorsJSON))
	}

	if update.MonitoringLinks != nil {
		monitoringLinksJSON, err := json.Marshal(*update.MonitoringLinks)
		if err != nil {
			return fmt.Errorf("failed to marshal monitoring links: %w", err)
		}
		setParts = append(setParts, "monitoring_links = ?")
		args = append(args, string(monitoringLinksJSON))
	}

	if update.Status != nil {
		setParts = append(setParts, "status = ?")
		args = append(args, *update.Status)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add updated_at field
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())

	// Build the query
	setClause := ""
	for i, part := range setParts {
		if i > 0 {
			setClause += ", "
		}
		setClause += part
	}

	query := fmt.Sprintf("UPDATE workflows SET %s WHERE workflow_id = ?", setClause)
	args = append(args, workflowID)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	return nil
}

// ListWorkflows retrieves a list of workflows with pagination
func (s *SQLiteDB) ListWorkflows(limit, offset int) ([]Workflow, error) {
	query := `
		SELECT id, workflow_id, nodes, validators, monitoring_links, status, config, created_at, updated_at
		FROM workflows
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		var workflow Workflow
		var nodesJSON, validatorsJSON, configJSON, monitoringLinksJSON string

		err := rows.Scan(
			&workflow.ID,
			&workflow.WorkflowID,
			&nodesJSON,
			&validatorsJSON,
			&monitoringLinksJSON,
			&workflow.Status,
			&configJSON,
			&workflow.CreatedAt,
			&workflow.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal([]byte(nodesJSON), &workflow.Nodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
		}

		if err := json.Unmarshal([]byte(validatorsJSON), &workflow.Validators); err != nil {
			return nil, fmt.Errorf("failed to unmarshal validators: %w", err)
		}

		if err := json.Unmarshal([]byte(configJSON), &workflow.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if err := json.Unmarshal([]byte(monitoringLinksJSON), &workflow.MonitoringLinks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal monitoring links: %w", err)
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return workflows, nil
}

// DeleteWorkflow deletes a workflow record
func (s *SQLiteDB) DeleteWorkflow(workflowID string) error {
	query := "DELETE FROM workflows WHERE workflow_id = ?"

	result, err := s.db.Exec(query, workflowID)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	return nil
}

// Ping checks if the database connection is alive
func (s *SQLiteDB) Ping() error {
	return s.db.Ping()
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
