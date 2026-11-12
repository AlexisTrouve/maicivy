package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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

// Route publique des icônes de projets — de bout en bout : faux maiProFiles (la liste des projets
// affichés) + faux Gitea (httptest, jamais la production) + vrai service + vrai handler.
// Ce qu'elle doit garantir : icône servie, absence = 404 propre, dépôt exclu jamais servi ni lu,
// aucun moyen de lui faire lire un autre fichier du dépôt ni de nommer un dépôt.

func iconePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 220, 255
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// iconsApp — l'app de test et le journal des chemins demandés au faux Gitea.
func iconsApp(t *testing.T) (*fiber.App, func() []string) {
	t.Helper()
	icone := iconePNG(t)
	var mu sync.Mutex
	var chemins []string
	gitea := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		chemins = append(chemins, r.URL.Path)
		mu.Unlock()
		if r.Header.Get("Authorization") != "token jeton" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/repos/StillHammer/Avec/contents":
			_, _ = w.Write([]byte(`[{"name":"icon.png","type":"file","sha":"abcdef0123456789"},{"name":"secrets.env","type":"file","sha":"1"}]`))
		case "/api/v1/repos/StillHammer/Avec/media/icon.png":
			_, _ = w.Write(icone)
		case "/api/v1/repos/StillHammer/Sans/contents":
			_, _ = w.Write([]byte(`[{"name":"README.md","type":"file","sha":"2"}]`))
		case "/api/v1/repos/StillHammer/Prive/contents":
			_, _ = w.Write([]byte(`[{"name":"icon.png","type":"file","sha":"3"}]`))
		case "/api/v1/repos/StillHammer/Prive/media/icon.png":
			_, _ = w.Write(icone)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(gitea.Close)
	mpf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id":"avec","links":{"repo":"git.etheryale.com/StillHammer/Avec"}},
			{"id":"sans","links":{"repo":"git.etheryale.com/StillHammer/Sans"}},
			{"id":"prive","links":{"repo":"git.etheryale.com/StillHammer/Prive"}}
		]`))
	}))
	t.Cleanup(mpf.Close)

	svc := services.NewProjectIconService(services.NewMaiProFilesClientForTest(mpf.URL, mpf.Client()), nil, gitea.URL, "jeton", "StillHammer", "prive")
	app := fiber.New()
	NewProjectIconsHandler(svc).RegisterRoutes(app.Group("/api/v1"))
	return app, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), chemins...)
	}
}

func get(t *testing.T, app *fiber.App, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, url, nil), 10_000)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func TestProjectIconsRoute_Liste(t *testing.T) {
	app, _ := iconsApp(t)
	resp, body := get(t, app, "/api/v1/project-icons")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out struct{ Icons map[string]string }
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, map[string]string{"avec": "/api/v1/project-icons/avec?v=abcdef012345"}, out.Icons)
	// Aucun nom de dépôt dans la réponse (ici, ils diffèrent des ids par la casse).
	require.NotContains(t, string(body), "Avec")
	require.NotContains(t, string(body), "Prive")
}

func TestProjectIconsRoute_IconeServie(t *testing.T) {
	app, _ := iconsApp(t)
	resp, body := get(t, app, "/api/v1/project-icons/avec?v=abcdef012345")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Contains(t, resp.Header.Get("Cache-Control"), "immutable", "URL versionnée : cache long")
	img, err := png.Decode(bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, 128, img.Bounds().Dx(), "réduite par le backend, pas l'original de 300 px")
	require.Equal(t, color.NRGBAModel.Convert(img.At(5, 5)).(color.NRGBA).R, uint8(220))
	// Sans la bonne version : servie aussi, mais pas « immutable ».
	resp, _ = get(t, app, "/api/v1/project-icons/avec")
	require.NotContains(t, resp.Header.Get("Cache-Control"), "immutable")
}

func TestProjectIconsRoute_SansIcone404(t *testing.T) {
	app, _ := iconsApp(t)
	for _, id := range []string{"sans", "inconnu"} {
		resp, body := get(t, app, "/api/v1/project-icons/"+id)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, id)
		require.JSONEq(t, `{"error":"icône introuvable"}`, string(body))
	}
}

func TestProjectIconsRoute_DepotExcluJamaisServiNiLu(t *testing.T) {
	app, chemins := iconsApp(t)
	resp, body := get(t, app, "/api/v1/project-icons/prive")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.JSONEq(t, `{"error":"icône introuvable"}`, string(body), "même réponse qu'un projet sans icône")
	_, list := get(t, app, "/api/v1/project-icons")
	require.NotContains(t, string(list), "prive")
	for _, c := range chemins() {
		require.NotContains(t, c, "/Prive", "le dépôt exclu n'est jamais lu")
	}
}

func TestProjectIconsRoute_AucunCheminArbitraire(t *testing.T) {
	app, chemins := iconsApp(t)
	for _, url := range []string{
		"/api/v1/project-icons/avec/secrets.env",
		"/api/v1/project-icons/..%2Fsecrets.env",
		"/api/v1/project-icons/avec%2Fsecrets.env",
		"/api/v1/project-icons/Avec", // un NOM DE DÉPÔT n'est pas un id de projet
		"/api/v1/project-icons/avec.png",
		"/api/v1/project-icons/avec?path=secrets.env&repo=Prive",
	} {
		resp, body := get(t, app, url)
		if strings.Contains(url, "?path=") {
			// Paramètres ignorés : c'est l'icône de « avec », rien d'autre.
			require.Equal(t, http.StatusOK, resp.StatusCode)
			_, err := png.Decode(bytes.NewReader(body))
			require.NoError(t, err, url)
			continue
		}
		require.Equal(t, http.StatusNotFound, resp.StatusCode, url)
	}
	for _, c := range chemins() {
		require.NotContains(t, c, "secrets.env", "aucun autre fichier du dépôt n'est demandé à Gitea")
		require.NotContains(t, c, "/Prive")
	}
}
