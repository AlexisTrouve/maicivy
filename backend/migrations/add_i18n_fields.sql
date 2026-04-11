-- Migration: Add i18n (internationalization) fields for English translations
-- Date: 2026-01-12
-- Description: Adds English translation columns for experiences, skills, and projects
--              to support bilingual CV (French/English)
-- Author: Alexi

-- ==============================================================================
-- EXPERIENCES TABLE - Add English fields
-- ==============================================================================

-- Title translation (Développeur IT Polyvalent -> Versatile IT Developer)
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS title_en VARCHAR(255);
COMMENT ON COLUMN experiences.title_en IS 'English translation of job title';

-- Description translation (main job description)
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS description_en TEXT;
COMMENT ON COLUMN experiences.description_en IS 'English translation of job description';

-- Catchphrase translation (short phrase for card display)
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS catchphrase_en VARCHAR(200);
COMMENT ON COLUMN experiences.catchphrase_en IS 'English translation of catchphrase';

-- Functional description translation (what the job involved functionally)
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS functional_description_en TEXT;
COMMENT ON COLUMN experiences.functional_description_en IS 'English translation of functional description';

-- Technical description translation (technical details and stack)
ALTER TABLE experiences ADD COLUMN IF NOT EXISTS technical_description_en TEXT;
COMMENT ON COLUMN experiences.technical_description_en IS 'English translation of technical description';

-- ==============================================================================
-- SKILLS TABLE - Add English fields
-- ==============================================================================

-- Skill name translation (Go, C++, etc. usually stay the same, but some need translation)
ALTER TABLE skills ADD COLUMN IF NOT EXISTS name_en VARCHAR(100);
COMMENT ON COLUMN skills.name_en IS 'English translation of skill name';

-- Description translation (explains the skill)
ALTER TABLE skills ADD COLUMN IF NOT EXISTS description_en TEXT;
COMMENT ON COLUMN skills.description_en IS 'English translation of skill description';

-- ==============================================================================
-- PROJECTS TABLE - Add English fields
-- ==============================================================================

-- Title translation
ALTER TABLE projects ADD COLUMN IF NOT EXISTS title_en VARCHAR(255);
COMMENT ON COLUMN projects.title_en IS 'English translation of project title';

-- Description translation (short description)
ALTER TABLE projects ADD COLUMN IF NOT EXISTS description_en TEXT;
COMMENT ON COLUMN projects.description_en IS 'English translation of project description';

-- Catchphrase translation
ALTER TABLE projects ADD COLUMN IF NOT EXISTS catchphrase_en VARCHAR(200);
COMMENT ON COLUMN projects.catchphrase_en IS 'English translation of catchphrase';

-- Functional description translation
ALTER TABLE projects ADD COLUMN IF NOT EXISTS functional_description_en TEXT;
COMMENT ON COLUMN projects.functional_description_en IS 'English translation of functional description';

-- Technical description translation
ALTER TABLE projects ADD COLUMN IF NOT EXISTS technical_description_en TEXT;
COMMENT ON COLUMN projects.technical_description_en IS 'English translation of technical description';

-- ==============================================================================
-- INDEXES
-- ==============================================================================

-- Optional: Add indexes for common search patterns if needed
-- These are not critical for performance but can be added if text search is needed
-- CREATE INDEX IF NOT EXISTS idx_experiences_title_en ON experiences USING gin(to_tsvector('english', title_en));
-- CREATE INDEX IF NOT EXISTS idx_skills_name_en ON skills USING gin(to_tsvector('english', name_en));
-- CREATE INDEX IF NOT EXISTS idx_projects_title_en ON projects USING gin(to_tsvector('english', title_en));

-- ==============================================================================
-- NOTES
-- ==============================================================================

-- 1. All _en columns are NULLABLE by design:
--    - If NULL, the service will fall back to French (default)
--    - This allows gradual translation (don't need to translate everything at once)
--
-- 2. Translation strategy:
--    - Technical terms (technologies, tools) often stay the same in both languages
--    - Job titles and descriptions need professional, technical translation
--    - Keep the same level of detail and tone across languages
--
-- 3. Usage in Go services:
--    - A helper function will be created: getLocalizedField(frValue, enValue, lang)
--    - If lang == "en" and enValue != "", return enValue
--    - Otherwise, return frValue (French is the default/fallback)
--
-- 4. How to populate:
--    - Run seed_data_i18n.sql after this migration
--    - Or manually UPDATE records with translations
--
-- 5. This migration is IDEMPOTENT:
--    - Uses ADD COLUMN IF NOT EXISTS
--    - Can be run multiple times safely
--
-- 6. No data migration needed:
--    - Existing French data remains untouched
--    - English columns start as NULL
--    - Application falls back to French if English is NULL

-- ==============================================================================
-- VERIFICATION QUERIES
-- ==============================================================================

-- Check if columns were added successfully:
-- SELECT column_name, data_type, character_maximum_length
-- FROM information_schema.columns
-- WHERE table_name IN ('experiences', 'skills', 'projects')
--   AND column_name LIKE '%_en'
-- ORDER BY table_name, ordinal_position;

-- Count records with English translations:
-- SELECT
--     'experiences' as table_name,
--     COUNT(*) as total_records,
--     COUNT(title_en) as translated_records,
--     ROUND(100.0 * COUNT(title_en) / COUNT(*), 2) as translation_percentage
-- FROM experiences
-- UNION ALL
-- SELECT
--     'skills' as table_name,
--     COUNT(*) as total_records,
--     COUNT(name_en) as translated_records,
--     ROUND(100.0 * COUNT(name_en) / COUNT(*), 2) as translation_percentage
-- FROM skills
-- UNION ALL
-- SELECT
--     'projects' as table_name,
--     COUNT(*) as total_records,
--     COUNT(title_en) as translated_records,
--     ROUND(100.0 * COUNT(title_en) / COUNT(*), 2) as translation_percentage
-- FROM projects;
