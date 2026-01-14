package main

import (
	"log"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/database"
	"maicivy/internal/models"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// makeLinks is a helper to create LinksJSON from link data
func makeLinks(links ...models.LinkData) models.LinksJSON {
	return models.LinksJSON(links)
}

func main() {
	// Charger .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Charger config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connexion DB
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	log.Println("🌱 Starting database seeding...")

	seedExperiences(db)
	seedSkills(db)
	seedProjects(db)

	log.Println("✅ Database seeding completed!")
}

func seedExperiences(db *gorm.DB) {
	// First, check if experiences already exist - skip seeding if data exists
	var count int64
	db.Model(&models.Experience{}).Count(&count)
	if count > 0 {
		log.Printf("⏭️  Skipping experiences seeding - %d entries already exist", count)
		return
	}

	experiences := []models.Experience{
		{
			Title:       "Full-Stack & AI Developer",
			Company:     "Freelance / Personal Projects",
			Description: "Development of innovative full-stack solutions integrating AI. Creation of maicivy (interactive CV with AI letter generation), MCP servers for Office automation, and productivity tools.",
			Catchphrase: "Building the future of AI-powered development tools",
			FunctionalDescription: `As an independent developer, I design and build complete applications that leverage the latest AI technologies to solve real-world problems.

Key achievements:
- Created maicivy, an interactive CV platform with AI-powered cover letter generation
- Developed MCP (Model Context Protocol) servers for seamless AI-Office integration
- Built automation tools that save hours of manual work daily
- Contributed to open-source projects in the AI/LLM ecosystem`,
			TechnicalDescription: `Technical stack and architecture decisions:

**Backend (Go + Fiber):**
- RESTful API design with proper error handling and validation
- PostgreSQL for persistent data with GORM ORM
- Redis for caching, rate limiting, and session management
- Claude and OpenAI API integrations with proper prompt engineering

**Frontend (Next.js 14):**
- App Router with server components for optimal performance
- Tailwind CSS with dark mode support
- Three.js for immersive 3D effects
- Framer Motion for smooth animations

**DevOps:**
- Docker containerization with multi-stage builds
- CI/CD with GitHub Actions
- Prometheus + Grafana monitoring`,
			StartDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"Go", "Next.js", "TypeScript", "PostgreSQL", "Redis", "Claude API", "Three.js"},
			Tags:         pq.StringArray{"fullstack", "ai", "devops"},
			Category:     "fullstack",
			Featured:     true,
			Images:       pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/maicivy", Icon: "github"},
				models.LinkData{Name: "LinkedIn", URL: "https://linkedin.com/in/alexi", Icon: "linkedin"},
			),
		},
		{
			Title:       "C++ / Game Engine Developer",
			Company:     "Personal Projects",
			Description: "Design and development of GroveEngine, a modular C++ engine with ultra-fast hot-reload system (0.4ms). Architecture optimized for rapid iteration with Claude Code.",
			Catchphrase: "Pushing the boundaries of real-time code iteration",
			FunctionalDescription: `GroveEngine is a custom game engine built from the ground up with a focus on developer experience and rapid iteration.

Key features:
- Ultra-fast hot-reload system that reloads code changes in 0.4ms
- Modular architecture allowing easy extension and customization
- Integrated editor with ImGui for real-time debugging
- AI-assisted development workflow with Claude Code integration

The engine is designed for creating 2D/3D games and interactive applications with minimal compile-time friction.`,
			TechnicalDescription: `Architecture and technical details:

**Core Engine:**
- Modern C++20 with concepts and modules
- Custom ECS (Entity Component System) for game objects
- Hot-reload via DLL/SO unloading with state preservation
- Memory pool allocators for performance-critical paths

**Build System:**
- CMake with precompiled headers for fast builds
- Incremental compilation with dependency tracking
- Cross-platform support (Windows, Linux, macOS)

**Rendering:**
- OpenGL 4.6 / Vulkan abstraction layer
- ImGui integration for debug UI and editor
- Custom shader hot-reload system

**Performance:**
- 0.4ms average hot-reload time
- 60+ FPS with complex scenes
- Minimal memory fragmentation`,
			StartDate:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"C++", "CMake", "ImGui", "Hot-Reload", "OpenGL"},
			Tags:         pq.StringArray{"cpp", "gamedev", "engine"},
			Category:     "other",
			Featured:     true,
			Images:       pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/groveengine", Icon: "github"},
			),
		},
		{
			Title:       "Tools & Automation Developer",
			Company:     "Open Source Projects",
			Description: "Creation of VBA MCP Server for extraction, analysis, and injection of VBA code in Office files. 24 tools to automate Excel, Word, and Access with Claude.",
			Catchphrase: "Bridging AI and Office automation",
			FunctionalDescription: `The VBA MCP Server enables AI assistants like Claude to interact directly with Microsoft Office applications, revolutionizing how we automate business workflows.

Key capabilities:
- Extract VBA code from Excel, Word, and Access files
- Analyze and understand existing VBA modules
- Inject new or modified VBA code into Office documents
- Provide 24 specialized tools for common automation tasks

Use cases include automated report generation, data processing pipelines, and legacy system integration.`,
			TechnicalDescription: `Technical implementation:

**MCP Server (TypeScript/Node.js):**
- Model Context Protocol implementation for Claude integration
- Async/await patterns for non-blocking operations
- Comprehensive error handling and recovery

**COM Interop:**
- Direct communication with Office applications via COM
- Support for Office 2016+ (Windows)
- Graceful degradation for missing components

**VBA Processing:**
- AST parsing for VBA code analysis
- Code injection with proper module management
- Support for forms, classes, and standard modules

**Tools Provided:**
- File operations (open, save, export)
- Code management (read, write, delete modules)
- Macro execution and debugging
- Sheet/document manipulation`,
			StartDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"TypeScript", "MCP", "COM", "Office", "Node.js"},
			Tags:         pq.StringArray{"devops", "automation", "tools"},
			Category:     "devops",
			Featured:     true,
			Images:       pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/vba-mcp-server", Icon: "github"},
				models.LinkData{Name: "NPM", URL: "https://npmjs.com/package/vba-mcp-server", Icon: "npm"},
			),
		},
		{
			Title:       "Language Creator & Linguistic Systems",
			Company:     "Confluent Project",
			Description: "Design of a complete constructed language for a tabletop RPG universe. Linguistic system with 67 roots, SOV grammar, multi-LLM translation API, and real-time web interface.",
			Catchphrase: "Creating languages that tell stories",
			FunctionalDescription: `Confluent is a conlang (constructed language) project designed for an immersive tabletop RPG experience.

Language features:
- 67 root words that combine to form complex vocabulary
- SOV (Subject-Object-Verb) grammar structure
- Agglutinative morphology with clear rules
- Writing system with custom glyphs

The project includes a web-based translator that uses multiple LLMs to understand context and produce accurate translations, making the language accessible to players without linguistic expertise.`,
			TechnicalDescription: `Technical components:

**Language Engine:**
- Rule-based morphological analyzer
- Custom lexicon database with etymology tracking
- Phonological rules for pronunciation

**Translation API:**
- Multi-LLM approach (Claude, GPT-4) for context understanding
- Fallback chain for reliability
- Caching layer for common phrases

**Web Interface (React):**
- Real-time translation with debouncing
- Interactive grammar tutorials
- Vocabulary explorer with audio pronunciation
- Character/glyph input system

**Backend (Node.js):**
- REST API for translation services
- WebSocket for real-time features
- Rate limiting and usage tracking`,
			StartDate:    time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"Node.js", "Claude API", "OpenAI", "Linguistics", "React"},
			Tags:         pq.StringArray{"ai", "linguistics", "creative"},
			Category:     "other",
			Featured:     false,
			Images:       pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "Demo", URL: "https://confluent-lang.dev", Icon: "globe"},
			),
		},
	}

	for _, exp := range experiences {
		if err := db.Create(&exp).Error; err != nil {
			log.Printf("Failed to seed experience %s: %v", exp.Title, err)
		} else {
			log.Printf("Seeded experience: %s at %s", exp.Title, exp.Company)
		}
	}
}

func seedSkills(db *gorm.DB) {
	// First, check if skills already exist - skip seeding if data exists
	var count int64
	db.Model(&models.Skill{}).Count(&count)
	if count > 0 {
		log.Printf("⏭️  Skipping skills seeding - %d entries already exist", count)
		return
	}

	skills := []models.Skill{
		// Languages
		{Name: "Go", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"backend", "performance"}, YearsExperience: 2, Icon: "golang", Featured: true},
		{Name: "TypeScript", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"frontend", "backend"}, YearsExperience: 3, Icon: "typescript", Featured: true},
		{Name: "C++", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"systems", "gamedev", "performance"}, YearsExperience: 3, Icon: "cplusplus", Featured: true},
		{Name: "JavaScript", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"frontend", "backend"}, YearsExperience: 4, Icon: "javascript", Featured: false},
		{Name: "VBA", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"automation", "office"}, YearsExperience: 2, Icon: "visualbasic", Featured: false},

		// Frameworks
		{Name: "Next.js", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend", "fullstack"}, YearsExperience: 2, Icon: "nextjs", Featured: true},
		{Name: "React", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend"}, YearsExperience: 3, Icon: "react", Featured: true},
		{Name: "Three.js", Level: models.SkillLevelIntermediate, Category: "Frameworks", Tags: pq.StringArray{"frontend", "3d", "creative"}, YearsExperience: 1, Icon: "threejs", Featured: true},
		{Name: "Node.js", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"backend"}, YearsExperience: 3, Icon: "nodejs", Featured: false},

		// Databases
		{Name: "PostgreSQL", Level: models.SkillLevelAdvanced, Category: "Databases", Tags: pq.StringArray{"backend", "sql"}, YearsExperience: 2, Icon: "postgresql", Featured: true},
		{Name: "Redis", Level: models.SkillLevelIntermediate, Category: "Databases", Tags: pq.StringArray{"backend", "cache"}, YearsExperience: 1, Icon: "redis", Featured: true},

		// DevOps & Tools
		{Name: "Docker", Level: models.SkillLevelIntermediate, Category: "DevOps", Tags: pq.StringArray{"devops", "containers"}, YearsExperience: 2, Icon: "docker", Featured: true},
		{Name: "CMake", Level: models.SkillLevelAdvanced, Category: "DevOps", Tags: pq.StringArray{"cpp", "build"}, YearsExperience: 3, Icon: "cmake", Featured: false},
		{Name: "Git", Level: models.SkillLevelAdvanced, Category: "DevOps", Tags: pq.StringArray{"devops", "vcs"}, YearsExperience: 5, Icon: "git", Featured: false},

		// AI & Special
		{Name: "Claude API", Level: models.SkillLevelAdvanced, Category: "AI", Tags: pq.StringArray{"ai", "llm"}, YearsExperience: 1, Icon: "anthropic", Featured: true},
		{Name: "MCP (Model Context Protocol)", Level: models.SkillLevelExpert, Category: "AI", Tags: pq.StringArray{"ai", "automation", "tools"}, YearsExperience: 1, Icon: "mcp", Featured: true},
		{Name: "OpenAI API", Level: models.SkillLevelIntermediate, Category: "AI", Tags: pq.StringArray{"ai", "llm"}, YearsExperience: 1, Icon: "openai", Featured: false},
	}

	for _, skill := range skills {
		if err := db.Create(&skill).Error; err != nil {
			log.Printf("⚠️  Failed to seed skill %s: %v", skill.Name, err)
		} else {
			log.Printf("✅ Seeded skill: %s (%s)", skill.Name, skill.Level)
		}
	}
}

func seedProjects(db *gorm.DB) {
	// First, check if projects already exist - skip seeding if data exists
	var count int64
	db.Model(&models.Project{}).Count(&count)
	if count > 0 {
		log.Printf("⏭️  Skipping projects seeding - %d entries already exist", count)
		return
	}

	projects := []models.Project{
		{
			Title:       "maicivy",
			Description: "Interactive CV with AI-powered cover letter generation. Modern stack with Next.js 14, Go, Three.js for 3D effects, PostgreSQL and Redis.",
			Catchphrase: "Your CV, reimagined with AI",
			FunctionalDescription: `maicivy is an innovative interactive CV platform that goes beyond traditional resume formats.

Key features:
- Dynamic CV that adapts to viewer interests (themes: Backend, Frontend, DevOps, AI)
- AI-powered cover letter generation using Claude and GPT-4
- Real-time analytics dashboard showing visitor interactions
- Immersive 3D effects with Three.js

The dual-letter generation feature creates both a professional motivation letter AND an "anti-motivation letter" - a humorous take that showcases personality while maintaining professionalism.`,
			TechnicalDescription: `Full-stack architecture:

**Backend (Go + Fiber):**
- High-performance REST API with Fiber framework
- GORM for PostgreSQL with optimized queries
- Redis for caching, sessions, and rate limiting
- WebSocket support for real-time analytics
- Claude and OpenAI integrations with streaming

**Frontend (Next.js 14):**
- App Router with React Server Components
- Tailwind CSS with dark/light mode
- Three.js for interactive 3D background
- Framer Motion for fluid animations
- shadcn/ui component library

**Infrastructure:**
- Docker Compose for local development
- Nginx reverse proxy with SSL
- Prometheus + Grafana monitoring
- GitHub Actions CI/CD pipeline`,
			GithubURL:      "",
			DemoURL:        "",
			Technologies:   pq.StringArray{"Next.js", "Go", "Three.js", "PostgreSQL", "Redis", "Claude API"},
			Category:       "fullstack",
			GithubStars:    0,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     true,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/maicivy/maicivy", Icon: "github"},
				models.LinkData{Name: "Live Demo", URL: "https://maicivy.dev", Icon: "globe"},
			),
		},
		{
			Title:       "GroveEngine",
			Description: "Modular C++ engine with ultra-fast hot-reload system (0.4ms). Architecture optimized for development with Claude Code and rapid iteration.",
			Catchphrase: "Code fast, iterate faster",
			FunctionalDescription: `GroveEngine is a game engine built for the modern AI-assisted development workflow.

Core philosophy:
- Minimal friction between code changes and seeing results
- Modular design where each system can be developed independently
- First-class debugging and profiling tools
- Designed to work seamlessly with AI coding assistants

The 0.4ms hot-reload time means you can see your changes almost instantly, making the development feedback loop incredibly tight.`,
			TechnicalDescription: `Engine architecture:

**Hot-Reload System:**
- Custom DLL/SO loader with state serialization
- Automatic dependency tracking between modules
- Zero-copy where possible for large data structures
- Graceful fallback on reload failures

**Core Systems:**
- ECS (Entity Component System) with archetypal storage
- Job system for parallel processing
- Custom allocators (linear, pool, stack)
- Event system with compile-time type safety

**Rendering Pipeline:**
- OpenGL 4.6 with compute shader support
- Vulkan backend in development
- ImGui integration for editor and debug UI
- Custom shader cross-compiler (GLSL/HLSL/SPIR-V)

**Tools:**
- Scene editor with undo/redo
- Performance profiler with flame graphs
- Memory allocation tracker
- Live shader editing`,
			GithubURL:      "",
			Technologies:   pq.StringArray{"C++", "CMake", "ImGui", "Hot-Reload", "OpenGL"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "C++",
			Featured:       true,
			InProgress:     true,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/groveengine", Icon: "github"},
			),
		},
		{
			Title:       "VBA MCP Server",
			Description: "MCP server for extraction, analysis, and injection of VBA code in Office files. 24 tools to automate Excel, Word, and Access with Claude.",
			Catchphrase: "AI meets Office automation",
			FunctionalDescription: `The VBA MCP Server bridges the gap between modern AI assistants and legacy Office automation.

Capabilities:
- Extract VBA code from any Office document
- Analyze existing macros and understand their logic
- Generate new VBA code with Claude's assistance
- Inject code back into Office files
- Execute macros and capture results

Use cases include:
- Modernizing legacy Excel applications
- Automating repetitive Office tasks
- Debugging complex VBA codebases
- Teaching VBA through AI-assisted explanations`,
			TechnicalDescription: `Technical stack:

**MCP Implementation:**
- Full Model Context Protocol support
- 24 specialized tools for Office interaction
- Streaming responses for long operations
- Comprehensive error handling

**Office Integration:**
- COM automation for Excel, Word, Access
- Support for .xlsm, .docm, .accdb formats
- Module, class, and form handling
- Named range and table operations

**Code Processing:**
- VBA tokenizer and parser
- Syntax highlighting for responses
- Code formatting and linting
- Dependency analysis between modules

**Security:**
- Sandboxed macro execution
- Permission system for dangerous operations
- Audit logging of all changes`,
			GithubURL:      "",
			Technologies:   pq.StringArray{"TypeScript", "MCP", "COM", "Office", "Node.js"},
			Category:       "devops",
			GithubStars:    0,
			GithubLanguage: "TypeScript",
			Featured:       true,
			InProgress:     false,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "GitHub", URL: "https://github.com/vba-mcp-server", Icon: "github"},
				models.LinkData{Name: "NPM Package", URL: "https://npmjs.com/package/vba-mcp-server", Icon: "npm"},
				models.LinkData{Name: "Documentation", URL: "https://vba-mcp-server.dev/docs", Icon: "book"},
			),
		},
		{
			Title:       "Confluent",
			Description: "Complete constructed language for a tabletop RPG universe. Linguistic system (67 roots, SOV grammar), multi-LLM translation API, and real-time web interface.",
			Catchphrase: "A language that breathes life into worlds",
			FunctionalDescription: `Confluent is more than a conlang - it's a complete linguistic ecosystem for immersive storytelling.

Language features:
- 67 semantic roots that combine following strict rules
- SOV (Subject-Object-Verb) word order
- Agglutinative morphology with prefixes and suffixes
- Tonal system for emotional nuance
- Custom script with calligraphic variants

The web interface allows:
- Real-time translation with context awareness
- Grammar lessons with interactive exercises
- Vocabulary builder with spaced repetition
- Community dictionary contributions`,
			TechnicalDescription: `System architecture:

**Language Engine:**
- Finite-state transducers for morphology
- Context-free grammar for syntax
- Custom phonological rule engine
- Etymology database with historical forms

**Translation Service:**
- Multi-LLM orchestration (Claude, GPT-4)
- Semantic embedding for context matching
- Fallback cascade for reliability
- Response caching with invalidation

**Web Application:**
- React with TypeScript
- Real-time updates via WebSocket
- Audio synthesis for pronunciation
- SVG rendering for custom script

**API:**
- RESTful endpoints for translation
- GraphQL for complex queries
- Rate limiting per user tier
- Webhook support for integrations`,
			GithubURL:      "",
			Technologies:   pq.StringArray{"Node.js", "Claude API", "OpenAI", "Linguistics", "React"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "TypeScript",
			Featured:       true,
			InProgress:     true,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "Live Translator", URL: "https://confluent-lang.dev", Icon: "globe"},
				models.LinkData{Name: "Documentation", URL: "https://confluent-lang.dev/docs", Icon: "book"},
			),
		},
		{
			Title:       "Freelance Dashboard",
			Description: "VBA MCP Demo - Excel dashboard for freelance tracking with KPIs, pivot tables, and VBA automation.",
			Catchphrase: "Track your freelance business like a pro",
			FunctionalDescription: `A comprehensive Excel dashboard created as a demonstration of VBA MCP Server capabilities.

Features:
- Income and expense tracking with categories
- Client management with contact history
- Project timeline with Gantt-like visualization
- Invoice generation with customizable templates
- KPI dashboard with charts and trends
- Pivot tables for data analysis

This project showcases how AI can assist in building complex Excel applications with proper VBA architecture.`,
			TechnicalDescription: `Excel architecture:

**Data Layer:**
- Structured tables for all data
- Data validation rules
- Named ranges for formulas
- Connection to external data sources

**VBA Modules:**
- Form handling for data entry
- Report generation automation
- Email integration for invoices
- Backup and restore functionality

**Dashboard:**
- Interactive charts with slicers
- Conditional formatting for KPIs
- Sparklines for trends
- Auto-refresh on data changes

Built entirely through VBA MCP Server to demonstrate AI-assisted Office development.`,
			GithubURL:      "",
			Technologies:   pq.StringArray{"Excel", "VBA", "MCP"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "VBA",
			Featured:       false,
			InProgress:     false,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "Download", URL: "https://github.com/vba-mcp-server/demos/freelance-dashboard", Icon: "download"},
			),
		},
		{
			Title:       "TimeTrack Pro",
			Description: "VBA MCP Demo - Access time manager with client/project hour tracking. Showcase of MCP Server Access capabilities.",
			Catchphrase: "Every minute counts",
			FunctionalDescription: `TimeTrack Pro is an Access database application demonstrating the VBA MCP Server's Access integration.

Core features:
- Time entry with start/stop timer
- Client and project hierarchy
- Billable vs non-billable tracking
- Report generation (daily, weekly, monthly)
- Invoice preparation based on time entries
- Multi-user support with permissions

Designed to show how AI can build and modify Access applications through the MCP protocol.`,
			TechnicalDescription: `Access architecture:

**Database Design:**
- Normalized schema (3NF)
- Referential integrity constraints
- Indexed queries for performance
- Audit trail tables

**Forms:**
- Time entry with real-time calculation
- Client/project management
- Report parameter dialogs
- Admin settings panel

**Reports:**
- Crystal-style report layouts
- Grouped summaries
- Chart integration
- Export to PDF/Excel

**VBA:**
- Timer class module
- Custom validation functions
- Email integration
- Backup automation

All code generated and modified through VBA MCP Server interaction with Claude.`,
			GithubURL:      "",
			Technologies:   pq.StringArray{"Access", "VBA", "SQL", "MCP"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "VBA",
			Featured:       false,
			InProgress:     false,
			Images:         pq.StringArray{},
			Links: makeLinks(
				models.LinkData{Name: "Download", URL: "https://github.com/vba-mcp-server/demos/timetrack-pro", Icon: "download"},
			),
		},
	}

	for _, project := range projects {
		if err := db.Create(&project).Error; err != nil {
			log.Printf("Failed to seed project %s: %v", project.Title, err)
		} else {
			log.Printf("Seeded project: %s", project.Title)
		}
	}
}

// Helper pour créer *time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
