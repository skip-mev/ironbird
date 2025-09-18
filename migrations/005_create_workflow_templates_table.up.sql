-- Create workflow templates table for pre-configured workflows
CREATE TABLE workflow_templates (
    template_id TEXT PRIMARY KEY NOT NULL,
    description TEXT,
    config TEXT NOT NULL, -- JSON serialized TestnetWorkflowRequest
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT DEFAULT ''
);