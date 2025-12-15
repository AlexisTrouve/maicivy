package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// setupTimelineTestDB initialise une base de données de test
func setupTimelineTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrations
	err = db.AutoMigrate(&models.Experience{}, &models.Project{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// seedTimelineData insère des données de test
func seedTimelineData(t *testing.T, db *gorm.DB) {
	now := time.Now()
	lastYear := now.AddDate(-1, 0, 0)
	twoYearsAgo := now.AddDate(-2, 0, 0)
	threeYearsAgo := now.AddDate(-3, 0, 0)

	// Expériences
	experiences := []models.Experience{
		{
			Title:        "Senior Backend Developer",
			Company:      "Tech Corp",
			Description:  "Leading backend development team",
			StartDate:    lastYear,
			EndDate:      nil, // Emploi actuel
			Technologies: pq.StringArray{"Go", "PostgreSQL", "Redis"},
			Tags:         pq.StringArray{"Go", "PostgreSQL", "Redis"},
			Category:     "backend",
		},
		{
			Title:        "Full-Stack Developer",
			Company:      "Startup Inc",
			Description:  "Building web applications",
			StartDate:    threeYearsAgo,
			EndDate:      &twoYearsAgo,
			Technologies: pq.StringArray{"Node.js", "React", "MongoDB"},
			Tags:         pq.StringArray{"Node.js", "React", "MongoDB"},
			Category:     "fullstack",
		},
	}

	for _, exp := range experiences {
		if err := db.Create(&exp).Error; err != nil {
			t.Fatalf("Failed to seed experience: %v", err)
		}
	}

	// Projets
	projects := []models.Project{
		{
			Title:          "maicivy",
			Description:    "CV interactif avec IA",
			GithubURL:      "https://github.com/user/maicivy",
			Technologies:   pq.StringArray{"Go", "Next.js", "PostgreSQL"},
			Category:       "backend",
			GithubStars:    42,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "Old Project",
			Description:    "Archived project",
			Technologies:   pq.StringArray{"Python", "Django"},
			Category:       "backend",
			InProgress:     false,
		},
	}

	for _, proj := range projects {
		if err := db.Create(&proj).Error; err != nil {
			t.Fatalf("Failed to seed project: %v", err)
		}
	}
}

func TestGetTimeline(t *testing.T) {
	// Setup
	db := setupTimelineTestDB(t)
	seedTimelineData(t, db)

	app := fiber.New()
	handler := NewTimelineHandler(db)
	app.Get("/api/v1/timeline", handler.GetTimeline)

	// Test: Récupérer toute la timeline
	t.Run("Get all timeline events", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// On devrait avoir 2 expériences + 2 projets = 4 événements
		assert.Equal(t, 4, len(events))
		assert.Equal(t, float64(4), data["total"])
	})

	// Test: Filtrer par catégorie
	t.Run("Filter by category", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/timeline?category=backend", nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// Devrait avoir 1 expérience backend + 2 projets backend
		assert.Equal(t, 3, len(events))

		// Vérifier que tous les events sont de la bonne catégorie
		for _, event := range events {
			e := event.(map[string]interface{})
			assert.Equal(t, "backend", e["category"])
		}
	})

	// Test: Vérifier le tri chronologique
	t.Run("Events sorted by date DESC", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		data := result["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// Vérifier que les dates sont en ordre décroissant
		var prevDate time.Time
		for i, event := range events {
			e := event.(map[string]interface{})
			currentDate, _ := time.Parse(time.RFC3339, e["start_date"].(string))

			if i > 0 {
				// La date actuelle doit être <= la date précédente
				assert.True(t, currentDate.Before(prevDate) || currentDate.Equal(prevDate),
					"Events should be sorted by date DESC")
			}

			prevDate = currentDate
		}
	})

	// Test: Vérifier les stats
	t.Run("Check stats calculation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		data := result["data"].(map[string]interface{})
		stats := data["stats"].(map[string]interface{})

		assert.Equal(t, float64(2), stats["total_experiences"])
		assert.Equal(t, float64(2), stats["total_projects"])
		assert.NotNil(t, stats["categories_breakdown"])
		assert.NotNil(t, stats["top_technologies"])
	})
}

func TestGetCategories(t *testing.T) {
	// Setup
	db := setupTimelineTestDB(t)
	seedTimelineData(t, db)

	app := fiber.New()
	handler := NewTimelineHandler(db)
	app.Get("/api/v1/timeline/categories", handler.GetCategories)

	req := httptest.NewRequest("GET", "/api/v1/timeline/categories", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result["success"].(bool))
	categories := result["categories"].([]interface{})

	// Devrait avoir "backend" et "fullstack"
	assert.GreaterOrEqual(t, len(categories), 2)

	// Vérifier que les catégories sont présentes
	categoryStrings := make([]string, len(categories))
	for i, cat := range categories {
		categoryStrings[i] = cat.(string)
	}
	assert.Contains(t, categoryStrings, "backend")
	assert.Contains(t, categoryStrings, "fullstack")
}

func TestGetMilestones(t *testing.T) {
	// Setup
	db := setupTimelineTestDB(t)
	seedTimelineData(t, db)

	app := fiber.New()
	handler := NewTimelineHandler(db)
	app.Get("/api/v1/timeline/milestones", handler.GetMilestones)

	req := httptest.NewRequest("GET", "/api/v1/timeline/milestones", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result["success"].(bool))
	milestones := result["milestones"].([]interface{})

	// Devrait avoir au moins quelques milestones générés automatiquement
	assert.Greater(t, len(milestones), 0)

	// Vérifier la structure d'un milestone
	if len(milestones) > 0 {
		milestone := milestones[0].(map[string]interface{})
		assert.NotEmpty(t, milestone["id"])
		assert.NotEmpty(t, milestone["title"])
		assert.NotEmpty(t, milestone["description"])
		assert.NotEmpty(t, milestone["date"])
		assert.NotEmpty(t, milestone["icon"])
		assert.NotEmpty(t, milestone["type"])
	}
}

func TestTimelineEventTypes(t *testing.T) {
	// Setup
	db := setupTimelineTestDB(t)
	seedTimelineData(t, db)

	app := fiber.New()
	handler := NewTimelineHandler(db)
	app.Get("/api/v1/timeline", handler.GetTimeline)

	req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	data := result["data"].(map[string]interface{})
	events := data["events"].([]interface{})

	// Vérifier que chaque événement a les champs requis
	for _, event := range events {
		e := event.(map[string]interface{})

		// Champs obligatoires
		assert.NotEmpty(t, e["id"])
		assert.NotEmpty(t, e["type"])
		assert.Contains(t, []string{"experience", "project"}, e["type"])
		assert.NotEmpty(t, e["title"])
		assert.NotEmpty(t, e["subtitle"])
		assert.NotEmpty(t, e["start_date"])
		assert.NotEmpty(t, e["category"])
		assert.NotNil(t, e["tags"])
		assert.NotNil(t, e["is_current"])

		// Vérifier le format de l'ID
		idStr := e["id"].(string)
		assert.True(t,
			len(idStr) > 4 && (idStr[:4] == "exp_" || idStr[:5] == "proj_"),
			"ID should have correct prefix")
	}
}

func TestTimelineFilterByDate(t *testing.T) {
	// Setup
	db := setupTimelineTestDB(t)
	seedTimelineData(t, db)

	app := fiber.New()
	handler := NewTimelineHandler(db)
	app.Get("/api/v1/timeline", handler.GetTimeline)

	// Test: Filtrer par date "from"
	t.Run("Filter by from date", func(t *testing.T) {
		// Filtrer pour obtenir seulement les événements de cette année
		fromDate := time.Now().AddDate(0, -6, 0).Format("2006-01-02")

		req := httptest.NewRequest("GET", "/api/v1/timeline?from="+fromDate, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		data := result["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// Devrait avoir au moins l'expérience actuelle
		assert.Greater(t, len(events), 0)

		// Vérifier que toutes les dates sont >= from
		from, _ := time.Parse("2006-01-02", fromDate)
		for _, event := range events {
			e := event.(map[string]interface{})
			eventDate, _ := time.Parse(time.RFC3339, e["start_date"].(string))
			assert.True(t, eventDate.After(from) || eventDate.Equal(from),
				"Event date should be >= from date")
		}
	})
}
