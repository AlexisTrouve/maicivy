package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// newGlobalRateLimitApp monte le limiteur global devant deux routes, avec la MÊME config proxy
// que la prod (ProxyHeader X-Real-IP) : une requête sans cet en-tête — healthcheck, rendu serveur
// du frontend — a donc une IP VIDE, exactement comme sur VPS142.
func newGlobalRateLimitApp(t *testing.T) (*fiber.App, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	app := fiber.New(fiber.Config{ProxyHeader: "X-Real-IP"})
	app.Use(NewRateLimit(rdb).Global())
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/health/deep", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/api/v1/blog/posts", func(c *fiber.Ctx) error { return c.SendString("posts") })
	return app, mr
}

func statusOf(t *testing.T, app *fiber.App, path string) int {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp.StatusCode
}

// Panne réelle du 2026-10-09 : la clé `ratelimit:global:` (IP vide) est restée à 162 avec un
// TTL de -1 — INCR avait eu lieu, l'EXPIRE jamais (backend recréé entre les deux au déploiement).
// Le compteur ne redescendait donc plus jamais : 429 permanent sur tout le blog.
// Invariant : une clé de compteur sans TTL se répare à la requête suivante, elle n'est jamais immortelle.
func TestGlobalRateLimit_OrphanKeyWithoutTTLHeals(t *testing.T) {
	app, mr := newGlobalRateLimitApp(t)

	// État exact mesuré en prod : compteur au-dessus du plafond, AUCUN TTL.
	mr.Set("ratelimit:global:", "162")

	// Cette requête peut encore être refusée (le compteur est réellement au-dessus)…
	statusOf(t, app, "/api/v1/blog/posts")

	// …mais la clé doit en ressortir avec une échéance.
	if ttl := mr.TTL("ratelimit:global:"); ttl <= 0 {
		t.Fatalf("clé orpheline toujours sans TTL après une requête (ttl=%v) : 429 permanent", ttl)
	}

	// Une fenêtre plus tard, le service doit être rendu.
	mr.FastForward(GlobalRateWindow + time.Second)
	if got := statusOf(t, app, "/api/v1/blog/posts"); got != fiber.StatusOK {
		t.Fatalf("après expiration de la fenêtre : status %d, attendu 200", got)
	}
}

// Fonctionnement nominal : la première requête d'une fenêtre pose bien le TTL.
func TestGlobalRateLimit_FirstRequestSetsTTL(t *testing.T) {
	app, mr := newGlobalRateLimitApp(t)
	statusOf(t, app, "/api/v1/blog/posts")
	if ttl := mr.TTL("ratelimit:global:"); ttl <= 0 || ttl > GlobalRateWindow {
		t.Fatalf("ttl=%v, attendu dans ]0, %v]", ttl, GlobalRateWindow)
	}
}

// Le plafond fonctionne toujours : la requête n°101 d'une même IP est refusée.
func TestGlobalRateLimit_StillLimits(t *testing.T) {
	app, _ := newGlobalRateLimitApp(t)
	for i := 0; i < GlobalRateLimit; i++ {
		if got := statusOf(t, app, "/api/v1/blog/posts"); got != fiber.StatusOK {
			t.Fatalf("requête %d refusée trop tôt (status %d)", i+1, got)
		}
	}
	if got := statusOf(t, app, "/api/v1/blog/posts"); got != fiber.StatusTooManyRequests {
		t.Fatalf("requête %d : status %d, attendu 429", GlobalRateLimit+1, got)
	}
}

// /health est la sonde du healthcheck docker et des moniteurs : un 429 dessus marque le backend
// `unhealthy` sans qu'il le soit. Elle ne doit ni être refusée, ni consommer le quota.
func TestGlobalRateLimit_HealthIsExempt(t *testing.T) {
	app, mr := newGlobalRateLimitApp(t)
	mr.Set("ratelimit:global:", "1000")
	mr.SetTTL("ratelimit:global:", GlobalRateWindow)

	for _, p := range []string{"/health", "/health/deep"} {
		if got := statusOf(t, app, p); got != fiber.StatusOK {
			t.Fatalf("%s : status %d sous quota épuisé, attendu 200", p, got)
		}
	}
	if v, _ := mr.Get("ratelimit:global:"); v != "1000" {
		t.Fatalf("/health a consommé le quota (compteur=%s)", v)
	}
}
