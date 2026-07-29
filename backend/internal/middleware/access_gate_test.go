package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// setupAccessGateTestDB : DB SQLite locale pour access_gate_test (pas de redis)
func setupAccessGateTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Visitor{})
	return db
}

func TestAccessGate_InsufficientVisits(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db := setupAccessGateTestDB()

	visitor := models.Visitor{
		SessionID:  "test-session",
		VisitCount: 2,
	}
	db.Create(&visitor)

	app := fiber.New()
	app.Use(AccessGate(AccessGateConfig{
		Redis:     redisClient,
		DB:        db,
		MinVisits: 3,
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "test-session"})
	resp, _ := app.Test(req)

	assert.Equal(t, 403, resp.StatusCode)
}

func TestAccessGate_SufficientVisits(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db := setupAccessGateTestDB()

	visitor := models.Visitor{
		SessionID:  "test-session",
		VisitCount: 3,
	}
	db.Create(&visitor)

	app := fiber.New()
	app.Use(AccessGate(AccessGateConfig{
		Redis:     redisClient,
		DB:        db,
		MinVisits: 3,
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "test-session"})
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)
}

func TestAccessGate_ProfileBypass(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db := setupAccessGateTestDB()

	visitor := models.Visitor{
		SessionID:       "recruiter-session",
		VisitCount:      0,
		ProfileDetected: models.ProfileTypeRecruiter,
	}
	db.Create(&visitor)

	app := fiber.New()
	app.Use(AccessGate(AccessGateConfig{
		Redis:           redisClient,
		DB:              db,
		MinVisits:       3,
		BypassOnProfile: true,
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	// mr.Ctx est un champ context.Context dans les nouvelles versions de miniredis
	redisClient.Set(context.Background(), "visitor:recruiter-session:profile", "recruiter", 0)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "recruiter-session"})
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)
}

func TestAccessGate_NoSession(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db := setupAccessGateTestDB()

	app := fiber.New()
	app.Use(AccessGate(AccessGateConfig{
		Redis:     redisClient,
		DB:        db,
		MinVisits: 3,
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 401, resp.StatusCode)
}
