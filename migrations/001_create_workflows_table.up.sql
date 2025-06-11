CREATE TABLE IF NOT EXISTS workflows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT UNIQUE NOT NULL,
    nodes TEXT DEFAULT '[]',
    validators TEXT DEFAULT '[]',
    loadbalancers TEXT DEFAULT '[]',
    wallets TEXT DEFAULT '{}',
    monitoring_links TEXT DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    config TEXT NOT NULL,
    load_test_spec TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);


CREATE TRIGGER IF NOT EXISTS update_workflows_updated_at
    AFTER UPDATE ON workflows
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE workflows SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
