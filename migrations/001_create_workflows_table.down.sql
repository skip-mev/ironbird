DROP TRIGGER IF EXISTS update_workflows_updated_at;
DROP INDEX IF EXISTS idx_workflows_created_at;
DROP INDEX IF EXISTS idx_workflows_status;
DROP INDEX IF EXISTS idx_workflows_workflow_id;
DROP TABLE IF EXISTS workflows;
