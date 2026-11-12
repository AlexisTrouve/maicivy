package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProspects : résolveur de test — le prospecteur est « actif » tant que active vaut vrai.
type fakeProspects struct{ active bool }

func (f *fakeProspects) ResolveProspect(c *fiber.Ctx) (ProspectIdentity, bool) {
	if !f.active || c.Cookies(ProspectCookieName) == "" {
		return ProspectIdentity{}, false
	}
	return ProspectIdentity{ID: "p1", Name: "Marc", Ref: "abc123"}, true
}

// Un jeton prospecteur ne valide JAMAIS comme owner, et inversement.
func TestProspectCookie_JamaisOwner(t *testing.T) {
	tok := SignProspectCookie("s", "p1", time.Hour)
	id, ok := VerifyProspectCookie(tok, "s")
	require.True(t, ok)
	assert.Equal(t, "p1", id)
	assert.False(t, VerifyAdminCookie(tok, "s"), "un cookie prospecteur ne doit jamais passer pour owner")

	_, ok = VerifyProspectCookie(SignAdminCookie("s", time.Hour), "s")
	assert.False(t, ok)
	_, ok = VerifyProspectCookie(SignProspectCookie("s", "p1", -time.Second), "s")
	assert.False(t, ok, "expiré")
	_, ok = VerifyProspectCookie(tok, "autre")
	assert.False(t, ok, "autre secret")
}

// Plafond quotidien : au-delà de ProspectDailyMax → 429 PROSPECT_DAILY_LIMIT ; avant, is_owner n'est
// jamais posé (donc pas d'Opus, et les compteurs sont bien incrémentés).
func TestAIRateLimit_ProspectPlafondEtJamaisOwner(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	app := fiber.New()
	app.Post("/gen", AIRateLimit(AIRateLimitConfig{
		Redis: rdb, MaxPerDay: 5, SessionSecret: "s",
		Prospects: &fakeProspects{active: true}, ProspectDailyMax: 2,
	}), func(c *fiber.Ctx) error {
		isOwner, _ := c.Locals("is_owner").(bool)
		assert.False(t, isOwner, "un prospecteur n'est jamais owner")
		_, ok := ProspectFromCtx(c)
		assert.True(t, ok)
		require.NoError(t, IncrementAIRateLimit(c, rdb, time.Minute))
		return c.SendStatus(200)
	})

	statuses := []int{}
	var last string
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/gen", nil)
		// Ni session ni IP partagée : seul le plafond personnel doit jouer.
		req.Header.Set("Cookie", ProspectCookieName+"=x")
		resp, err := app.Test(req)
		require.NoError(t, err)
		statuses = append(statuses, resp.StatusCode)
		last = resp.Header.Get("X-RateLimit-Type")
	}
	assert.Equal(t, []int{200, 200, 429}, statuses)
	assert.Equal(t, "prospect-daily", last)
}

// Plafond NÉGATIF = plafond levé (décision d'Alexi du 2026-10-15) : un prospecteur enchaîne les
// générations sans 429 PROSPECT_DAILY_LIMIT, et n'est toujours JAMAIS owner. Le compteur continue
// d'avancer (mesure d'usage), il ne bloque simplement plus.
func TestAIRateLimit_ProspectPlafondLeve(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	app := fiber.New()
	app.Post("/gen", AIRateLimit(AIRateLimitConfig{
		Redis: rdb, MaxPerDay: 5, SessionSecret: "s",
		Prospects: &fakeProspects{active: true}, ProspectDailyMax: -1,
	}), func(c *fiber.Ctx) error {
		isOwner, _ := c.Locals("is_owner").(bool)
		assert.False(t, isOwner, "un prospecteur n'est jamais owner, même sans plafond")
		require.NoError(t, IncrementAIRateLimit(c, rdb, time.Minute))
		return c.SendStatus(200)
	})

	for i := 0; i < 60; i++ {
		req := httptest.NewRequest("POST", "/gen", nil)
		req.Header.Set("Cookie", ProspectCookieName+"=x")
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode, "génération %d refusée alors que le plafond est levé", i+1)
	}
}

// Le garde de rôle : 403 hors liste, passage sur la liste, et rien ne change sans identité.
func TestAdminRoleGuard(t *testing.T) {
	fp := &fakeProspects{active: true}
	app := fiber.New()
	app.Use("/admin", AdminRoleGuard("s", fp, []AllowedRoute{{Method: "GET", Path: "/admin/ok/:id"}}))
	app.Get("/admin/ok/:id", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/admin/secret", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	get := func(path, cookie string) int {
		req := httptest.NewRequest("GET", path, nil)
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp.StatusCode
	}
	pc := ProspectCookieName + "=x"
	assert.Equal(t, 200, get("/admin/ok/42", pc))
	assert.Equal(t, 403, get("/admin/secret", pc))
	assert.Equal(t, 200, get("/admin/secret", "maicivy_admin="+SignAdminCookie("s", time.Hour)), "owner inchangé")
	fp.active = false
	assert.Equal(t, 200, get("/admin/secret", pc), "sans identité, le garde laisse la main au handler (401 owner-only)")
}
