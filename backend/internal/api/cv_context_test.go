package api

import (
	"context"
	"encoding/json"
	"io"
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
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/dossiervivant"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// QUOI : les routes du CV adapté au besoin du visiteur, et les deux portes qui y déposent un besoin
// (outil du chat public, run du dossier vivant).
//   - sans cookie, ou cookie non reconnu par le tracking → `none`, aucune génération ;
//   - session signée sans besoin → `none` (la page garde le CV standard) ;
//   - besoin noté → UNE génération en fond, `pending` puis `ready`, PDF rendu depuis le JSON ;
//   - les vrais plafonds visiteurs (middleware.AIBudget sur Redis) bornent la génération.
// COMMENT : vrai CVContextService + vrai AIBudget sur miniredis ; seul le générateur (l'appel IA) et
// le rendu Chromium sont faux. Le middleware `trackedAs` reproduit ce que fait le tracking pour un
// cookie VÉRIFIÉ : il recopie la session dans les locals.

const ctxSession = "11111111-2222-3333-4444-555555555555.sig"

type countingGen struct{ calls atomic.Int32 }

func (g *countingGen) GenerateDynamicCV(_ context.Context, _, lang string) (*services.AdaptiveCVResponse, error) {
	g.calls.Add(1)
	return &services.AdaptiveCVResponse{Theme: config.CVTheme{ID: "dynamic", Name: "Lead Go " + lang}}, nil
}

// trackedAs : simule le tracking. verified=true ↔ le cookie reçu est signé par nous.
func trackedAs(verified bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if verified {
			c.Locals("session_id", c.Cookies(middleware.SessionCookieName))
		} else {
			c.Locals("session_id", "session-neuve-emise-par-le-tracking")
		}
		return c.Next()
	}
}

type ctxFixture struct {
	app    *fiber.App
	gen    *countingGen
	svc    *services.CVContextService
	redis  *redis.Client
	mr     *miniredis.Miniredis
	render atomic.Int32
}

func newCtxFixture(t *testing.T, verified bool) *ctxFixture {
	t.Helper()
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	f := &ctxFixture{gen: &countingGen{}, redis: r, mr: mr}
	f.svc = services.NewCVContextService(r, f.gen, middleware.AIBudget{Redis: r, MaxPerDay: 5, MaxPerDayPerIP: 3, GlobalDailyMax: 200})

	h := &CVHandler{}
	h.SetContext(f.svc, func(cv *services.AdaptiveCVResponse, lang string) ([]byte, error) {
		f.render.Add(1)
		return []byte("%PDF-1.4 " + cv.Theme.Name), nil
	})
	f.app = fiber.New()
	f.app.Use(trackedAs(verified))
	f.app.Get("/api/v1/cv/context", h.GetContext)
	f.app.Get("/api/v1/cv/context/pdf", h.GetContextPDF)
	return f
}

func (f *ctxFixture) get(t *testing.T, path string, withCookie bool) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if withCookie {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: ctxSession})
	}
	resp, err := f.app.Test(req, -1)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func (f *ctxFixture) view(t *testing.T, lang string) services.CVContextView {
	t.Helper()
	resp, body := f.get(t, "/api/v1/cv/context?lang="+lang, true)
	require.Equal(t, fiber.StatusOK, resp.StatusCode, string(body))
	assert.Equal(t, "private, no-store", resp.Header.Get("Cache-Control"), "réponse liée au cookie : jamais en cache partagé")
	var v services.CVContextView
	require.NoError(t, json.Unmarshal(body, &v))
	return v
}

const ctxNeed = "Nous cherchons un développeur Go pour industrialiser une plateforme d'agents IA."

func TestCVContext_SansCookie_Rien(t *testing.T) {
	f := newCtxFixture(t, true)
	require.NoError(t, f.svc.RecordNeed(context.Background(), services.CVVisitor{SessionID: ctxSession, IP: "0.0.0.0"}, services.NeedSourceChat, ctxNeed, "fr"))
	time.Sleep(50 * time.Millisecond) // laisse finir la génération lancée par RecordNeed
	before := f.gen.calls.Load()

	resp, body := f.get(t, "/api/v1/cv/context?lang=fr", false)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"none"}`, string(body), "sans cookie, rien ne fuit — même si un besoin existe ailleurs")
	resp, _ = f.get(t, "/api/v1/cv/context/pdf?lang=fr", false)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.Equal(t, before, f.gen.calls.Load(), "aucune génération pour un visiteur sans cookie")
}

func TestCVContext_CookieNonVerifie_Rien(t *testing.T) {
	f := newCtxFixture(t, false)
	resp, body := f.get(t, "/api/v1/cv/context?lang=fr", true)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"status":"none"}`, string(body))
	assert.Empty(t, f.mr.Keys(), "un cookie forgé ne fait rien écrire")
}

func TestCVContext_SansBesoin_Standard(t *testing.T) {
	f := newCtxFixture(t, true)
	assert.Equal(t, services.CVContextNone, f.view(t, "fr").Status)
	resp, _ := f.get(t, "/api/v1/cv/context/pdf?lang=fr", true)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.Zero(t, f.gen.calls.Load())
}

func TestCVContext_BesoinDuChat_GenereUneFoisPuisPDF(t *testing.T) {
	f := newCtxFixture(t, true)

	// Le besoin arrive par l'outil du chat public (hook réel du ChatHandler).
	chat := &ChatHandler{}
	chat.SetNeedRecorder(f.svc)
	app := fiber.New()
	app.Use(trackedAs(true))
	app.Post("/note", func(c *fiber.Ctx) error {
		hook := chat.noteNeedHook(c, false, nil, services.ChatCapabilities{})
		require.NotNil(t, hook)
		hook(ctxNeed, "fr")
		hook(ctxNeed, "fr") // le modèle rappelle l'outil avec le même besoin
		return c.SendStatus(fiber.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/note", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: ctxSession})
	_, err := app.Test(req, -1)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return f.view(t, "fr").Status == services.CVContextReady }, 3*time.Second, 20*time.Millisecond)
	v := f.view(t, "fr")
	assert.Equal(t, services.NeedSourceChat, v.Source)
	assert.Equal(t, ctxNeed, v.Need)
	require.NotNil(t, v.CV)
	assert.Equal(t, "Lead Go fr", v.CV.Theme.Name)
	assert.EqualValues(t, 1, f.gen.calls.Load(), "une seule génération par session et par besoin")

	// Le budget visiteur réel a bougé (session, IP, global).
	n, _ := f.redis.Get(context.Background(), "ratelimit:ai:"+ctxSession+":daily").Int()
	assert.Equal(t, 1, n)
	g, _ := f.redis.Get(context.Background(), "ratelimit:ai:global:daily").Int()
	assert.Equal(t, 1, g)

	resp, body := f.get(t, "/api/v1/cv/context/pdf?lang=fr", true)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "cv_adapte_fr.pdf")
	assert.Equal(t, "%PDF-1.4 Lead Go fr", string(body))
	assert.EqualValues(t, 1, f.gen.calls.Load(), "le PDF se rend depuis le JSON stocké, sans IA")
}

func TestCVContext_PlafondVisiteurAtteint_Limited(t *testing.T) {
	f := newCtxFixture(t, true)
	// Le visiteur a déjà consommé ses 5 générations du jour (lettres, CV…).
	require.NoError(t, f.redis.Set(context.Background(), "ratelimit:ai:"+ctxSession+":daily", 5, time.Hour).Err())
	require.NoError(t, f.svc.RecordNeed(context.Background(), services.CVVisitor{SessionID: ctxSession, IP: "0.0.0.0"}, services.NeedSourceChat, ctxNeed, "fr"))

	assert.Equal(t, services.CVContextLimited, f.view(t, "fr").Status)
	assert.Zero(t, f.gen.calls.Load(), "plafond atteint → aucune génération en fond")
}

func TestCVContext_TTL(t *testing.T) {
	f := newCtxFixture(t, true)
	require.NoError(t, f.svc.RecordNeed(context.Background(), services.CVVisitor{SessionID: ctxSession, IP: "0.0.0.0"}, services.NeedSourceChat, ctxNeed, "fr"))
	require.Eventually(t, func() bool { return f.view(t, "fr").Status == services.CVContextReady }, 3*time.Second, 20*time.Millisecond)

	f.mr.FastForward(services.CVContextTTL + time.Second)
	assert.Equal(t, services.CVContextNone, f.view(t, "fr").Status, "24 h après, le besoin et le CV ont disparu")
	resp, _ := f.get(t, "/api/v1/cv/context/pdf?lang=fr", true)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Le hook n'est offert qu'à un VISITEUR à session signée.
func TestChatNoteNeedHook_Portee(t *testing.T) {
	rec := &fakeNeedRecorder{}
	h := &ChatHandler{needs: rec}
	cases := map[string]struct {
		verified bool
		owner    bool
		prospect *middleware.ProspectIdentity
		caps     services.ChatCapabilities
		want     bool
	}{
		"visiteur signé":       {verified: true, want: true},
		"session non vérifiée": {verified: false},
		"owner":                {verified: true, owner: true},
		"prospecteur":          {verified: true, prospect: &middleware.ProspectIdentity{ID: "p1"}},
		"composer":             {verified: true, caps: services.ChatCapabilities{Compose: true}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			app := fiber.New()
			app.Use(trackedAs(tc.verified))
			var got bool
			app.Get("/", func(c *fiber.Ctx) error {
				got = h.noteNeedHook(c, tc.owner, tc.prospect, tc.caps) != nil
				return nil
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: ctxSession})
			_, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
	assert.Nil(t, (&ChatHandler{}).noteNeedHook(nil, false, nil, services.ChatCapabilities{}), "sans recorder branché : jamais offert")
}

type fakeNeedRecorder struct {
	mu    sync.Mutex
	calls []services.CVVisitor
	needs []string
	src   []string
}

func (r *fakeNeedRecorder) RecordNeed(_ context.Context, v services.CVVisitor, source, summary, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, v)
	r.needs = append(r.needs, summary)
	r.src = append(r.src, source)
	return nil
}

// Le dossier vivant : un besoin VALIDE d'un visiteur signé est rangé (le besoin, jamais le nom) ;
// une saisie refusée ou un visiteur sans cookie ne range rien.
func TestDossierVivant_RangeLeBesoinPourLeCV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(refuseAll))
	t.Cleanup(srv.Close)
	ai, err := services.NewAIService(&config.AIConfig{AnthropicAPIKey: "eai_test", AnthropicBaseURL: srv.URL,
		MaxRequestsPerMinute: 60, RequestTimeout: 5 * time.Second}, nil)
	require.NoError(t, err)
	svc := dossiervivant.NewService(ai, emptyMontagne{}, dossiervivant.Options{Model: "claude-haiku-5-5", Beat: -1})

	run := func(t *testing.T, body string, withCookie bool) *fakeNeedRecorder {
		rec := &fakeNeedRecorder{}
		h := NewDossierVivantHandler(svc, services.NewDossierStore(memRedis(t)), dvSecret, dvKey)
		h.SetNeedRecorder(rec)
		app := fiber.New()
		app.Use(trackedAs(true))
		h.RegisterPublicRoutes(app.Group("/api/v1"))
		req := dvRequest(dvPublicPath, body, nil)
		if withCookie {
			req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: ctxSession})
		}
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		_, _ = io.ReadAll(resp.Body)
		return rec
	}

	rec := run(t, dvBody, true)
	require.Len(t, rec.calls, 1)
	assert.Equal(t, ctxSession, rec.calls[0].SessionID)
	assert.Equal(t, services.NeedSourceDossier, rec.src[0])
	assert.Equal(t, "Six personnes traitent 1 400 tickets support par jour.", rec.needs[0])
	assert.NotContains(t, rec.needs[0], "Inès", "la présentation du visiteur n'est jamais rangée")

	assert.Empty(t, run(t, dvBody, false).calls, "sans cookie : rien")
	assert.Empty(t, run(t, `{"need":"aide-moi","who":"Inès Morel, CTO chez Tessera","lang":"fr"}`, true).calls, "saisie refusée : rien")
}
