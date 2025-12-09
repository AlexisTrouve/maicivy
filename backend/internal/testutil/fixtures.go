// backend/internal/testutil/fixtures.go
package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
	"maicivy/internal/cache"
	"maicivy/internal/models"
	"gorm.io/gorm"
)

// Fixtures pré-définies pour tests
var (
	FixtureExperienceBackend = &models.Experience{
		Title:        "Backend Developer",
		Company:      "Tech Corp",
		Description:  "Developed APIs in Go with PostgreSQL and Redis",
		StartDate:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      nil, // Current
		Technologies: []string{"go", "postgresql", "redis", "fiber", "docker"},
		Tags:         []string{"backend", "api", "database"},
		Category:     "backend",
		Featured:     false,
	}

	FixtureExperienceFrontend = &models.Experience{
		Title:        "Frontend Developer",
		Company:      "Design Studio",
		Description:  "Built React applications with TypeScript and Next.js",
		StartDate:    time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      &[]time.Time{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}[0],
		Technologies: []string{"react", "typescript", "nextjs", "tailwind"},
		Tags:         []string{"frontend", "ui", "spa"},
		Category:     "frontend",
		Featured:     false,
	}

	FixtureExperienceFullStack = &models.Experience{
		Title:        "Full-Stack Engineer",
		Company:      "StartupCo",
		Description:  "Full-stack development with Go backend and React frontend",
		StartDate:    time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      nil, // Current
		Technologies: []string{"go", "react", "postgresql", "docker", "kubernetes"},
		Tags:         []string{"fullstack", "api", "ui"},
		Category:     "fullstack",
		Featured:     true,
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
		Title:          "maicivy - AI-powered CV",
		Description:    "Interactive CV with AI-generated motivation letters",
		GithubURL:      "https://github.com/user/maicivy",
		DemoURL:        "https://maicivy.example.com",
		Technologies:   []string{"go", "react", "postgresql", "redis", "claude-ai"},
		Category:       "fullstack",
		Featured:       true,
		GithubStars:    42,
		GithubForks:    10,
		GithubLanguage: "Go",
		InProgress:     false,
	}

	FixtureProjectEcommerce = &models.Project{
		Title:          "E-commerce Platform",
		Description:    "Scalable e-commerce backend with microservices",
		GithubURL:      "https://github.com/user/ecommerce",
		Technologies:   []string{"go", "postgresql", "redis", "rabbitmq"},
		Category:       "backend",
		Featured:       false,
		GithubStars:    15,
		GithubForks:    3,
		GithubLanguage: "Go",
		InProgress:     false,
	}

)

// Helper pour créer dataset complet pour tests
func CreateFullCVDataset(db *gorm.DB) error {
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

	return nil
}

// Helper pour créer visiteur test
func CreateTestVisitor(db *gorm.DB, sessionID string, visitCount int) (*models.Visitor, error) {
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
func CreateTestLetter(db *gorm.DB, visitorID string, companyName string) (*models.GeneratedLetter, error) {
	vid, err := uuid.Parse(visitorID)
	if err != nil {
		return nil, err
	}

	letter := &models.GeneratedLetter{
		VisitorID:   vid,
		CompanyName: companyName,
		LetterType:  models.LetterTypeMotivation,
		Content:     "Test motivation letter for " + companyName,
		AIModel:     "test-model",
		TokensUsed:  100,
		CompanyInfo: `{"industry": "tech"}`,
	}

	if err := db.Create(letter).Error; err != nil {
		return nil, err
	}

	return letter, nil
}

// Helper pour cleanup database
func CleanupDatabase(db *gorm.DB) error {
	tables := []string{
		"generated_letters",
		"analytics_events",
		"visitors",
		"projects",
		"skills",
		"experiences",
	}

	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}

// Helper pour créer mock Redis data
func SeedRedisCache(redisCache *cache.RedisCache) error {
	ctx := context.Background()

	// Set visitor counts
	redisCache.Set(ctx, "visitor:test-session-1:count", "1", 3600*time.Second)
	redisCache.Set(ctx, "visitor:test-session-2:count", "3", 3600*time.Second)
	redisCache.Set(ctx, "visitor:test-session-3:count", "5", 3600*time.Second)

	// Set rate limits
	redisCache.Set(ctx, "ratelimit:ai:test-session-1", "0", 86400*time.Second)
	redisCache.Set(ctx, "ratelimit:ai:test-session-2", "2", 86400*time.Second)

	// Note: HSet is not available in RedisCache wrapper
	// Use redis client directly if needed for hash operations

	return nil
}
