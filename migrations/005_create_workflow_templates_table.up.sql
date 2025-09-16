-- Create workflow templates table for pre-configured workflows
CREATE TABLE workflow_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    config TEXT NOT NULL, -- JSON serialized TestnetWorkflowRequest
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT DEFAULT ''
);