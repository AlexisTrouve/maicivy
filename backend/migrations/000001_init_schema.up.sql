-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: experiences
CREATE TABLE IF NOT EXISTS experiences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    description TEXT,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    technologies TEXT[],
    tags TEXT[],
    category VARCHAR(100) NOT NULL,
    featured BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category pour filtrage
CREATE INDEX idx_experiences_category ON experiences(category);
CREATE INDEX idx_experiences_deleted_at ON experiences(deleted_at);

-- Table: skills
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    level VARCHAR(20) NOT NULL CHECK (level IN ('beginner', 'intermediate', 'advanced', 'expert')),
    category VARCHAR(100) NOT NULL,
    tags TEXT[],
    years_experience INTEGER DEFAULT 0,
    description TEXT,
    featured BOOLEAN DEFAULT FALSE,
    icon VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category et level
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_level ON skills(level);
CREATE INDEX idx_skills_deleted_at ON skills(deleted_at);

-- Table: projects
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    github_url VARCHAR(500),
    demo_url VARCHAR(500),
    image_url VARCHAR(500),
    technologies TEXT[],
    category VARCHAR(100) NOT NULL,
    github_stars INTEGER DEFAULT 0,
    github_forks INTEGER DEFAULT 0,
    github_language VARCHAR(50),
    featured BOOLEAN DEFAULT FALSE,
    in_progress BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category et featured
CREATE INDEX idx_projects_category ON projects(category);
CREATE INDEX idx_projects_featured ON projects(featured);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);

-- Table: visitors
CREATE TABLE IF NOT EXISTS visitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    ip_hash VARCHAR(64),
    user_agent TEXT,
    browser VARCHAR(100),
    os VARCHAR(100),
    device VARCHAR(50),
    visit_count INTEGER DEFAULT 1,
    first_visit TIMESTAMP NOT NULL,
    last_visit TIMESTAMP NOT NULL,
    profile_detected VARCHAR(50) DEFAULT 'unknown',
    company_name VARCHAR(255),
    linkedin_url VARCHAR(500),
    country VARCHAR(100),
    city VARCHAR(100),
    timezone VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur session_id (lookup primaire), ip_hash, profile
CREATE UNIQUE INDEX idx_visitors_session_id ON visitors(session_id);
CREATE INDEX idx_visitors_ip_hash ON visitors(ip_hash);
CREATE INDEX idx_visitors_profile_detected ON visitors(profile_detected);
CREATE INDEX idx_visitors_deleted_at ON visitors(deleted_at);

-- Table: generated_letters
CREATE TABLE IF NOT EXISTS generated_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    letter_type VARCHAR(20) NOT NULL CHECK (letter_type IN ('motivation', 'anti_motivation')),
    content TEXT NOT NULL,
    ai_model VARCHAR(50),
    tokens_used INTEGER DEFAULT 0,
    generation_ms INTEGER DEFAULT 0,
    company_info JSONB,
    downloaded BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur visitor_id et letter_type
CREATE INDEX idx_generated_letters_visitor_id ON generated_letters(visitor_id);
CREATE INDEX idx_generated_letters_letter_type ON generated_letters(letter_type);
CREATE INDEX idx_generated_letters_created_at ON generated_letters(created_at DESC);
CREATE INDEX idx_generated_letters_deleted_at ON generated_letters(deleted_at);

-- Table: analytics_events
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    page_url VARCHAR(500),
    referrer VARCHAR(500),
    session_duration INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur visitor_id, event_type et created_at (analytics queries)
CREATE INDEX idx_analytics_events_visitor_id ON analytics_events(visitor_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at DESC);
CREATE INDEX idx_analytics_events_deleted_at ON analytics_events(deleted_at);

-- Index composite pour requêtes fréquentes
CREATE INDEX idx_analytics_events_type_date ON analytics_events(event_type, created_at DESC);

-- Trigger pour updated_at automatique
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_experiences_updated_at BEFORE UPDATE ON experiences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_skills_updated_at BEFORE UPDATE ON skills FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_visitors_updated_at BEFORE UPDATE ON visitors FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_generated_letters_updated_at BEFORE UPDATE ON generated_letters FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_analytics_events_updated_at BEFORE UPDATE ON analytics_events FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
