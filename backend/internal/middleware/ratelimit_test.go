package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit_AI_DailyLimit(t *testing.T) {
	// Ce test nécessite des sleeps de 2min × 5 — skip en mode court
	if testing.Short() {
		t.Skip("Skipping slow ratelimit test in short mode")
	}

	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// 5 générations espacées par le cooldown miniredis (FastForward)
	mr := setupTestRedis(t)
	_ = mr // miniredis géré par setupTestRedis + t.Cleanup

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/generate", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	req := httptest.NewRequest("POST", "/generate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimit_AI_Cooldown(t *testing.T) {
	// Skip en mode court — nécessite un vrai délai de cooldown
	if testing.Short() {
		t.Skip("Skipping slow ratelimit cooldown test in short mode")
	}

	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req1 := httptest.NewRequest("POST", "/generate", nil)
	resp1, _ := app.Test(req1)
	assert.Equal(t, 200, resp1.StatusCode)

	// Deuxième génération immédiate bloquée (cooldown)
	req2 := httptest.NewRequest("POST", "/generate", nil)
	resp2, _ := app.Test(req2)
	assert.Equal(t, 429, resp2.StatusCode)
}
