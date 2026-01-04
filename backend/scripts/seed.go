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
	experiences := []models.Experience{
		{
			Title:        "Développeur Full-Stack & IA",
			Company:      "Freelance / Projets Personnels",
			Description:  "Développement de solutions full-stack innovantes intégrant l'IA. Création de maicivy (CV interactif avec génération de lettres IA), serveurs MCP pour automatisation Office, et outils de productivité.",
			StartDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil, // Emploi actuel
			Technologies: pq.StringArray{"Go", "Next.js", "TypeScript", "PostgreSQL", "Redis", "Claude API", "Three.js"},
			Tags:         pq.StringArray{"fullstack", "ai", "devops"},
			Category:     "fullstack",
			Featured:     true,
		},
		{
			Title:        "Développeur C++ / Game Engine",
			Company:      "Projets Personnels",
			Description:  "Conception et développement de GroveEngine, un moteur C++ modulaire avec système de hot-reload ultra-rapide (0.4ms). Architecture optimisée pour l'itération rapide avec Claude Code.",
			StartDate:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"C++", "CMake", "ImGui", "Hot-Reload", "OpenGL"},
			Tags:         pq.StringArray{"cpp", "gamedev", "engine"},
			Category:     "other",
			Featured:     true,
		},
		{
			Title:        "Développeur Outils & Automatisation",
			Company:      "Projets Open Source",
			Description:  "Création du VBA MCP Server pour l'extraction, analyse et injection de code VBA dans les fichiers Office. 24 outils pour automatiser Excel, Word et Access avec Claude.",
			StartDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"TypeScript", "MCP", "COM", "Office", "Node.js"},
			Tags:         pq.StringArray{"devops", "automation", "tools"},
			Category:     "devops",
			Featured:     true,
		},
		{
			Title:        "Créateur de Langue & Systèmes Linguistiques",
			Company:      "Projet Confluent",
			Description:  "Conception d'une langue construite complète pour un univers JDR. Système linguistique avec 67 racines, grammaire SOV, API de traduction multi-LLM et interface web temps réel.",
			StartDate:    time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil,
			Technologies: pq.StringArray{"Node.js", "Claude API", "OpenAI", "Linguistics", "React"},
			Tags:         pq.StringArray{"ai", "linguistics", "creative"},
			Category:     "other",
			Featured:     false,
		},
	}

	for _, exp := range experiences {
		if err := db.Create(&exp).Error; err != nil {
			log.Printf("⚠️  Failed to seed experience %s: %v", exp.Title, err)
		} else {
			log.Printf("✅ Seeded experience: %s at %s", exp.Title, exp.Company)
		}
	}
}

func seedSkills(db *gorm.DB) {
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
	projects := []models.Project{
		{
			Title:          "maicivy",
			Description:    "CV interactif intelligent avec génération de lettres de motivation par IA. Stack moderne avec Next.js 14, Go, Three.js pour les effets 3D, PostgreSQL et Redis.",
			GithubURL:      "",
			DemoURL:        "",
			Technologies:   pq.StringArray{"Next.js", "Go", "Three.js", "PostgreSQL", "Redis", "Claude API"},
			Category:       "fullstack",
			GithubStars:    0,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "GroveEngine",
			Description:    "Moteur C++ modulaire avec système de hot-reload ultra-rapide (0.4ms). Architecture optimisée pour le développement avec Claude Code et itération rapide.",
			GithubURL:      "",
			Technologies:   pq.StringArray{"C++", "CMake", "ImGui", "Hot-Reload", "OpenGL"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "C++",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "VBA MCP Server",
			Description:    "Serveur MCP pour extraction, analyse et injection de code VBA dans les fichiers Office. 24 outils pour automatiser Excel, Word et Access avec Claude.",
			GithubURL:      "",
			Technologies:   pq.StringArray{"TypeScript", "MCP", "COM", "Office", "Node.js"},
			Category:       "devops",
			GithubStars:    0,
			GithubLanguage: "TypeScript",
			Featured:       true,
			InProgress:     false,
		},
		{
			Title:          "Confluent",
			Description:    "Langue construite complète pour un univers JDR. Système linguistique (67 racines, grammaire SOV), API de traduction multi-LLM et interface web temps réel.",
			GithubURL:      "",
			Technologies:   pq.StringArray{"Node.js", "Claude API", "OpenAI", "Linguistics", "React"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "TypeScript",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "Freelance Dashboard",
			Description:    "Demo VBA MCP - Dashboard Excel pour suivi freelance avec KPIs, tableaux croisés dynamiques et automatisation VBA.",
			GithubURL:      "",
			Technologies:   pq.StringArray{"Excel", "VBA", "MCP"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "VBA",
			Featured:       false,
			InProgress:     false,
		},
		{
			Title:          "TimeTrack Pro",
			Description:    "Demo VBA MCP - Gestionnaire de temps Access avec suivi heures par client/projet. Vitrine des capacités Access du serveur MCP.",
			GithubURL:      "",
			Technologies:   pq.StringArray{"Access", "VBA", "SQL", "MCP"},
			Category:       "other",
			GithubStars:    0,
			GithubLanguage: "VBA",
			Featured:       false,
			InProgress:     false,
		},
	}

	for _, project := range projects {
		if err := db.Create(&project).Error; err != nil {
			log.Printf("⚠️  Failed to seed project %s: %v", project.Title, err)
		} else {
			log.Printf("✅ Seeded project: %s", project.Title)
		}
	}
}

// Helper pour créer *time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
