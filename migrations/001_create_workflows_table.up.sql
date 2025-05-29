CREATE TABLE IF NOT EXISTS workflows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT UNIQUE NOT NULL,
    nodes TEXT DEFAULT '[]',
    validators TEXT DEFAULT '[]',
    monitoring_links TEXT DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    config TEXT NOT NULL,
    repo TEXT,
    sha TEXT,
    chain_name TEXT,
    runner_type TEXT,
    num_of_nodes INTEGER,
    num_of_validators INTEGER,
    long_running_testnet BOOLEAN DEFAULT FALSE,
    testnet_duration INTEGER DEFAULT 0,
    load_test_spec TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_workflows_workflow_id ON workflows(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at);

-- Trigger to automatically update updated_at
CREATE TRIGGER IF NOT EXISTS update_workflows_updated_at 
    AFTER UPDATE ON workflows
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE workflows SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
