-- Seed data i18n - English translations for maicivy CV
-- Updated: 2026-01-12
-- Description: Professional English translations for all experiences, skills, and projects
-- Note: This must be run AFTER seed_data.sql and add_i18n_fields.sql

-- ============================================================================
-- EXPERIENCES - English Translations
-- ============================================================================

-- Cogesco (2021-2024) - Senior position
UPDATE experiences
SET
    title_en = 'Versatile IT Developer (C++, VBA, SQL, .Net, Unity3D, AI)',
    description_en = 'Implementation of automation tools using Microsoft Access with rapid, iterative deployment. Creation of a customizable 3D demonstrator for product previsualization. Comprehensive IT support including AI-powered automated newsletters, servers, and network infrastructure.',
    catchphrase_en = NULL,  -- Will be added when available
    functional_description_en = NULL,  -- Will be added when available
    technical_description_en = NULL  -- Will be added when available
WHERE company = 'Cogesco' AND EXTRACT(YEAR FROM start_date) = 2021;

-- Taglabs (2018-2020)
UPDATE experiences
SET
    title_en = 'C++ / Unity Mobile Developer',
    description_en = 'Development of advanced CAD software using point cloud scanning technology (C++, Qt). Software architecture design and user interface creation. Parallel development of a complementary mobile application (Unity, C#). Innovative solutions for complex 3D data processing.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE company = 'Taglabs';

-- Alors Evidemment (2017) - Internship
UPDATE experiences
SET
    title_en = 'IT Developer Intern',
    description_en = 'Creation of mobile quiz applications with client-server communication, primarily using Unity3D in C#.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE company = 'Alors Evidemment';

-- Cogesco (2015) - First internship
UPDATE experiences
SET
    title_en = 'VBA Developer Intern',
    description_en = 'Creation and deployment of workforce and employee management software. Development of automated advertising tools for social networks. Generation of automated tests based on defined constraints. Microsoft Access with VBA and Automate 8.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE company = 'Cogesco' AND EXTRACT(YEAR FROM start_date) = 2015;

-- ============================================================================
-- SKILLS - English Translations
-- ============================================================================

-- Programming Languages

UPDATE skills SET name_en = 'Go', description_en = 'Backend development with Fiber framework, APIs, microservices'
WHERE name = 'Go';

UPDATE skills SET name_en = 'TypeScript', description_en = 'Full-stack development with type safety'
WHERE name = 'TypeScript';

UPDATE skills SET name_en = 'C++', description_en = 'Game engine development, systems programming'
WHERE name = 'C++';

UPDATE skills SET name_en = 'JavaScript', description_en = 'Web development, Node.js applications'
WHERE name = 'JavaScript';

UPDATE skills SET name_en = 'VBA', description_en = 'Office automation and macro development'
WHERE name = 'VBA';

UPDATE skills SET name_en = 'Python', description_en = 'Scripting, automation, data processing'
WHERE name = 'Python';

UPDATE skills SET name_en = 'C#', description_en = '.NET development'
WHERE name = 'C#';

UPDATE skills SET name_en = 'SQL', description_en = 'Database queries, optimization, T-SQL, PostgreSQL'
WHERE name = 'SQL';

-- Tools & Office

UPDATE skills SET name_en = 'Excel', description_en = 'Advanced formulas, pivot tables, Power Query'
WHERE name = 'Excel';

UPDATE skills SET name_en = 'Access', description_en = 'Access databases, forms, queries'
WHERE name = 'Access';

-- Methodologies

UPDATE skills SET name_en = 'Agile', description_en = 'Scrum, Kanban, sprints'
WHERE name = 'Agile';

UPDATE skills SET name_en = 'Legacy Migration', description_en = 'Migrating legacy systems to modern solutions'
WHERE name = 'Legacy Migration';

-- Frameworks

UPDATE skills SET name_en = 'Next.js', description_en = 'React framework with SSR, App Router'
WHERE name = 'Next.js';

UPDATE skills SET name_en = 'React', description_en = 'Component-based UI development'
WHERE name = 'React';

UPDATE skills SET name_en = 'Three.js', description_en = '3D graphics and WebGL'
WHERE name = 'Three.js';

UPDATE skills SET name_en = 'Node.js', description_en = 'Server-side JavaScript runtime'
WHERE name = 'Node.js';

-- Databases

UPDATE skills SET name_en = 'PostgreSQL', description_en = 'Relational database with advanced features'
WHERE name = 'PostgreSQL';

UPDATE skills SET name_en = 'Redis', description_en = 'In-memory caching and sessions'
WHERE name = 'Redis';

-- DevOps & Tools

UPDATE skills SET name_en = 'Docker', description_en = 'Containerization for development and production'
WHERE name = 'Docker';

UPDATE skills SET name_en = 'CMake', description_en = 'C++ build system and project management'
WHERE name = 'CMake';

UPDATE skills SET name_en = 'Git', description_en = 'Version control and collaboration'
WHERE name = 'Git';

-- AI & Special

UPDATE skills SET name_en = 'Claude API', description_en = 'Anthropic Claude integration for AI features'
WHERE name = 'Claude API';

UPDATE skills SET name_en = 'MCP (Model Context Protocol)', description_en = 'Tool integration protocol for AI assistants'
WHERE name = 'MCP (Model Context Protocol)';

UPDATE skills SET name_en = 'OpenAI API', description_en = 'GPT models integration'
WHERE name = 'OpenAI API';

-- ============================================================================
-- PROJECTS - English Translations
-- ============================================================================

-- maicivy
UPDATE projects
SET
    title_en = 'maicivy',  -- Project name stays the same
    description_en = 'Intelligent interactive CV with AI-powered cover letter generation. Modern stack with Next.js 14, Go, Three.js for 3D effects, PostgreSQL and Redis.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'maicivy';

-- GroveEngine
UPDATE projects
SET
    title_en = 'GroveEngine',  -- Project name stays the same
    description_en = 'Modular C++ engine with ultra-fast hot-reload system (0.4ms). Architecture optimized for development with Claude Code and rapid iteration.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'GroveEngine';

-- VBA MCP Server
UPDATE projects
SET
    title_en = 'VBA MCP Server',  -- Project name stays the same
    description_en = 'MCP server for extraction, analysis and injection of VBA code in Office files. 24 tools to automate Excel, Word and Access with Claude.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'VBA MCP Server';

-- Confluent
UPDATE projects
SET
    title_en = 'Confluent',  -- Project name stays the same
    description_en = 'Complete constructed language for an RPG universe. Linguistic system (67 roots, SOV grammar), multi-LLM translation API and real-time web interface.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'Confluent';

-- Freelance Dashboard
UPDATE projects
SET
    title_en = 'Freelance Dashboard',
    description_en = 'VBA MCP Demo - Excel dashboard for freelance tracking with KPIs, dynamic pivot tables and VBA automation.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'Freelance Dashboard';

-- TimeTrack Pro
UPDATE projects
SET
    title_en = 'TimeTrack Pro',
    description_en = 'VBA MCP Demo - Access time management system with hours tracking per client/project. Showcase of MCP server''s Access capabilities.',
    catchphrase_en = NULL,
    functional_description_en = NULL,
    technical_description_en = NULL
WHERE title = 'TimeTrack Pro';

-- ============================================================================
-- VERIFICATION & STATISTICS
-- ============================================================================

DO $$
DECLARE
    exp_total INTEGER;
    exp_translated INTEGER;
    skill_total INTEGER;
    skill_translated INTEGER;
    proj_total INTEGER;
    proj_translated INTEGER;
BEGIN
    -- Count experiences
    SELECT COUNT(*) INTO exp_total FROM experiences;
    SELECT COUNT(*) INTO exp_translated FROM experiences WHERE title_en IS NOT NULL;

    -- Count skills
    SELECT COUNT(*) INTO skill_total FROM skills;
    SELECT COUNT(*) INTO skill_translated FROM skills WHERE name_en IS NOT NULL;

    -- Count projects
    SELECT COUNT(*) INTO proj_total FROM projects;
    SELECT COUNT(*) INTO proj_translated FROM projects WHERE title_en IS NOT NULL;

    RAISE NOTICE '=======================================================';
    RAISE NOTICE 'i18n Seed Data Insertion Complete!';
    RAISE NOTICE '=======================================================';
    RAISE NOTICE 'Experiences: %/% translated (% %%)',
        exp_translated, exp_total, ROUND(100.0 * exp_translated / NULLIF(exp_total, 0), 2);
    RAISE NOTICE 'Skills: %/% translated (% %%)',
        skill_translated, skill_total, ROUND(100.0 * skill_translated / NULLIF(skill_total, 0), 2);
    RAISE NOTICE 'Projects: %/% translated (% %%)',
        proj_translated, proj_total, ROUND(100.0 * proj_translated / NULLIF(proj_total, 0), 2);
    RAISE NOTICE '=======================================================';
    RAISE NOTICE 'Notes:';
    RAISE NOTICE '- All main titles/names/descriptions have been translated';
    RAISE NOTICE '- Catchphrase, functional_description, technical_description';
    RAISE NOTICE '  are NULL and will be added when modal content is available';
    RAISE NOTICE '- French data remains as fallback for untranslated content';
    RAISE NOTICE '=======================================================';
END $$;

-- ============================================================================
-- TRANSLATION NOTES
-- ============================================================================

-- 1. Translation principles applied:
--    - Professional tone for developer CV
--    - Technical accuracy maintained across languages
--    - Industry-standard terminology used
--    - Concise yet comprehensive descriptions
--
-- 2. Items NOT translated (intentionally):
--    - Technology names (Go, C++, Unity3D, etc.)
--    - Framework names (Next.js, React, Three.js, etc.)
--    - Tool names (Docker, Git, CMake, etc.)
--    - Project names (maicivy, GroveEngine, Confluent, etc.)
--    - Company names (Cogesco, Taglabs, etc.)
--
-- 3. Fields left NULL:
--    - catchphrase_en: Short marketing phrases (not in original seed)
--    - functional_description_en: Detailed modal content (not in seed)
--    - technical_description_en: Detailed modal content (not in seed)
--    These can be added later when modal content is fully defined
--
-- 4. Quality assurance:
--    - All translations reviewed for accuracy
--    - Technical terms verified against industry standards
--    - Consistency checked across similar roles/projects
--    - Grammar and spelling validated
--
-- 5. Future updates:
--    - Add catchphrases when marketing content is defined
--    - Add detailed descriptions for project modals
--    - Update as CV content evolves
--    - Consider adding more languages (ES, DE, etc.)
