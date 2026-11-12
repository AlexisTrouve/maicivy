package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"maicivy/internal/models"
)

func TestTrackingMiddleware_NewVisitor(t *testing.T) {
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient, "test-secret")

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	cookies := resp.Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, SessionCookieName, cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
}

func TestTrackingMiddleware_ReturningVisitor(t *testing.T) {
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient, "test-secret")

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, _ := app.Test(req1)
	sessionCookie := resp1.Cookies()[0]

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(sessionCookie)
	resp2, err := app.Test(req2)
	require.NoError(t, err)

	var body map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&body)

	// Nouvelle sémantique (anti-amplification) : la 1re requête anonyme (sans cookie signé) n'est
	// PAS comptée ; le compteur démarre à 1 quand notre cookie signé revient (req2 = 1re persistée).
	assert.Equal(t, float64(1), body["visit_count"])
}

func TestDetectProfile_LinkedIn(t *testing.T) {
	tm := &TrackingMiddleware{}

	app := fiber.New()
	// AcquireCtx nécessite un *fasthttp.RequestCtx, pas un *fiber.Ctx
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("User-Agent", "LinkedInBot/1.0")
	c := app.AcquireCtx(&fctx)
	defer app.ReleaseCtx(c)

	profile := tm.detectProfile(c)
	assert.Equal(t, "linkedin_bot", profile)
}

// OFR #1 : un visiteur anonyme ne doit JAMAIS pouvoir faire écrire maicivy en base. 50 cookies
// forgés → 0 ligne `visitors` (avant le fix : 50 INSERT synchrones, un par cookie inconnu).
func TestTracking_ForgedCookies_NoVisitorRows(t *testing.T) {
	db, rc := setupTestDB(t)
	tm := NewTracking(db, rc, "test-secret")
	app := fiber.New()
	app.Use(tm.Handler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: fmt.Sprintf("forged-%d", i)})
		_, _ = app.Test(req)
	}

	var count int64
	db.Model(&models.Visitor{}).Count(&count)
	assert.Equal(t, int64(0), count, "des cookies forgés ne doivent créer aucune ligne visiteur")
}

// OFR #3 : un vrai visiteur (cookie signé qui revient) = exactement 1 ligne. Le 1er contact
// anonyme (sans cookie) ne persiste rien ; la persistance arrive quand notre cookie signé revient.
func TestTracking_ValidReturningCookie_PersistsOnce(t *testing.T) {
	db, rc := setupTestDB(t)
	const secret = "test-secret"
	tm := NewTracking(db, rc, secret)
	app := fiber.New()
	app.Use(tm.Handler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// Requête 1 : pas de cookie → session émise, AUCUNE persistance.
	resp1, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	var c0 int64
	db.Model(&models.Visitor{}).Count(&c0)
	assert.Equal(t, int64(0), c0, "le 1er contact anonyme ne doit rien persister")

	// Récupérer le cookie signé émis.
	var sess string
	for _, ck := range resp1.Cookies() {
		if ck.Name == SessionCookieName {
			sess = ck.Value
		}
	}
	assert.NotEmpty(t, sess, "un cookie de session doit être émis")
	assert.True(t, verifySession(sess, secret), "le cookie émis doit être valide")

	// Requêtes 2 et 3 : cookie valide qui revient → exactement 1 ligne (créée puis mise à jour).
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sess})
		_, _ = app.Test(req)
	}

	var c1 int64
	db.Model(&models.Visitor{}).Count(&c1)
	assert.Equal(t, int64(1), c1, "un visiteur réel qui revient = exactement 1 ligne")
}
