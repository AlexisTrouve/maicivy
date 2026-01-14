package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/database"
	"maicivy/internal/models"
)

func TestTrackingMiddleware_NewVisitor(t *testing.T) {
	// Setup
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient)

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	// Test
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Vérifier cookie créé
	cookies := resp.Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, SessionCookieName, cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
}

func TestTrackingMiddleware_ReturningVisitor(t *testing.T) {
	// Setup
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient)

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	// Première visite
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, _ := app.Test(req1)
	sessionCookie := resp1.Cookies()[0]

	// Deuxième visite (avec cookie)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(sessionCookie)
	resp2, err := app.Test(req2)
	require.NoError(t, err)

	// Parser response
	var body map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&body)

	// Assertions
	assert.Equal(t, float64(2), body["visit_count"])
}

func TestDetectProfile_LinkedIn(t *testing.T) {
	tm := &TrackingMiddleware{}

	app := fiber.New()
	c := app.AcquireCtx(&fiber.Ctx{})
	defer app.ReleaseCtx(c)

	c.Request().Header.Set("User-Agent", "LinkedInBot/1.0")

	profile := tm.detectProfile(c)
	assert.Equal(t, "linkedin_bot", profile)
}
