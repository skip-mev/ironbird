-- Add template tracking fields to existing workflows table
ALTER TABLE workflows ADD COLUMN template_id TEXT DEFAULT '';
ALTER TABLE workflows ADD COLUMN run_name TEXT DEFAULT '';