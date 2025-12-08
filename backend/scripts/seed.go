package main

import (
	"log"
	"os"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/database"
	"maicivy/internal/models"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
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

func seedExperiences(db interface{ Create(interface{}) error }) {
	experiences := []models.Experience{
		{
			Title:        "Senior Backend Developer",
			Company:      "TechCorp Inc.",
			Description:  "Led backend architecture migration from monolith to microservices using Go and Kubernetes.",
			StartDate:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil, // Emploi actuel
			Technologies: pq.StringArray{"Go", "PostgreSQL", "Redis", "Kubernetes", "Docker"},
			Tags:         pq.StringArray{"backend", "microservices", "devops"},
			Category:     "backend",
			Featured:     true,
		},
		{
			Title:        "Full-Stack Developer",
			Company:      "StartupXYZ",
			Description:  "Developed complete SaaS platform with React frontend and Node.js backend.",
			StartDate:    time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      ptrTime(time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC)),
			Technologies: pq.StringArray{"React", "Node.js", "TypeScript", "MongoDB", "AWS"},
			Tags:         pq.StringArray{"fullstack", "saas", "cloud"},
			Category:     "fullstack",
			Featured:     false,
		},
		{
			Title:        "C++ Developer",
			Company:      "GameDev Studios",
			Description:  "Implemented game engine features and optimized rendering pipeline.",
			StartDate:    time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      ptrTime(time.Date(2020, 5, 31, 0, 0, 0, 0, time.UTC)),
			Technologies: pq.StringArray{"C++", "OpenGL", "Vulkan", "CMake"},
			Tags:         pq.StringArray{"cpp", "gamedev", "graphics"},
			Category:     "other",
			Featured:     false,
		},
	}

	for _, exp := range experiences {
		if err := db.Create(&exp); err != nil {
			log.Printf("⚠️  Failed to seed experience %s: %v", exp.Title, err)
		} else {
			log.Printf("✅ Seeded experience: %s at %s", exp.Title, exp.Company)
		}
	}
}

func seedSkills(db interface{ Create(interface{}) error }) {
	skills := []models.Skill{
		{Name: "Go", Level: models.SkillLevelExpert, Category: "Languages", Tags: pq.StringArray{"backend", "performance"}, YearsExperience: 5, Icon: "golang", Featured: true},
		{Name: "PostgreSQL", Level: models.SkillLevelAdvanced, Category: "Databases", Tags: pq.StringArray{"backend", "sql"}, YearsExperience: 6, Icon: "postgresql", Featured: true},
		{Name: "Redis", Level: models.SkillLevelAdvanced, Category: "Databases", Tags: pq.StringArray{"backend", "cache"}, YearsExperience: 4, Icon: "redis", Featured: true},
		{Name: "Docker", Level: models.SkillLevelAdvanced, Category: "DevOps", Tags: pq.StringArray{"devops", "containers"}, YearsExperience: 5, Icon: "docker", Featured: true},
		{Name: "Kubernetes", Level: models.SkillLevelIntermediate, Category: "DevOps", Tags: pq.StringArray{"devops", "orchestration"}, YearsExperience: 3, Icon: "kubernetes", Featured: false},
		{Name: "TypeScript", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"frontend", "backend"}, YearsExperience: 5, Icon: "typescript", Featured: true},
		{Name: "React", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend"}, YearsExperience: 5, Icon: "react", Featured: true},
		{Name: "Next.js", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend", "fullstack"}, YearsExperience: 3, Icon: "nextjs", Featured: true},
		{Name: "C++", Level: models.SkillLevelIntermediate, Category: "Languages", Tags: pq.StringArray{"systems", "performance"}, YearsExperience: 4, Icon: "cplusplus", Featured: false},
		{Name: "Python", Level: models.SkillLevelIntermediate, Category: "Languages", Tags: pq.StringArray{"backend", "ai", "scripting"}, YearsExperience: 4, Icon: "python", Featured: false},
	}

	for _, skill := range skills {
		if err := db.Create(&skill); err != nil {
			log.Printf("⚠️  Failed to seed skill %s: %v", skill.Name, err)
		} else {
			log.Printf("✅ Seeded skill: %s (%s)", skill.Name, skill.Level)
		}
	}
}

func seedProjects(db interface{ Create(interface{}) error }) {
	projects := []models.Project{
		{
			Title:          "maicivy - AI-Powered CV",
			Description:    "Interactive CV with AI-generated motivation/anti-motivation letters. Built with Go, Next.js, PostgreSQL, and Redis.",
			GithubURL:      "https://github.com/yourusername/maicivy",
			DemoURL:        "https://maicivy.example.com",
			Technologies:   pq.StringArray{"Go", "Next.js", "PostgreSQL", "Redis", "Docker", "AI"},
			Category:       "fullstack",
			GithubStars:    42,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "Microservices E-Commerce",
			Description:    "Scalable e-commerce platform with microservices architecture, event-driven communication, and CQRS pattern.",
			GithubURL:      "https://github.com/yourusername/ecommerce-microservices",
			Technologies:   pq.StringArray{"Go", "gRPC", "Kafka", "PostgreSQL", "Kubernetes"},
			Category:       "backend",
			GithubStars:    156,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     false,
		},
		{
			Title:          "Real-Time Chat Application",
			Description:    "WebSocket-based chat with rooms, private messages, typing indicators, and message history.",
			GithubURL:      "https://github.com/yourusername/realtime-chat",
			DemoURL:        "https://chat.example.com",
			Technologies:   pq.StringArray{"Node.js", "Socket.io", "React", "MongoDB", "Redis"},
			Category:       "fullstack",
			GithubStars:    89,
			GithubLanguage: "TypeScript",
			Featured:       false,
			InProgress:     false,
		},
	}

	for _, project := range projects {
		if err := db.Create(&project); err != nil {
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
