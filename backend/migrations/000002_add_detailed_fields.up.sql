-- Migration: Add detailed fields to projects and experiences
-- Date: 2026-01-05
-- Description: Adds catchphrase, functional/technical descriptions, images, and links fields

-- Add new columns to projects table
ALTER TABLE projects ADD COLUMN IF NOT EXISTS catchphrase VARCHAR(200);
ALTER TABLE projects ADD COLUMN IF NOT EXISTS functional_description TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS technical_description TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS images TEXT[];
ALTER TABLE projects ADD COLUMN IF NOT EXISTS links JSONB;

-- Add new columns to experiences table
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS catchphrase VARCHAR(200);
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS functional_description TEXT;
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS technical_description TEXT;
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS images TEXT[];
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS links JSONB;

-- Add indexes for links JSONB (GIN index for efficient querying)
CREATE INDEX IF NOT EXISTS idx_projects_links ON projects USING GIN (links);
CREATE INDEX IF NOT EXISTS idx_experiences_links ON experiences USING GIN (links);
