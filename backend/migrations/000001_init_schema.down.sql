-- Rollback de la migration initiale
DROP TRIGGER IF EXISTS update_analytics_events_updated_at ON analytics_events;
DROP TRIGGER IF EXISTS update_generated_letters_updated_at ON generated_letters;
DROP TRIGGER IF EXISTS update_visitors_updated_at ON visitors;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_skills_updated_at ON skills;
DROP TRIGGER IF EXISTS update_experiences_updated_at ON experiences;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS analytics_events;
DROP TABLE IF EXISTS generated_letters;
DROP TABLE IF EXISTS visitors;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS experiences;

DROP EXTENSION IF EXISTS "uuid-ossp";
