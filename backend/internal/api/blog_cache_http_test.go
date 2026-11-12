package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// ============================================================================
// Blog : codes HTTP et invalidation, de bout en bout (Fiber + faux maiProFiles + miniredis).
// ----------------------------------------------------------------------------
// POURQUOI : avant, GET /blog/posts/:slug répondait 404 pour TOUTE erreur — une panne de
// maiProFiles faisait passer des articles existants pour supprimés. Et aucune écriture de l'owner
// ne vidait un cache persistant (il n'y en avait pas). Ces tests verrouillent les deux.
// ============================================================================

// fauxMPF — faux maiProFiles : un article « premier », des 404 pour le reste, panne pilotable.
type fauxMPF struct {
	mu      sync.Mutex
	enPanne bool
	titre   string
}

func (f *fauxMPF) serveur(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		panne, titre := f.enPanne, f.titre
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case panne:
			w.WriteHeader(http.StatusBadGateway)
		case r.Method == http.MethodPost && r.URL.Path == "/blog/posts/1/publish":
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.URL.Path == "/blog/posts":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"posts": []map[string]interface{}{{"id": 1, "slug": "premier", "title": titre}},
				"total": 1, "page": 1, "per_page": 10,
			})
		case r.URL.Path == "/blog/posts/premier":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "slug": "premier", "title": titre})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fauxMPF) regler(panne bool, titre string) {
	f.mu.Lock()
	f.enPanne, f.titre = panne, titre
	f.mu.Unlock()
}

func appBlog(t *testing.T) (*fiber.App, *fauxMPF) {
	t.Helper()
	faux := &fauxMPF{titre: "v1"}
	srv := faux.serveur(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	client := services.NewMaiProFilesClientForTest(srv.URL, srv.Client())
	h := NewBlogHandler(client, nil, rdb, nil, "cle-owner")
	app := fiber.New()
	h.RegisterRoutes(app.Group("/api/v1"))
	return app, faux
}

func appeler(t *testing.T, app *fiber.App, method, url string, entetes map[string]string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	for k, v := range entetes {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	corps, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(corps)
}

func TestBlogHTTP_ArticleInexistantRend404(t *testing.T) {
	app, _ := appBlog(t)
	code, _ := appeler(t, app, "GET", "/api/v1/blog/posts/fantome?lang=fr", nil)
	require.Equal(t, http.StatusNotFound, code)
}

func TestBlogHTTP_PanneSansCacheRend502PasUn404(t *testing.T) {
	app, faux := appBlog(t)
	faux.regler(true, "v1")
	code, _ := appeler(t, app, "GET", "/api/v1/blog/posts/premier?lang=fr", nil)
	require.Equal(t, http.StatusBadGateway, code, "une panne amont n'est pas un article supprimé")
	code, _ = appeler(t, app, "GET", "/api/v1/blog/posts?lang=fr", nil)
	require.Equal(t, http.StatusBadGateway, code)
}

func TestBlogHTTP_PanneAvecCacheServiLArticle(t *testing.T) {
	app, faux := appBlog(t)
	code, _ := appeler(t, app, "GET", "/api/v1/blog/posts/premier?lang=fr", nil)
	require.Equal(t, http.StatusOK, code)

	faux.regler(true, "v1")
	code, corps := appeler(t, app, "GET", "/api/v1/blog/posts/premier?lang=fr", nil)
	require.Equal(t, http.StatusOK, code, "maiProFiles tombé : la version en cache doit être servie")
	require.Contains(t, corps, `"slug":"premier"`)
}

func TestBlogHTTP_PublicationOwnerInvalideLeCache(t *testing.T) {
	app, faux := appBlog(t)
	_, corps := appeler(t, app, "GET", "/api/v1/blog/posts?lang=fr", nil)
	require.Contains(t, corps, `"title":"v1"`)

	faux.regler(false, "v2")
	code, _ := appeler(t, app, "POST", "/api/v1/blog/posts/1/publish", map[string]string{"X-Owner-Key": "cle-owner"})
	require.Equal(t, http.StatusOK, code)

	_, corps = appeler(t, app, "GET", "/api/v1/blog/posts?lang=fr", nil)
	require.Contains(t, corps, `"title":"v2"`, "après une publication, la liste doit refléter la nouvelle version tout de suite")
}

func TestBlogHTTP_EcritureRefuseeNInvalidePas(t *testing.T) {
	app, faux := appBlog(t)
	_, _ = appeler(t, app, "GET", "/api/v1/blog/posts?lang=fr", nil)
	faux.regler(false, "v2")

	code, _ := appeler(t, app, "POST", "/api/v1/blog/posts/1/publish", map[string]string{"X-Owner-Key": "mauvaise"})
	require.Equal(t, http.StatusUnauthorized, code)

	_, corps := appeler(t, app, "GET", "/api/v1/blog/posts?lang=fr", nil)
	require.Contains(t, corps, `"title":"v1"`, "une requête refusée ne doit rien invalider")
}
