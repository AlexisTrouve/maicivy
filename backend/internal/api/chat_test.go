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

	"maicivy/internal/middleware"
	"maicivy/internal/services"
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

// QUOI : politique d'un tour de chat (modèle + capabilities) selon le rôle.
// POURQUOI : l'onglet Agent est ouvert aux prospecteurs via le MÊME /chat/stream que l'owner. Un
// cookie prospecteur ne doit JAMAIS faire basculer sur le modèle owner, et l'agent d'un prospecteur
// (hors Composer) doit avoir exactement les droits d'un visiteur : aucun tool de rédaction, aucun
// ancrage tarifaire — rien de plus que les données publiques du portfolio.
func TestChatHandler_TurnPolicy_Prospect(t *testing.T) {
	h := NewChatHandler(nil, nil, "", "owner-model", "visitor-model", 200, 300, 260)
	p := &middleware.ProspectIdentity{ID: "p1", Name: "Operatrice", Ref: "abc"}

	// Agent (compose=false) : modèle visiteur, capabilities nulles = toolset public.
	model, caps := h.turnPolicy(false, p, false)
	assert.Equal(t, "visitor-model", model)
	assert.NotEqual(t, "owner-model", model)
	assert.Equal(t, services.ChatCapabilities{}, caps)

	// Composer : toujours le modèle visiteur, plancher appliqué, écrit au nom du prospecteur.
	model, caps = h.turnPolicy(false, p, true)
	assert.Equal(t, "visitor-model", model)
	assert.True(t, caps.Compose)
	assert.Equal(t, 260, caps.RateFloor)
	require.NotNil(t, caps.Prospect)
	assert.Equal(t, "Operatrice", caps.Prospect.Name)

	// Owner : modèle owner, voix d'Alexis (pas de Prospect).
	model, caps = h.turnPolicy(true, nil, true)
	assert.Equal(t, "owner-model", model)
	assert.Nil(t, caps.Prospect)

	// Visiteur anonyme qui poste compose:true : ignoré.
	model, caps = h.turnPolicy(false, nil, true)
	assert.Equal(t, "visitor-model", model)
	assert.False(t, caps.Compose)
}
