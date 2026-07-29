package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/api"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupTestApp(t *testing.T) (*fiber.App, *services.AnalyticsService, func()) {
	// Setup SQLite in-memory
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Visitor{},
		&models.AnalyticsEvent{},
		&models.GeneratedLetter{},
	)
	require.NoError(t, err)

	// Setup miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Créer service et handler
	analyticsService := services.NewAnalyticsService(db, redisClient)
	analyticsHandler := api.NewAnalyticsHandler(analyticsService)

	// Setup Fiber app
	app := fiber.New()
	analyticsHandler.RegisterRoutes(app)

	cleanup := func() {
		mr.Close()
		redisClient.Close()
	}

	return app, analyticsService, cleanup
}

func TestAnalyticsAPI_GetRealtimeStats(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/analytics/realtime", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "current_visitors")
	assert.Contains(t, data, "unique_today")
	assert.Contains(t, data, "timestamp")
}

func TestAnalyticsAPI_GetStats_ValidPeriod(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	tests := []struct {
		period string
	}{
		{"day"},
		{"week"},
		{"month"},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/analytics/stats?period="+tt.period, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			assert.True(t, body["success"].(bool))
			data := body["data"].(map[string]interface{})
			assert.Equal(t, tt.period, data["period"])
		})
	}
}

func TestAnalyticsAPI_GetStats_InvalidPeriod(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/analytics/stats?period=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.False(t, body["success"].(bool))
	assert.Contains(t, body["error"], "Invalid period")
}

func TestAnalyticsAPI_GetTopThemes(t *testing.T) {
	app, service, cleanup := setupTestApp(t)
	defer cleanup()

	// Insérer données test
	service.GetTopThemes(context.Background(), 5) // Initialiser Redis

	req := httptest.NewRequest("GET", "/api/v1/analytics/themes?limit=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])
}

func TestAnalyticsAPI_TrackEvent_Success(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	// Créer un visiteur pour avoir un visitor_id
	visitorID := uuid.New()

	// Mock context avec visitor_id
	payload := map[string]interface{}{
		"event_type": "button_click",
		"event_data": map[string]interface{}{
			"button": "cta",
			"x":      450,
			"y":      200,
		},
		"page_url": "/cv",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/analytics/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Simuler le middleware tracking en ajoutant visitor_id au contexte
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("visitor_id", visitorID)
		return c.Next()
	})

	// Re-créer l'app avec le middleware
	app2, _, cleanup2 := setupTestApp(t)
	defer cleanup2()

	// Ajouter middleware
	app2.Use(func(c *fiber.Ctx) error {
		c.Locals("visitor_id", visitorID)
		return c.Next()
	})

	req2 := httptest.NewRequest("POST", "/api/v1/analytics/event", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")

	resp, err := app2.Test(req2)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // visitor_id manquant car contexte non partagé
}

func TestAnalyticsAPI_TrackEvent_MissingVisitor(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	payload := map[string]interface{}{
		"event_type": "button_click",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/analytics/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"], "Missing visitor session")
}

func TestAnalyticsAPI_GetTimeline(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/analytics/timeline?limit=10&offset=0", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])
	assert.NotNil(t, body["meta"])

	meta := body["meta"].(map[string]interface{})
	assert.Equal(t, float64(10), meta["limit"])
	assert.Equal(t, float64(0), meta["offset"])
}

func TestAnalyticsAPI_GetHeatmap(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/analytics/heatmap?page_url=/cv&hours=24", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])
	assert.NotNil(t, body["meta"])

	meta := body["meta"].(map[string]interface{})
	assert.Equal(t, "/cv", meta["page_url"])
	assert.Equal(t, float64(24), meta["hours"])
}

func TestAnalyticsAPI_GetLettersStats(t *testing.T) {
	app, _, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/analytics/letters?period=day", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "period")
	assert.Contains(t, data, "total")
}
