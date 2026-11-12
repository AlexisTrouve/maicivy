package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// ============================================================================
// Durée de lecture des articles publiés via l'API (signalé par WanMira le 2026-10-11).
// ----------------------------------------------------------------------------
// Mesuré : tous les articles créés par POST /api/v1/blog/posts avaient reading_time_minutes = 0
// (devblog, drifterra, au moins depuis le 03/09), donc « 0 min read » sur la page. CreatePost ne
// calculait rien et le client maiProFiles sérialise le champ en omitempty : il n'était jamais
// envoyé. UpdatePost avait le même trou : un contenu réécrit gardait l'ancienne durée.
// ============================================================================

// fauxMPFEcriture : faux maiProFiles qui CAPTURE le corps des écritures.
type fauxMPFEcriture struct {
	mu    sync.Mutex
	corps map[string]map[string]interface{} // "POST /blog/posts" → body décodé
}

func (f *fauxMPFEcriture) serveur(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.corps[r.Method+" "+r.URL.Path] = body
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/blog/posts":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":7,"slug":"un-article","title":"Un article"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/blog/posts/7":
			_, _ = w.Write([]byte(`{"id":7,"slug":"un-article","title":"Un article"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fauxMPFEcriture) recu(cle string) map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.corps[cle]
}

func appBlogEcriture(t *testing.T) (*fiber.App, *fauxMPFEcriture) {
	t.Helper()
	t.Setenv("MAIPROFILES_API_KEY", "cle-mpf")
	faux := &fauxMPFEcriture{corps: map[string]map[string]interface{}{}}
	srv := faux.serveur(t)
	client := services.NewMaiProFilesClientForTest(srv.URL, srv.Client())
	h := NewBlogHandler(client, nil, nil, nil, "cle-owner")
	app := fiber.New()
	h.RegisterRoutes(app.Group("/api/v1"))
	return app, faux
}

func ecrire(t *testing.T, app *fiber.App, method, url, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Owner-Key", "cle-owner")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp.StatusCode
}

// motsDeContenu : un markdown de n mots, sérialisé en JSON.
func motsDeContenu(n int) string {
	b, _ := json.Marshal(strings.TrimSpace(strings.Repeat("mot ", n)))
	return string(b)
}

func TestCreatePost_EnvoieUneDureeDeLecture(t *testing.T) {
	app, faux := appBlogEcriture(t)

	code := ecrire(t, app, http.MethodPost, "/api/v1/blog/posts",
		`{"title":"Un article","content":`+motsDeContenu(1000)+`,"publish":false}`)
	require.Equal(t, http.StatusCreated, code)

	corps := faux.recu("POST /blog/posts")
	require.NotNil(t, corps, "la création doit atteindre maiProFiles")
	require.EqualValues(t, 5, corps["reading_time_minutes"], "1000 mots à 200 mots/min = 5 min")
}

func TestCreatePost_ArticleCourtVautAuMoinsUneMinute(t *testing.T) {
	app, faux := appBlogEcriture(t)
	ecrire(t, app, http.MethodPost, "/api/v1/blog/posts",
		`{"title":"Court","content":`+motsDeContenu(40)+`,"publish":false}`)
	require.EqualValues(t, 1, faux.recu("POST /blog/posts")["reading_time_minutes"], "jamais « 0 min read »")
}

func TestUpdatePost_RecalculeLaDureeQuandLeContenuChange(t *testing.T) {
	app, faux := appBlogEcriture(t)

	ecrire(t, app, http.MethodPut, "/api/v1/blog/posts/7", `{"content":`+motsDeContenu(600)+`}`)
	require.EqualValues(t, 3, faux.recu("PUT /blog/posts/7")["reading_time_minutes"])

	// Sans nouveau contenu, la durée n'est pas envoyée (update partiel : on n'écrase rien).
	ecrire(t, app, http.MethodPut, "/api/v1/blog/posts/7", `{"title":"Nouveau titre"}`)
	_, present := faux.recu("PUT /blog/posts/7")["reading_time_minutes"]
	require.False(t, present, "un update sans contenu ne doit pas toucher la durée de lecture")
}
