package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// fauxMaiProFiles : sert un contenu distinct par langue et note chaque ?lang= reçu.
// POURQUOI : prouver de bout en bout (client HTTP → provider) que zh part en lang=zh et revient en
// chinois, et que de/it ne déclenchent aucun appel propre (repli sur le cache anglais).
func fauxMaiProFiles(t *testing.T) (*httptest.Server, func() map[string]bool) {
	var mu sync.Mutex
	vus := map[string]bool{}
	titres := map[string]string{"fr": "Moteur de jeu", "en": "Game engine", "zh": "游戏引擎"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		mu.Lock()
		vus[lang] = true
		mu.Unlock()
		titre := titres[lang]
		var body any
		switch r.URL.Path {
		case "/experiences":
			body = []map[string]any{{"id": "e1", "title": titre, "company": "X", "start_date": "2024-01", "description": map[string]string{"short": titre}}}
		case "/skills":
			body = []map[string]any{{"name": "Go", "level": "expert", "description": titre}}
		case "/projects":
			body = []map[string]any{{"id": "p1", "name": titre, "description": map[string]string{"short": titre}}}
		case "/projects/index":
			body = []map[string]any{{"id": "p1", "mini": titre}}
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv, func() map[string]bool {
		mu.Lock()
		defer mu.Unlock()
		return vus
	}
}

func TestContentProvider_ZhNatifDeItEnAnglais(t *testing.T) {
	srv, vus := fauxMaiProFiles(t)
	p := NewMaiProFilesContentProvider(nil)
	p.client = NewMaiProFilesClientForTest(srv.URL, srv.Client())

	zh := p.GetProjects("zh")
	require.Len(t, zh, 1)
	require.Equal(t, "游戏引擎", zh[0].Title)
	require.Equal(t, "游戏引擎", zh[0].Catchphrase)
	require.Equal(t, "游戏引擎", p.GetExperiences("zh")[0].Title)
	require.Equal(t, "游戏引擎", p.GetSkills("zh")[0].Description)
	require.True(t, vus()["zh"], "une requête lang=zh doit partir vers maiProFiles")

	// de/it : toujours l'anglais, jamais le français ni le chinois.
	for _, l := range []string{"de", "it"} {
		require.Equal(t, "Game engine", p.GetProjects(l)[0].Title, l)
		require.False(t, vus()[l], "aucune requête lang=%s : repli sur le cache en", l)
	}
	require.Equal(t, "Moteur de jeu", p.GetProjects("fr")[0].Title)
	require.Equal(t, "Game engine", p.GetProjects("en")[0].Title)
}
