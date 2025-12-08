// backend/internal/testutil/fixtures.go
package testutil

import (
	"time"

	"github.com/maicivy/internal/models"
)

// Fixtures pré-définies pour tests
var (
	FixtureExperienceBackend = &models.Experience{
		Title:       "Backend Developer",
		Company:     "Tech Corp",
		Description: "Developed APIs in Go with PostgreSQL and Redis",
		StartDate:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil, // Current
		Tags:        []string{"go", "postgresql", "redis", "fiber", "docker"},
		Category:    "backend",
		Location:    "Remote",
	}

	FixtureExperienceFrontend = &models.Experience{
		Title:       "Frontend Developer",
		Company:     "Design Studio",
		Description: "Built React applications with TypeScript and Next.js",
		StartDate:   time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     &[]time.Time{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}[0],
		Tags:        []string{"react", "typescript", "nextjs", "tailwind"},
		Category:    "frontend",
		Location:    "Paris, France",
	}

	FixtureExperienceFullStack = &models.Experience{
		Title:       "Full-Stack Engineer",
		Company:     "StartupCo",
		Description: "Full-stack development with Go backend and React frontend",
		StartDate:   time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil, // Current
		Tags:        []string{"go", "react", "postgresql", "docker", "kubernetes"},
		Category:    "fullstack",
		Location:    "San Francisco, USA",
	}

	FixtureSkillGo = &models.Skill{
		Name:            "Go",
		Level:           "Advanced",
		Category:        "backend",
		YearsExperience: 4,
		Tags:            []string{"backend", "api", "microservices", "performance"},
	}

	FixtureSkillReact = &models.Skill{
		Name:            "React",
		Level:           "Intermediate",
		Category:        "frontend",
		YearsExperience: 3,
		Tags:            []string{"frontend", "ui", "spa", "components"},
	}

	FixtureSkillPostgreSQL = &models.Skill{
		Name:            "PostgreSQL",
		Level:           "Advanced",
		Category:        "database",
		YearsExperience: 5,
		Tags:            []string{"database", "sql", "backend", "optimization"},
	}

	FixtureSkillDocker = &models.Skill{
		Name:            "Docker",
		Level:           "Intermediate",
		Category:        "devops",
		YearsExperience: 3,
		Tags:            []string{"devops", "containerization", "deployment"},
	}

	FixtureProjectMaicivy = &models.Project{
		Title:        "maicivy - AI-powered CV",
		Description:  "Interactive CV with AI-generated motivation letters",
		GithubURL:    "https://github.com/user/maicivy",
		DemoURL:      "https://maicivy.example.com",
		Technologies: []string{"go", "react", "postgresql", "redis", "claude-ai"},
		Category:     "fullstack",
		Featured:     true,
		Stars:        42,
	}

	FixtureProjectEcommerce = &models.Project{
		Title:        "E-commerce Platform",
		Description:  "Scalable e-commerce backend with microservices",
		GithubURL:    "https://github.com/user/ecommerce",
		Technologies: []string{"go", "postgresql", "redis", "rabbitmq"},
		Category:     "backend",
		Featured:     false,
		Stars:        15,
	}

	FixtureThemeBackend = &models.Theme{
		Name:        "backend",
		Keywords:    []string{"go", "api", "database", "microservices", "postgresql", "redis"},
		Description: "Backend development focus",
		Weight:      1.0,
	}

	FixtureThemeFrontend = &models.Theme{
		Name:        "frontend",
		Keywords:    []string{"react", "typescript", "nextjs", "ui", "tailwind"},
		Description: "Frontend development focus",
		Weight:      1.0,
	}

	FixtureThemeFullStack = &models.Theme{
		Name:        "fullstack",
		Keywords:    []string{"go", "react", "postgresql", "api", "ui"},
		Description: "Full-stack development",
		Weight:      1.0,
	}

	FixtureThemeDevOps = &models.Theme{
		Name:        "devops",
		Keywords:    []string{"docker", "kubernetes", "ci-cd", "monitoring"},
		Description: "DevOps and infrastructure",
		Weight:      1.0,
	}
)

// Helper pour créer dataset complet pour tests
func CreateFullCVDataset(db *DB) error {
	experiences := []*models.Experience{
		FixtureExperienceBackend,
		FixtureExperienceFrontend,
		FixtureExperienceFullStack,
	}

	skills := []*models.Skill{
		FixtureSkillGo,
		FixtureSkillReact,
		FixtureSkillPostgreSQL,
		FixtureSkillDocker,
	}

	projects := []*models.Project{
		FixtureProjectMaicivy,
		FixtureProjectEcommerce,
	}

	themes := []*models.Theme{
		FixtureThemeBackend,
		FixtureThemeFrontend,
		FixtureThemeFullStack,
		FixtureThemeDevOps,
	}

	// Insert experiences
	for _, exp := range experiences {
		if err := db.Create(exp).Error; err != nil {
			return err
		}
	}

	// Insert skills
	for _, skill := range skills {
		if err := db.Create(skill).Error; err != nil {
			return err
		}
	}

	// Insert projects
	for _, project := range projects {
		if err := db.Create(project).Error; err != nil {
			return err
		}
	}

	// Insert themes
	for _, theme := range themes {
		if err := db.Create(theme).Error; err != nil {
			return err
		}
	}

	return nil
}

// Helper pour créer visiteur test
func CreateTestVisitor(db *DB, sessionID string, visitCount int) (*models.Visitor, error) {
	visitor := &models.Visitor{
		SessionID:       sessionID,
		IPHash:          "test-hash-" + sessionID,
		UserAgent:       "Mozilla/5.0 Test",
		VisitCount:      visitCount,
		FirstVisit:      time.Now().AddDate(0, 0, -visitCount),
		LastVisit:       time.Now(),
		ProfileDetected: "",
	}

	if err := db.Create(visitor).Error; err != nil {
		return nil, err
	}

	return visitor, nil
}

// Helper pour créer lettre test
func CreateTestLetter(db *DB, visitorID uint, companyName string) (*models.GeneratedLetter, error) {
	letter := &models.GeneratedLetter{
		VisitorID:            visitorID,
		CompanyName:          companyName,
		MotivationLetter:     "Test motivation letter for " + companyName,
		AntiMotivationLetter: "Test anti-motivation letter for " + companyName,
		Theme:                "backend",
		CompanyInfo:          map[string]string{"industry": "tech"},
	}

	if err := db.Create(letter).Error; err != nil {
		return nil, err
	}

	return letter, nil
}

// Helper pour cleanup database
func CleanupDatabase(db *DB) error {
	tables := []string{
		"generated_letters",
		"analytics_events",
		"visitors",
		"projects",
		"skills",
		"experiences",
		"themes",
	}

	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}

// Helper pour créer mock Redis data
func SeedRedisCache(cache *Cache) error {
	ctx := context.Background()

	// Set visitor counts
	cache.Set(ctx, "visitor:test-session-1:count", "1", 3600)
	cache.Set(ctx, "visitor:test-session-2:count", "3", 3600)
	cache.Set(ctx, "visitor:test-session-3:count", "5", 3600)

	// Set rate limits
	cache.Set(ctx, "ratelimit:ai:test-session-1", "0", 86400)
	cache.Set(ctx, "ratelimit:ai:test-session-2", "2", 86400)

	// Set analytics stats
	cache.HSet(ctx, "analytics:stats:cv_themes", "backend", "150")
	cache.HSet(ctx, "analytics:stats:cv_themes", "frontend", "89")
	cache.HSet(ctx, "analytics:stats:cv_themes", "fullstack", "64")

	return nil
}
