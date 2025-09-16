package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/skip-mev/ironbird/server/proto"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type DB interface {
	CreateWorkflow(workflow *Workflow) error
	GetWorkflow(workflowID string) (*Workflow, error)
	UpdateWorkflow(workflowID string, update WorkflowUpdate) error
	ListWorkflows(limit, offset int) ([]Workflow, error)
	DeleteWorkflow(workflowID string) error

	CreateWorkflowTemplate(template *WorkflowTemplate) error
	GetWorkflowTemplate(templateID string) (*WorkflowTemplate, error)
	UpdateWorkflowTemplate(templateID string, template *WorkflowTemplate) error
	ListWorkflowTemplates(limit, offset int) ([]WorkflowTemplate, error)
	DeleteWorkflowTemplate(templateID string) error

	ListTemplateWorkflows(templateID string, limit, offset int) ([]Workflow, error)

	Ping() error
	Close() error
}

type SQLiteDB struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewSQLiteDB(dbPath string, logger *zap.Logger) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	return &SQLiteDB{db: db, logger: logger}, nil
}

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

func (s *SQLiteDB) CreateWorkflow(workflow *Workflow) error {
	nodesJSON, err := workflow.NodesJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	validatorsJSON, err := workflow.ValidatorsJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal validators: %w", err)
	}

	loadBalancersJSON, err := workflow.LoadBalancersJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal loadbalancers: %w", err)
	}

	walletsJSON, err := workflow.WalletsJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal wallets: %w", err)
	}

	configJSON, err := workflow.ConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	monitoringLinksJSON, err := json.Marshal(workflow.MonitoringLinks)
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring links: %w", err)
	}

	loadTestSpecJSON, err := workflow.LoadTestSpecJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal load test spec: %w", err)
	}

	now := time.Now()
	query := `
		INSERT INTO workflows (
			workflow_id, nodes, validators, loadbalancers, wallets, monitoring_links, status, config, 
			load_test_spec, provider, template_id, run_name, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`

	err = s.db.QueryRow(
		query,
		workflow.WorkflowID,
		string(nodesJSON),
		string(validatorsJSON),
		string(loadBalancersJSON),
		string(walletsJSON),
		string(monitoringLinksJSON),
		workflow.Status,
		string(configJSON),
		string(loadTestSpecJSON),
		workflow.Provider,
		workflow.TemplateID,
		workflow.RunName,
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

func (s *SQLiteDB) GetWorkflow(workflowID string) (*Workflow, error) {
	query := `
		SELECT id, workflow_id, nodes, validators, loadbalancers, wallets, monitoring_links, status, config, 
		    load_test_spec, provider, template_id, run_name, created_at, updated_at
		FROM workflows
		WHERE workflow_id = ?`

	var workflow Workflow
	var nodesJSON, validatorsJSON, loadBalancersJSON, walletsJSON, configJSON, monitoringLinksJSON, loadTestSpecJSON string

	err := s.db.QueryRow(query, workflowID).Scan(
		&workflow.ID,
		&workflow.WorkflowID,
		&nodesJSON,
		&validatorsJSON,
		&loadBalancersJSON,
		&walletsJSON,
		&monitoringLinksJSON,
		&workflow.Status,
		&configJSON,
		&loadTestSpecJSON,
		&workflow.Provider,
		&workflow.TemplateID,
		&workflow.RunName,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := json.Unmarshal([]byte(nodesJSON), &workflow.Nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}

	if err := json.Unmarshal([]byte(validatorsJSON), &workflow.Validators); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validators: %w", err)
	}

	if err := json.Unmarshal([]byte(loadBalancersJSON), &workflow.LoadBalancers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal loadbalancers: %w", err)
	}

	if walletsJSON != "" && walletsJSON != "{}" {
		workflow.Wallets = &pb.WalletInfo{}
		if err := protojson.Unmarshal([]byte(walletsJSON), workflow.Wallets); err != nil {
			return nil, fmt.Errorf("failed to unmarshal wallets: %w", err)
		}
	}

	if err := json.Unmarshal([]byte(configJSON), &workflow.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := json.Unmarshal([]byte(monitoringLinksJSON), &workflow.MonitoringLinks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal monitoring links: %w", err)
	}

	if loadTestSpecJSON != "" && loadTestSpecJSON != "{}" {
		workflow.LoadTestSpec = json.RawMessage(loadTestSpecJSON)
	}

	return &workflow, nil
}

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

	if update.LoadBalancers != nil {
		loadBalancersJSON, err := json.Marshal(*update.LoadBalancers)
		if err != nil {
			return fmt.Errorf("failed to marshal loadbalancers: %w", err)
		}
		setParts = append(setParts, "loadbalancers = ?")
		args = append(args, string(loadBalancersJSON))
	}

	if update.Wallets != nil {
		walletsJSON, err := protojson.Marshal(update.Wallets)
		if err != nil {
			return fmt.Errorf("failed to marshal wallets: %w", err)
		}
		setParts = append(setParts, "wallets = ?")
		args = append(args, string(walletsJSON))
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

	if update.Provider != nil {
		setParts = append(setParts, "provider = ?")
		args = append(args, *update.Provider)
	}

	if update.TemplateID != nil {
		setParts = append(setParts, "template_id = ?")
		args = append(args, *update.TemplateID)
	}

	if update.RunName != nil {
		setParts = append(setParts, "run_name = ?")
		args = append(args, *update.RunName)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())

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

func (s *SQLiteDB) ListWorkflows(limit, offset int) ([]Workflow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT id, workflow_id, nodes, validators, loadbalancers, wallets, monitoring_links, status, config, 
		    load_test_spec, provider, template_id, run_name, created_at, updated_at
		FROM workflows
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		var workflow Workflow
		var nodesJSON, validatorsJSON, loadBalancersJSON, walletsJSON, configJSON, monitoringLinksJSON, loadTestSpecJSON string

		err := rows.Scan(
			&workflow.ID,
			&workflow.WorkflowID,
			&nodesJSON,
			&validatorsJSON,
			&loadBalancersJSON,
			&walletsJSON,
			&monitoringLinksJSON,
			&workflow.Status,
			&configJSON,
			&loadTestSpecJSON,
			&workflow.Provider,
			&workflow.TemplateID,
			&workflow.RunName,
			&workflow.CreatedAt,
			&workflow.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		if err := json.Unmarshal([]byte(nodesJSON), &workflow.Nodes); err != nil {
			s.logger.Error("failed to unmarshal nodes for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal nodes for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(validatorsJSON), &workflow.Validators); err != nil {
			s.logger.Error("failed to unmarshal validators for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal validators for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(loadBalancersJSON), &workflow.LoadBalancers); err != nil {
			s.logger.Error("failed to unmarshal loadbalancers for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal loadbalancers for workflow %s: %w", workflow.WorkflowID, err)
		}

		if walletsJSON != "" && walletsJSON != "{}" {
			workflow.Wallets = &pb.WalletInfo{}
			if err := protojson.Unmarshal([]byte(walletsJSON), workflow.Wallets); err != nil {
				s.logger.Error("failed to unmarshal wallets for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
				return nil, fmt.Errorf("failed to unmarshal wallets for workflow %s: %w", workflow.WorkflowID, err)
			}
		}

		if err := json.Unmarshal([]byte(configJSON), &workflow.Config); err != nil {
			s.logger.Error("failed to unmarshal config for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal config for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(monitoringLinksJSON), &workflow.MonitoringLinks); err != nil {
			s.logger.Error("failed to unmarshal monitoring links for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal monitoring links for workflow %s: %w", workflow.WorkflowID, err)
		}

		if loadTestSpecJSON != "" && loadTestSpecJSON != "{}" {
			workflow.LoadTestSpec = json.RawMessage(loadTestSpecJSON)
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return workflows, err
}

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

func (s *SQLiteDB) Ping() error {
	return s.db.Ping()
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteDB) CreateWorkflowTemplate(template *WorkflowTemplate) error {
	configJSON, err := template.ConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO workflow_templates (template_id, name, description, config, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err = s.db.Exec(query,
		template.TemplateID,
		template.Name,
		template.Description,
		string(configJSON),
		now,
		now,
		template.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create workflow template: %w", err)
	}

	return nil
}

func (s *SQLiteDB) GetWorkflowTemplate(templateID string) (*WorkflowTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, template_id, name, description, config, created_at, updated_at, created_by
		FROM workflow_templates
		WHERE template_id = ?`

	var template WorkflowTemplate
	var configJSON string

	err := s.db.QueryRowContext(ctx, query, templateID).Scan(
		&template.ID,
		&template.TemplateID,
		&template.Name,
		&template.Description,
		&configJSON,
		&template.CreatedAt,
		&template.UpdatedAt,
		&template.CreatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow template not found: %s", templateID)
		}
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}

	if err := json.Unmarshal([]byte(configJSON), &template.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &template, nil
}

func (s *SQLiteDB) UpdateWorkflowTemplate(templateID string, template *WorkflowTemplate) error {
	configJSON, err := template.ConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE workflow_templates 
		SET name = ?, description = ?, config = ?, updated_at = ?
		WHERE template_id = ?`

	result, err := s.db.Exec(query,
		template.Name,
		template.Description,
		string(configJSON),
		time.Now(),
		templateID,
	)
	if err != nil {
		return fmt.Errorf("failed to update workflow template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workflow template not found: %s", templateID)
	}

	return nil
}

func (s *SQLiteDB) ListWorkflowTemplates(limit, offset int) (templates []WorkflowTemplate, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT id, template_id, name, description, config, created_at, updated_at, created_by
		FROM workflow_templates
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow templates: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	for rows.Next() {
		var template WorkflowTemplate
		var configJSON string

		err := rows.Scan(
			&template.ID,
			&template.TemplateID,
			&template.Name,
			&template.Description,
			&configJSON,
			&template.CreatedAt,
			&template.UpdatedAt,
			&template.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow template: %w", err)
		}

		if err := json.Unmarshal([]byte(configJSON), &template.Config); err != nil {
			s.logger.Error("failed to unmarshal config for template", zap.String("template_id", template.TemplateID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal config for template %s: %w", template.TemplateID, err)
		}

		templates = append(templates, template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return templates, err
}

func (s *SQLiteDB) DeleteWorkflowTemplate(templateID string) error {
	query := "DELETE FROM workflow_templates WHERE template_id = ?"

	result, err := s.db.Exec(query, templateID)
	if err != nil {
		return fmt.Errorf("failed to delete workflow template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("workflow template not found: %s", templateID)
	}

	return nil
}

// Template workflow tracking implementation
func (s *SQLiteDB) ListTemplateWorkflows(templateID string, limit, offset int) (workflows []Workflow, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT id, workflow_id, nodes, validators, loadbalancers, wallets, monitoring_links, status, config, 
		    load_test_spec, provider, template_id, run_name, created_at, updated_at
		FROM workflows
		WHERE template_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, templateID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list template workflows: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error("failed to close rows", zap.Error(closeErr))
		}
	}()

	for rows.Next() {
		var workflow Workflow
		var nodesJSON, validatorsJSON, loadBalancersJSON, walletsJSON, configJSON, monitoringLinksJSON, loadTestSpecJSON string

		err := rows.Scan(
			&workflow.ID,
			&workflow.WorkflowID,
			&nodesJSON,
			&validatorsJSON,
			&loadBalancersJSON,
			&walletsJSON,
			&monitoringLinksJSON,
			&workflow.Status,
			&configJSON,
			&loadTestSpecJSON,
			&workflow.Provider,
			&workflow.TemplateID,
			&workflow.RunName,
			&workflow.CreatedAt,
			&workflow.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template workflow: %w", err)
		}

		if err := json.Unmarshal([]byte(nodesJSON), &workflow.Nodes); err != nil {
			s.logger.Error("failed to unmarshal nodes for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal nodes for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(validatorsJSON), &workflow.Validators); err != nil {
			s.logger.Error("failed to unmarshal validators for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal validators for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(loadBalancersJSON), &workflow.LoadBalancers); err != nil {
			s.logger.Error("failed to unmarshal loadbalancers for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal loadbalancers for workflow %s: %w", workflow.WorkflowID, err)
		}

		if walletsJSON != "" && walletsJSON != "{}" {
			workflow.Wallets = &pb.WalletInfo{}
			if err := protojson.Unmarshal([]byte(walletsJSON), workflow.Wallets); err != nil {
				s.logger.Error("failed to unmarshal wallets for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
				return nil, fmt.Errorf("failed to unmarshal wallets for workflow %s: %w", workflow.WorkflowID, err)
			}
		}

		if err := json.Unmarshal([]byte(configJSON), &workflow.Config); err != nil {
			s.logger.Error("failed to unmarshal config for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal config for workflow %s: %w", workflow.WorkflowID, err)
		}

		if err := json.Unmarshal([]byte(monitoringLinksJSON), &workflow.MonitoringLinks); err != nil {
			s.logger.Error("failed to unmarshal monitoring links for workflow", zap.String("workflow_id", workflow.WorkflowID), zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal monitoring links for workflow %s: %w", workflow.WorkflowID, err)
		}

		if loadTestSpecJSON != "" && loadTestSpecJSON != "{}" {
			workflow.LoadTestSpec = json.RawMessage(loadTestSpecJSON)
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return workflows, err
}
