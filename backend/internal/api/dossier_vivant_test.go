package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/dossiervivant"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// QUOI : les portes du run du dossier vivant.
//   - POST /dossier-vivant/run (PUBLIQUE) : un visiteur SANS cookie ouvre le flux ; deux visiteurs
//     en même temps ont chacun leur run (aucun verrou « un run à la fois ») ; une saisie hors
//     bornes (besoin de 50 ko…) est refusée AVANT tout appel de modèle.
//   - POST /admin/dossier-vivant/run (OWNER) : rien ne s'ouvre pour qui n'est pas l'owner — ni
//     flux, ni goroutine, ni appel à la passerelle.
//
// COMMENT : une fausse passerelle compte les appels reçus. Toute requête refusée doit la laisser à
// zéro. Un run accepté doit ouvrir le flux SSE (run.started d'abord).

const (
	dvSecret = "secret-de-test"
	dvKey    = "cle-owner-de-test"

	dvPublicPath = "/api/v1/dossier-vivant/run"
	dvOwnerPath  = "/api/v1/admin/dossier-vivant/run"
)

type emptyMontagne struct{}

func (emptyMontagne) Montagne(context.Context, string) (*services.Montagne, error) {
	return &services.Montagne{}, nil
}
func (emptyMontagne) Project(context.Context, string, string) (*services.MPFProject, error) {
	return &services.MPFProject{}, nil
}
func (emptyMontagne) Book(context.Context, string) (*services.MPFBook, error) {
	return &services.MPFBook{}, nil
}

// refuseAll — la fausse passerelle qui refuse tout : un run accepté s'arrête donc tout de suite
// sur un run.failed nommé, ce qui suffit à prouver le flux.
func refuseAll(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"non"}}`))
}

// dvAppWith — l'app avec les deux routes, sur une fausse passerelle `gw` ; rend le compteur
// d'appels reçus par la passerelle.
func dvAppWith(t *testing.T, gw http.HandlerFunc) (*fiber.App, *atomic.Int32) {
	t.Helper()
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		gw(w, r)
	}))
	t.Cleanup(srv.Close)
	ai, err := services.NewAIService(&config.AIConfig{AnthropicAPIKey: "eai_test", AnthropicBaseURL: srv.URL,
		MaxRequestsPerMinute: 60, RequestTimeout: 5 * time.Second}, nil)
	require.NoError(t, err)
	svc := dossiervivant.NewService(ai, emptyMontagne{}, dossiervivant.Options{Model: "claude-haiku-5-5", Beat: -1})

	app := fiber.New()
	h := NewDossierVivantHandler(svc, services.NewDossierStore(memRedis(t)), dvSecret, dvKey)
	h.RegisterPublicRoutes(app.Group("/api/v1"))
	h.RegisterOwnerRoutes(app.Group("/api/v1"))
	return app, calls
}

func dvApp(t *testing.T) (*fiber.App, *atomic.Int32) { return dvAppWith(t, refuseAll) }

const dvBody = `{"need":"Six personnes traitent 1 400 tickets support par jour.","who":"Inès Morel, CTO chez Tessera","lang":"fr"}`

func dvRequest(path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// assertStreamOpened — le flux part par run.started et l'arrêt sur la passerelle refusante est
// NOMMÉ dans le flux, pas une coupure muette.
func assertStreamOpened(t *testing.T, resp *http.Response, calls *atomic.Int32) {
	t.Helper()
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	body, _ := io.ReadAll(resp.Body)
	stream := string(body)
	assert.True(t, strings.HasPrefix(stream, "event: run.started\n"), stream)
	assert.Contains(t, stream, "event: run.failed\ndata: {\"type\":\"run.failed\",\"reason\":\"upstream\"}")
	assert.Positive(t, calls.Load())
}

func TestDossierVivant_VisiteurSansCookieOuvreLeFlux(t *testing.T) {
	app, calls := dvApp(t)
	resp, err := app.Test(dvRequest(dvPublicPath, dvBody, nil), -1)
	require.NoError(t, err)
	assertStreamOpened(t, resp, calls)
}

// Deux visiteurs en même temps : chacun son run, aucun 409. La passerelle retient chaque appel
// tant que les DEUX runs ne l'ont pas atteinte — un verrou global laisserait le second dehors et
// le compteur bloqué à 1.
func TestDossierVivant_DeuxVisiteursEnMemeTemps(t *testing.T) {
	both := make(chan struct{})
	var once sync.Once
	var inFlight atomic.Int32
	app, calls := dvAppWith(t, func(w http.ResponseWriter, r *http.Request) {
		if inFlight.Add(1) >= 2 {
			once.Do(func() { close(both) })
		}
		select {
		case <-both:
		case <-time.After(3 * time.Second):
		}
		refuseAll(w, r)
	})

	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := app.Test(dvRequest(dvPublicPath, dvBody, nil), -1)
			if err != nil {
				statuses <- -1
				return
			}
			_, _ = io.ReadAll(resp.Body)
			statuses <- resp.StatusCode
		}()
	}
	select {
	case <-both:
	case <-time.After(3 * time.Second):
		t.Fatalf("les deux runs n'ont jamais été en vol ensemble (appels reçus : %d)", calls.Load())
	}
	wg.Wait()
	close(statuses)
	for s := range statuses {
		assert.Equal(t, fiber.StatusOK, s, "aucun visiteur n'est refusé parce qu'un autre run tourne")
	}
}

// La saisie est une entrée externe : bornée côté serveur, refusée avant le flux et avant tout
// appel de modèle.
func TestDossierVivant_SaisieHorsBornes(t *testing.T) {
	need := "Six personnes traitent 1 400 tickets support par jour."
	cases := map[string]struct{ need, who, code string }{
		"besoin de 50 ko":          {strings.Repeat("a", 50_000), "Inès Morel, CTO chez Tessera", "NEED_LENGTH"},
		"besoin de 2 001 car.":     {strings.Repeat("é", 2001), "Inès Morel, CTO chez Tessera", "NEED_LENGTH"},
		"besoin trop court":        {"aide-moi", "Inès Morel, CTO chez Tessera", "NEED_LENGTH"},
		"présentation sans rôle":   {need, "Inès Morel", "WHO_FORMAT"},
		"présentation de 121 car.": {need, "Inès Morel, " + strings.Repeat("x", 109), "WHO_FORMAT"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			app, calls := dvApp(t)
			body := `{"need":"` + tc.need + `","who":"` + tc.who + `","lang":"fr"}`
			resp, err := app.Test(dvRequest(dvPublicPath, body, nil), -1)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
			assert.NotEqual(t, "text/event-stream", resp.Header.Get("Content-Type"), "aucun flux ne s'ouvre")
			got, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(got), tc.code)
			assert.Zero(t, calls.Load(), "rien ne part au modèle")
		})
	}
}

// Bornes exactes : 2 000 caractères (pas octets — un besoin en chinois tient ses 2 000 signes)
// passent.
func TestDossierVivant_BesoinALaBorneAccepte(t *testing.T) {
	app, calls := dvApp(t)
	body := `{"need":"` + strings.Repeat("工", 2000) + `","who":"Inès Morel, CTO chez Tessera","lang":"zh"}`
	resp, err := app.Test(dvRequest(dvPublicPath, body, nil), -1)
	require.NoError(t, err)
	assertStreamOpened(t, resp, calls)
}

func TestDossierVivant_RouteOwnerRefuseToutNonOwner(t *testing.T) {
	expired := "admin:1." + "x" // forme valide, signature et date fausses
	cases := map[string]map[string]string{
		"sans authentification":   nil,
		"mauvaise clé owner":      {"X-Owner-Key": "pas-la-bonne"},
		"clé owner vide":          {"X-Owner-Key": ""},
		"cookie de session seul":  {"Cookie": "maicivy_session=visiteur"},
		"cookie admin forgé":      {"Cookie": "maicivy_admin=" + expired},
		"cookie admin autre clef": {"Cookie": "maicivy_admin=" + middleware.SignAdminCookie("autre-secret", time.Hour)},
		"cookie admin expiré":     {"Cookie": "maicivy_admin=" + middleware.SignAdminCookie(dvSecret, -time.Minute)},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			app, calls := dvApp(t)
			resp, err := app.Test(dvRequest(dvOwnerPath, dvBody, headers), -1)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			assert.NotEqual(t, "text/event-stream", resp.Header.Get("Content-Type"), "aucun flux ne s'ouvre")
			assert.Zero(t, calls.Load(), "aucun appel de modèle pour une requête refusée")
		})
	}
}

func TestDossierVivant_OwnerOuvreLeFlux(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"cookie admin":    {"Cookie": "maicivy_admin=" + middleware.SignAdminCookie(dvSecret, time.Hour)},
		"clé X-Owner-Key": {"X-Owner-Key": dvKey},
	} {
		t.Run(name, func(t *testing.T) {
			app, calls := dvApp(t)
			resp, err := app.Test(dvRequest(dvOwnerPath, dvBody, headers), -1)
			require.NoError(t, err)
			assertStreamOpened(t, resp, calls)
		})
	}
}
