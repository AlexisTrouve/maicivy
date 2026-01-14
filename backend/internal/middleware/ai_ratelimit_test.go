package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestAIRateLimit_FirstRequest(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	app := fiber.New()

	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "test-session"})

	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)
}

func TestAIRateLimit_DailyLimitExceeded(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Simuler 5 générations déjà faites
	sessionID := "limited-session"
	dailyKey := "ratelimit:ai:" + sessionID + ":daily"
	redisClient.Set(mr.Ctx(), dailyKey, "5", 24*time.Hour)

	app := fiber.New()

	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: sessionID})

	resp, _ := app.Test(req)

	assert.Equal(t, 429, resp.StatusCode)
}

func TestAIRateLimit_CooldownActive(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Activer cooldown
	sessionID := "cooldown-session"
	cooldownKey := "ratelimit:ai:" + sessionID + ":cooldown"
	redisClient.Set(mr.Ctx(), cooldownKey, "1", 2*time.Minute)

	app := fiber.New()

	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: sessionID})

	resp, _ := app.Test(req)

	assert.Equal(t, 429, resp.StatusCode)
}

func TestIncrementAIRateLimit(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		// Simuler les locals mis par le middleware
		c.Locals("rate_limit_daily_key", "ratelimit:ai:test-session:daily")
		c.Locals("rate_limit_cooldown_key", "ratelimit:ai:test-session:cooldown")
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		err := IncrementAIRateLimit(c, redisClient, 2*time.Minute)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	// Vérifier que les clés ont été créées
	dailyKey := "ratelimit:ai:test-session:daily"
	cooldownKey := "ratelimit:ai:test-session:cooldown"

	dailyCount, _ := redisClient.Get(mr.Ctx(), dailyKey).Result()
	assert.Equal(t, "1", dailyCount)

	cooldownExists, _ := redisClient.Exists(mr.Ctx(), cooldownKey).Result()
	assert.Equal(t, int64(1), cooldownExists)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{2*time.Hour + 30*time.Minute, "2h30m"},
		{45 * time.Minute, "45m"},
		{30 * time.Second, "30s"},
		{1*time.Hour + 5*time.Minute, "1h5m"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		assert.Equal(t, test.expected, result)
	}
}
