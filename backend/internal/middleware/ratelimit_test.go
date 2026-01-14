package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit_AI_DailyLimit(t *testing.T) {
	// Setup
	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Test: 5 générations ok
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/generate", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Wait for cooldown
		time.Sleep(2 * time.Minute)
	}

	// Test: 6ème génération bloquée
	req := httptest.NewRequest("POST", "/generate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimit_AI_Cooldown(t *testing.T) {
	// Setup
	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Première génération ok
	req1 := httptest.NewRequest("POST", "/generate", nil)
	resp1, _ := app.Test(req1)
	assert.Equal(t, 200, resp1.StatusCode)

	// Deuxième génération immédiate bloquée (cooldown)
	req2 := httptest.NewRequest("POST", "/generate", nil)
	resp2, _ := app.Test(req2)
	assert.Equal(t, 429, resp2.StatusCode)

	// Après 2min, ok
	time.Sleep(2 * time.Minute)
	req3 := httptest.NewRequest("POST", "/generate", nil)
	resp3, _ := app.Test(req3)
	assert.Equal(t, 200, resp3.StatusCode)
}
