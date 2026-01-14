-- Rollback: Remove detailed fields from projects and experiences
-- Date: 2026-01-05

-- Drop indexes first
DROP INDEX IF EXISTS idx_projects_links;
DROP INDEX IF EXISTS idx_experiences_links;

-- Remove columns from projects table
ALTER TABLE projects DROP COLUMN IF EXISTS catchphrase;
ALTER TABLE projects DROP COLUMN IF EXISTS functional_description;
ALTER TABLE projects DROP COLUMN IF EXISTS technical_description;
ALTER TABLE projects DROP COLUMN IF EXISTS images;
ALTER TABLE projects DROP COLUMN IF EXISTS links;

-- Remove columns from experiences table
ALTER TABLE experiences DROP COLUMN IF EXISTS catchphrase;
ALTER TABLE experiences DROP COLUMN IF EXISTS functional_description;
ALTER TABLE experiences DROP COLUMN IF EXISTS technical_description;
ALTER TABLE experiences DROP COLUMN IF EXISTS images;
ALTER TABLE experiences DROP COLUMN IF EXISTS links;
