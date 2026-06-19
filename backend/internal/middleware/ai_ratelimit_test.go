package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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
	redisClient.Set(context.Background(), dailyKey, "5", 24*time.Hour)

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
	redisClient.Set(context.Background(), cooldownKey, "1", 2*time.Minute)

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

	dailyCount, _ := redisClient.Get(context.Background(), dailyKey).Result()
	assert.Equal(t, "1", dailyCount)

	cooldownExists, _ := redisClient.Exists(context.Background(), cooldownKey).Result()
	assert.Equal(t, int64(1), cooldownExists)
}

// TestAIRateLimit_GlobalDailyLimitExceeded vérifie le circuit-breaker coût :
// quand le compteur global atteint GlobalDailyMax, toute requête non-owner reçoit 503.
func TestAIRateLimit_GlobalDailyLimitExceeded(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// Simuler le plafond global atteint
	redisClient.Set(context.Background(), globalDailyKey, "200", 24*time.Hour)

	app := fiber.New()
	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
		GlobalDailyMax:   200,
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "any-session"})

	resp, _ := app.Test(req)
	assert.Equal(t, 503, resp.StatusCode)
}

// TestAIRateLimit_GlobalLimitOwnerBypass vérifie que l'owner (X-Owner-Key) passe
// même quand le plafond global est atteint — le circuit-breaker ne vise que les non-owners.
func TestAIRateLimit_GlobalLimitOwnerBypass(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient.Set(context.Background(), globalDailyKey, "200", 24*time.Hour)

	app := fiber.New()
	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
		GlobalDailyMax:   200,
		OwnerAPIKey:      "secret-owner-key",
	}))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("OK") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Owner-Key", "secret-owner-key")

	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestIncrementAIRateLimit_IncrementsGlobal vérifie que l'incrément alimente bien
// le compteur global lu par le circuit-breaker.
func TestIncrementAIRateLimit_IncrementsGlobal(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("rate_limit_daily_key", "ratelimit:ai:test-session:daily")
		c.Locals("rate_limit_cooldown_key", "ratelimit:ai:test-session:cooldown")
		return c.Next()
	})
	app.Get("/inc", func(c *fiber.Ctx) error {
		if err := IncrementAIRateLimit(c, redisClient, 2*time.Minute); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.SendString("OK")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/inc", nil))
	assert.Equal(t, 200, resp.StatusCode)

	globalCount, _ := redisClient.Get(context.Background(), globalDailyKey).Result()
	assert.Equal(t, "1", globalCount)
}

// TestAIRateLimit_ConcurrentSameSession_OnlyOnePasses verrouille le fix TOCTOU : avant, le check
// (lecture des compteurs) et l'incrément (après succès, dans le handler) n'étaient pas atomiques,
// donc N requêtes concurrentes d'une MÊME session passaient toutes avant qu'un cooldown ne s'arme
// → N générations payantes au lieu d'une. Le verrou in-flight (SETNX) ne doit en laisser passer
// qu'UNE à la fois.
func TestAIRateLimit_ConcurrentSameSession_OnlyOnePasses(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	app := fiber.New()
	app.Use(AIRateLimit(AIRateLimitConfig{
		Redis:            rc,
		MaxPerDay:        5,
		MaxPerDayPerIP:   100, // ne pas laisser la limite IP interférer avec le test de concurrence
		CooldownDuration: 2 * time.Minute,
	}))

	var reached int32
	app.Get("/test", func(c *fiber.Ctx) error {
		atomic.AddInt32(&reached, 1)
		time.Sleep(100 * time.Millisecond) // simule la latence de génération (la fenêtre TOCTOU)
		return c.SendString("OK")
	})

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			req.AddCookie(&http.Cookie{Name: "maicivy_session", Value: "race-session"})
			_, _ = app.Test(req, 5000)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&reached),
		"une seule génération concurrente par session doit atteindre le handler (verrou in-flight)")
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
