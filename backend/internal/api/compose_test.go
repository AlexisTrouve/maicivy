package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/middleware"
)

// QUOI : verrouille la garde owner de /admin/compose/verify.
//
// POURQUOI : cet endpoint traduit et analyse le brouillon d'un message client. Sans garde, n'importe
// qui pourrait faire tourner des appels modèle sur du texte arbitraire aux frais du propriétaire (le
// rate-limit visiteur ne s'applique pas aux routes /admin) — et lire ce qui est en cours de rédaction.
//
// COMMENT : requête sans cookie admin, puis avec un cookie forgé. Les deux doivent être refusées AVANT
// tout parsing de corps ou appel au service (d'où un service nil : s'il était touché, le test paniquerait).

func newComposeApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	// svc nil À DESSEIN : la garde doit rejeter avant d'atteindre le service.
	NewComposeHandler(nil, "secret-de-test").RegisterRoutes(app)
	return app
}

func postVerify(t *testing.T, app *fiber.App, cookie string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/admin/compose/verify",
		strings.NewReader(`{"draft":"Bonjour,\n\nAlexis"}`))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp.StatusCode
}

func TestComposeVerify_RefuseSansCookieAdmin(t *testing.T) {
	assert.Equal(t, fiber.StatusUnauthorized, postVerify(t, newComposeApp(t), ""))
}

func TestComposeVerify_RefuseCookieForge(t *testing.T) {
	assert.Equal(t, fiber.StatusUnauthorized,
		postVerify(t, newComposeApp(t), "maicivy_admin=admin:9999999999.signaturebidon"))
}

// --- Plafond des vérifications d'un PROSPECTEUR -------------------------------------------------
//
// QUOI : un prospecteur qui a épuisé son quota de vérifications reçoit 429 AVANT tout appel modèle ;
// sans quota câblé, il est refusé (échec fermé), jamais servi sans limite.
// COMMENT : l'identité est déposée dans les locals comme le fait AdminRoleGuard (clé "prospect") ;
// svc reste nil — s'il était atteint, le test paniquerait.

func newComposeProspectApp(t *testing.T, quota *middleware.ProspectVerifyQuota) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("prospect", middleware.ProspectIdentity{ID: "p1", Name: "Jeanne"})
		return c.Next()
	})
	h := NewComposeHandler(nil, "secret-de-test")
	h.SetProspectVerifyQuota(quota)
	h.RegisterRoutes(app)
	return app
}

func TestComposeVerify_ProspectAuPlafondRecoit429(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	quota := middleware.NewProspectVerifyQuota(rdb, 1)
	// Le seul appel du jour est déjà consommé.
	ok, err := quota.Consume(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, fiber.StatusTooManyRequests, postVerify(t, newComposeProspectApp(t, quota), ""))
}

func TestComposeVerify_ProspectSansQuotaRefuse(t *testing.T) {
	assert.Equal(t, fiber.StatusServiceUnavailable, postVerify(t, newComposeProspectApp(t, nil), ""))
}
