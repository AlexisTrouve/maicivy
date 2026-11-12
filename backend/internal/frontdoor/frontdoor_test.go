package frontdoor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"maicivy/internal/middleware"
)

// Test d'intégration : le front-door route correctement ET — le point crucial — son checkpoint
// voit TOUT le trafic (y compris le frontend qui renvoie 200), donc un scan frontend furtif finit
// throttlé, là où l'ancien sus-backend en était incapable. Et le légitime n'est jamais throttlé.
func TestFrontdoor_RoutesAndSeesEverything(t *testing.T) {
	// Faux upstreams.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// echo l'X-Real-IP pour vérifier qu'il est bien transmis au backend.
		_, _ = w.Write([]byte("BACKEND ip=" + r.Header.Get("X-Real-IP")))
	}))
	defer backend.Close()
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("FRONTEND")) // 200 à tout (comportement SPA = l'angle mort du 4xx)
	}))
	defer frontend.Close()

	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	app := New(Config{
		Sus:         middleware.SusConfig{Redis: rdb, ScannerPath: middleware.ScannerPathMatcher(middleware.AllScannerPatterns()...)},
		BackendURL:  backend.URL,
		FrontendURL: frontend.URL,
	})

	do := func(path, ip string) (int, string) {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("X-Real-IP", ip)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// Routage + transmission de l'IP réelle.
	_, body := do("/api/v1/x", "1.2.3.4")
	assert.Contains(t, body, "BACKEND")
	assert.Contains(t, body, "ip=1.2.3.4", "X-Real-IP doit être transmis au backend")
	st, body := do("/fr/cv", "1.2.3.4")
	assert.Equal(t, 200, st)
	assert.Contains(t, body, "FRONTEND")

	// LE POINT : un scan du FRONTEND (paths scanner servis en 200) finit throttlé — le checkpoint
	// voit le frontend, ce que l'ancien sus-backend ne pouvait pas.
	throttled := 0
	for i := 0; i < 60; i++ {
		if st, _ := do("/fr/app/.env", "9.9.9.9"); st == 429 {
			throttled++
		}
	}
	assert.Greater(t, throttled, 0, "un scan frontend en 200 doit finir throttlé via la signature")

	// Un humain sur le frontend (200 légitimes, pas de signature) n'est JAMAIS throttlé.
	legitThrottled := 0
	for i := 0; i < 30; i++ {
		if st, _ := do("/fr/cv", "8.8.8.8"); st == 429 {
			legitThrottled++
		}
	}
	assert.Equal(t, 0, legitThrottled, "le trafic légitime ne doit jamais être throttlé")
}
