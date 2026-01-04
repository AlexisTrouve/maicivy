-- Seed data for maicivy CV
-- Updated: 2026-01-03
-- Description: Real CV data for Alexis - authentic projects and skills only

-- ============================================================================
-- EXPERIENCES (4 entries - Professional only)
-- ============================================================================

INSERT INTO experiences (id, title, company, description, start_date, end_date, technologies, tags, category, featured, created_at, updated_at) VALUES

-- ===================== PROFESSIONAL EXPERIENCES =====================

-- Cogesco - Senior position (2021-2024)
(
    gen_random_uuid(),
    'Développeur IT Polyvalent (C++, VBA, SQL, .Net, Unity3D, IA)',
    'Cogesco',
    'Implémentation d''outils d''automatisation avec Microsoft Access, déploiement rapide et itératif. Création d''un démonstrateur 3D personnalisable pour la prévisualisation de produits. Support IT complet : newsletters automatisées par IA, serveurs et infrastructure réseau.',
    '2021-01-01',
    '2024-08-31',
    ARRAY['C++', 'VBA', 'SQL', '.NET', 'Unity3D', 'Access', 'AI'],
    ARRAY['fullstack', 'automation', 'devops'],
    'fullstack',
    true,
    NOW(),
    NOW()
),

-- Taglabs (2018-2020)
(
    gen_random_uuid(),
    'Développeur C++ / Unity Mobile',
    'Taglabs',
    'Développement de logiciel CAD avancé utilisant la technologie de scan par nuage de points (C++, Qt). Conception d''architecture logicielle et création d''interface utilisateur. Développement parallèle d''une application mobile complémentaire (Unity, C#). Solutions innovantes pour le traitement de données 3D complexes.',
    '2018-09-01',
    '2020-10-31',
    ARRAY['C++', 'Qt', 'Unity', 'C#', 'Point Cloud', '3D'],
    ARRAY['fullstack', 'cpp', 'mobile', '3d'],
    'fullstack',
    true,
    NOW(),
    NOW()
),

-- Alors Evidemment - Internship (2017)
(
    gen_random_uuid(),
    'Stagiaire Développeur IT',
    'Alors Evidemment',
    'Création d''applications mobiles de quiz avec communication client-serveur, notamment en utilisant Unity3D en C#.',
    '2017-04-01',
    '2017-08-31',
    ARRAY['Unity3D', 'C#', 'Mobile'],
    ARRAY['fullstack', 'mobile', 'gamedev'],
    'fullstack',
    true,
    NOW(),
    NOW()
),

-- Cogesco - First internship (2015)
(
    gen_random_uuid(),
    'Stagiaire Développeur VBA',
    'Cogesco',
    'Création et déploiement de logiciels de gestion du travail et des employés. Création d''outils publicitaires automatisés pour les réseaux sociaux. Génération de tests automatiques basés sur des contraintes définies. Microsoft Access en VBA et Automate 8.',
    '2015-06-01',
    '2015-10-31',
    ARRAY['VBA', 'Access', 'Automate 8', 'SQL'],
    ARRAY['fullstack', 'automation', 'tools'],
    'fullstack',
    true,
    NOW(),
    NOW()
);

-- Note: Personal projects are in the PROJECTS table, not here

-- ============================================================================
-- SKILLS (17 entries - Real skills only)
-- ============================================================================

INSERT INTO skills (id, name, level, category, tags, years_experience, description, featured, icon, created_at, updated_at) VALUES

-- Languages
(
    gen_random_uuid(),
    'Go',
    'advanced',
    'Languages',
    ARRAY['backend', 'performance'],
    2,
    'Backend development with Fiber framework, APIs, microservices',
    true,
    'golang',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'TypeScript',
    'advanced',
    'Languages',
    ARRAY['frontend', 'backend'],
    3,
    'Full-stack development with type safety',
    true,
    'typescript',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'C++',
    'advanced',
    'Languages',
    ARRAY['systems', 'gamedev', 'performance'],
    3,
    'Game engine development, systems programming',
    true,
    'cplusplus',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'JavaScript',
    'advanced',
    'Languages',
    ARRAY['frontend', 'backend'],
    4,
    'Web development, Node.js applications',
    false,
    'javascript',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'VBA',
    'advanced',
    'Languages',
    ARRAY['automation', 'office'],
    2,
    'Office automation and macro development',
    false,
    'visualbasic',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Python',
    'intermediate',
    'Languages',
    ARRAY['scripting', 'automation', 'data'],
    2,
    'Scripting, automation, data processing',
    false,
    'python',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'C#',
    'intermediate',
    'Languages',
    ARRAY['dotnet', 'backend'],
    2,
    '.NET development',
    false,
    'csharp',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'SQL',
    'advanced',
    'Languages',
    ARRAY['database', 'data'],
    4,
    'Database queries, optimization, T-SQL, PostgreSQL',
    false,
    'database',
    NOW(),
    NOW()
),

-- Tools & Office
(
    gen_random_uuid(),
    'Excel',
    'expert',
    'Tools',
    ARRAY['office', 'data', 'automation'],
    5,
    'Formules avancées, tableaux croisés, Power Query',
    false,
    'excel',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Access',
    'advanced',
    'Tools',
    ARRAY['office', 'database'],
    3,
    'Bases de données Access, formulaires, requêtes',
    false,
    'access',
    NOW(),
    NOW()
),

-- Methodologies
(
    gen_random_uuid(),
    'Agile',
    'advanced',
    'Methodologies',
    ARRAY['methodology', 'scrum'],
    3,
    'Scrum, Kanban, sprints',
    false,
    'agile',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Legacy Migration',
    'expert',
    'Methodologies',
    ARRAY['migration', 'modernization'],
    4,
    'Migration de systèmes legacy vers solutions modernes',
    false,
    'migration',
    NOW(),
    NOW()
),

-- Frameworks
(
    gen_random_uuid(),
    'Next.js',
    'advanced',
    'Frameworks',
    ARRAY['frontend', 'fullstack'],
    2,
    'React framework with SSR, App Router',
    true,
    'nextjs',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'React',
    'advanced',
    'Frameworks',
    ARRAY['frontend'],
    3,
    'Component-based UI development',
    true,
    'react',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Three.js',
    'intermediate',
    'Frameworks',
    ARRAY['frontend', '3d', 'creative'],
    1,
    '3D graphics and WebGL',
    true,
    'threejs',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Node.js',
    'advanced',
    'Frameworks',
    ARRAY['backend'],
    3,
    'Server-side JavaScript runtime',
    false,
    'nodejs',
    NOW(),
    NOW()
),

-- Databases
(
    gen_random_uuid(),
    'PostgreSQL',
    'advanced',
    'Databases',
    ARRAY['backend', 'sql'],
    2,
    'Relational database with advanced features',
    true,
    'postgresql',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Redis',
    'intermediate',
    'Databases',
    ARRAY['backend', 'cache'],
    1,
    'In-memory caching and sessions',
    true,
    'redis',
    NOW(),
    NOW()
),

-- DevOps & Tools
(
    gen_random_uuid(),
    'Docker',
    'intermediate',
    'DevOps',
    ARRAY['devops', 'containers'],
    2,
    'Containerization for development and production',
    true,
    'docker',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'CMake',
    'advanced',
    'DevOps',
    ARRAY['cpp', 'build'],
    3,
    'C++ build system and project management',
    false,
    'cmake',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'Git',
    'advanced',
    'DevOps',
    ARRAY['devops', 'vcs'],
    5,
    'Version control and collaboration',
    false,
    'git',
    NOW(),
    NOW()
),

-- AI & Special
(
    gen_random_uuid(),
    'Claude API',
    'advanced',
    'AI',
    ARRAY['ai', 'llm'],
    1,
    'Anthropic Claude integration for AI features',
    true,
    'anthropic',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'MCP (Model Context Protocol)',
    'expert',
    'AI',
    ARRAY['ai', 'automation', 'tools'],
    1,
    'Tool integration protocol for AI assistants',
    true,
    'mcp',
    NOW(),
    NOW()
),
(
    gen_random_uuid(),
    'OpenAI API',
    'intermediate',
    'AI',
    ARRAY['ai', 'llm'],
    1,
    'GPT models integration',
    false,
    'openai',
    NOW(),
    NOW()
);

-- ============================================================================
-- PROJECTS (6 entries - Real projects only)
-- ============================================================================

INSERT INTO projects (id, title, description, github_url, demo_url, image_url, technologies, category, github_stars, github_forks, github_language, featured, in_progress, created_at, updated_at) VALUES

-- maicivy
(
    gen_random_uuid(),
    'maicivy',
    'CV interactif intelligent avec génération de lettres de motivation par IA. Stack moderne avec Next.js 14, Go, Three.js pour les effets 3D, PostgreSQL et Redis.',
    '',
    '',
    NULL,
    ARRAY['Next.js', 'Go', 'Three.js', 'PostgreSQL', 'Redis', 'Claude API'],
    'fullstack',
    0,
    0,
    'Go',
    true,
    true,
    NOW(),
    NOW()
),

-- GroveEngine
(
    gen_random_uuid(),
    'GroveEngine',
    'Moteur C++ modulaire avec système de hot-reload ultra-rapide (0.4ms). Architecture optimisée pour le développement avec Claude Code et itération rapide.',
    '',
    '',
    NULL,
    ARRAY['C++', 'CMake', 'ImGui', 'Hot-Reload', 'OpenGL'],
    'other',
    0,
    0,
    'C++',
    true,
    true,
    NOW(),
    NOW()
),

-- VBA MCP Server
(
    gen_random_uuid(),
    'VBA MCP Server',
    'Serveur MCP pour extraction, analyse et injection de code VBA dans les fichiers Office. 24 outils pour automatiser Excel, Word et Access avec Claude.',
    '',
    '',
    NULL,
    ARRAY['TypeScript', 'MCP', 'COM', 'Office', 'Node.js'],
    'devops',
    0,
    0,
    'TypeScript',
    true,
    false,
    NOW(),
    NOW()
),

-- Confluent
(
    gen_random_uuid(),
    'Confluent',
    'Langue construite complète pour un univers JDR. Système linguistique (67 racines, grammaire SOV), API de traduction multi-LLM et interface web temps réel.',
    '',
    '',
    NULL,
    ARRAY['Node.js', 'Claude API', 'OpenAI', 'Linguistics', 'React'],
    'other',
    0,
    0,
    'TypeScript',
    true,
    true,
    NOW(),
    NOW()
),

-- Freelance Dashboard (VBA MCP demo)
(
    gen_random_uuid(),
    'Freelance Dashboard',
    'Demo VBA MCP - Dashboard Excel pour suivi freelance avec KPIs, tableaux croisés dynamiques et automatisation VBA.',
    '',
    '',
    NULL,
    ARRAY['Excel', 'VBA', 'MCP'],
    'other',
    0,
    0,
    'VBA',
    true,
    false,
    NOW(),
    NOW()
),

-- TimeTrack Pro (VBA MCP demo)
(
    gen_random_uuid(),
    'TimeTrack Pro',
    'Demo VBA MCP - Gestionnaire de temps Access avec suivi heures par client/projet. Vitrine des capacités Access du serveur MCP.',
    '',
    '',
    NULL,
    ARRAY['Access', 'VBA', 'SQL', 'MCP'],
    'other',
    0,
    0,
    'VBA',
    true,
    false,
    NOW(),
    NOW()
);

-- ============================================================================
-- Summary Statistics
-- ============================================================================

DO $$
DECLARE
    exp_count INTEGER;
    skill_count INTEGER;
    proj_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO exp_count FROM experiences;
    SELECT COUNT(*) INTO skill_count FROM skills;
    SELECT COUNT(*) INTO proj_count FROM projects;

    RAISE NOTICE '==============================================';
    RAISE NOTICE 'Seed Data Insertion Complete!';
    RAISE NOTICE '==============================================';
    RAISE NOTICE 'Experiences inserted: %', exp_count;
    RAISE NOTICE 'Skills inserted: %', skill_count;
    RAISE NOTICE 'Projects inserted: %', proj_count;
    RAISE NOTICE '==============================================';
END $$;
