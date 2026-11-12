package middleware

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

// QUOI : AIBudget (générations en fond) et AIRateLimit (routes) tirent sur LE MÊME budget.
// POURQUOI : si leurs clés divergeaient, un visiteur pourrait épuiser son quota par le chat puis
// générer encore par /cv/generate (ou l'inverse) — deux budgets distincts, sans erreur visible.
func TestAIBudget_PartageLesCompteursDuMiddleware(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	b := AIBudget{Redis: r, MaxPerDay: 2, MaxPerDayPerIP: 10, GlobalDailyMax: 100}

	require.True(t, b.Allow(ctx, "sess", "0.0.0.0"))
	require.NoError(t, b.Consume(ctx, "sess", "0.0.0.0"))
	require.NoError(t, b.Consume(ctx, "sess", "0.0.0.0"))
	assert.False(t, b.Allow(ctx, "sess", "0.0.0.0"), "plafond session atteint")
	for _, k := range []string{"ratelimit:ai:sess:daily", "ratelimit:ai:ip:0.0.0.0:daily", globalDailyKey} {
		assert.Greater(t, mr.TTL(k).Seconds(), 0.0, "compteur %s sans expiration", k)
	}

	// La route gardée par AIRateLimit (mêmes plafonds) refuse désormais la même session.
	app := fiber.New()
	app.Post("/gen", AIRateLimit(AIRateLimitConfig{Redis: r, MaxPerDay: 2, MaxPerDayPerIP: 10}), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("POST", "/gen", nil)
	req.Header.Set("Cookie", SessionCookieName+"=sess")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, "le quota consommé en fond compte pour la route")
}

func TestAIBudget_PlafondsIPEtGlobal(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	ip := AIBudget{Redis: r, MaxPerDay: 10, MaxPerDayPerIP: 1}
	require.NoError(t, ip.Consume(ctx, "a", "9.9.9.9"))
	assert.False(t, ip.Allow(ctx, "b", "9.9.9.9"), "une autre session sur la même IP est bornée par l'IP")

	global := AIBudget{Redis: r, MaxPerDay: 10, MaxPerDayPerIP: 10, GlobalDailyMax: 1}
	assert.False(t, global.Allow(ctx, "c", "8.8.8.8"), "circuit-breaker global déjà atteint par la génération précédente")
}
