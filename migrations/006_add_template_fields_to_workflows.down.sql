-- Remove template fields from workflows table
ALTER TABLE workflows DROP COLUMN IF EXISTS template_id;
ALTER TABLE workflows DROP COLUMN IF EXISTS run_name;