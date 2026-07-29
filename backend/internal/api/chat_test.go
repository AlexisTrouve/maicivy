package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChatHandler_IncrementRateLimit_FeedsChatBudget verrouille le bug corrigé : avant ce fix,
// StreamChat n'appelait JAMAIS IncrementAIRateLimit, donc chatter ne consommait jamais son propre
// quota (les compteurs "ratelimit:chat:..." ne bougeaient jamais). On simule les locals que
// chatRateLimitMW pose (KeyPrefix "chat") et on vérifie que incrementRateLimit alimente bien CE
// compteur, isolé de tout autre budget (CV/lettres).
func TestChatHandler_IncrementRateLimit_FeedsChatBudget(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	h := &ChatHandler{redis: redisClient}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// Simule EXACTEMENT ce que chatRateLimitMW (KeyPrefix "chat") pose dans les locals.
		c.Locals("rate_limit_daily_key", "ratelimit:chat:sess-1:daily")
		c.Locals("rate_limit_cooldown_key", "ratelimit:chat:sess-1:cooldown")
		c.Locals("rate_limit_ip_daily_key", "ratelimit:chat:ip:1.2.3.4:daily")
		h.incrementRateLimit(c)
		return c.SendString("OK")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	dailyCount, _ := redisClient.Get(context.Background(), "ratelimit:chat:sess-1:daily").Result()
	assert.Equal(t, "1", dailyCount, "le compteur journalier DÉDIÉ chat doit avoir été incrémenté")

	cooldownExists, _ := redisClient.Exists(context.Background(), "ratelimit:chat:sess-1:cooldown").Result()
	assert.Equal(t, int64(1), cooldownExists, "le cooldown (court, chatCooldownDuration) doit être posé")

	// Le budget "ai" (CV/lettres) de la MÊME session ne doit pas avoir bougé — isolation totale.
	aiCount, err := redisClient.Get(context.Background(), "ratelimit:ai:sess-1:daily").Result()
	assert.ErrorIs(t, err, redis.Nil, "le budget ai:... ne doit pas exister, isolé de chat:...")
	assert.Empty(t, aiCount)
}

// TestChatHandler_IncrementRateLimit_NilRedisNoop : contexte de test/dégradé (redis absent) → no-op,
// pas de panic (même garde que CVHandler.incrementRateLimit).
func TestChatHandler_IncrementRateLimit_NilRedisNoop(t *testing.T) {
	h := &ChatHandler{redis: nil}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		h.incrementRateLimit(c) // ne doit pas paniquer
		return c.SendString("OK")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

// TestChatHandler_IncrementRateLimit_OwnerBypass : owner (is_owner=true dans les locals, posé par
// chatRateLimitMW sur bypass) → aucun compteur incrémenté (cf. IncrementAIRateLimit).
func TestChatHandler_IncrementRateLimit_OwnerBypass(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	h := &ChatHandler{redis: redisClient}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("is_owner", true)
		h.incrementRateLimit(c)
		return c.SendString("OK")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, resp.StatusCode)

	keys, _ := redisClient.Keys(context.Background(), "ratelimit:*").Result()
	assert.Empty(t, keys, "owner bypass → aucune clé de rate-limit ne doit être créée")
}
