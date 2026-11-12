package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLangSuffix verrouille le repli de langue du client maiProFiles.
// CONTEXTE : maiProFiles ne contient du contenu QUE pour fr/en/ka. Le front expose fr/en/de/it/zh.
// AVANT : une langue non servie (de/it/zh) ne recevait PAS de param ?lang= → maiProFiles servait son
// défaut FRANÇAIS → un visiteur allemand voyait des fiches en français (incohérent avec l'UI/voix DE).
// MAINTENANT (décision B) : une langue non servie ET non vide replie sur l'ANGLAIS (neutre/international).
// EXCEPTION : lang="" reste le « défaut API » (FR) — utilisé par des appels internes non liés à une
// locale visiteur (stats numériques, profil du prompt système) qu'on ne veut pas modifier.
func TestLangSuffix(t *testing.T) {
	cases := []struct {
		name string
		path string
		lang string
		want string
	}{
		// Langues réellement servies par maiProFiles → passées telles quelles.
		{"fr servi tel quel", "/projects", "fr", "/projects?lang=fr"},
		{"en servi tel quel", "/projects", "en", "/projects?lang=en"},
		{"ka servi tel quel", "/projects", "ka", "/projects?lang=ka"},
		{"zh servi tel quel (natif depuis que maiProFiles traduit en chinois)", "/projects", "zh", "/projects?lang=zh"},

		// Langues du front non servies par maiProFiles → repli ANGLAIS (et surtout PAS le défaut FR).
		{"de → fallback en", "/projects", "de", "/projects?lang=en"},
		{"it → fallback en", "/projects", "it", "/projects?lang=en"},

		// lang vide = défaut API (FR) pour les appels internes → aucun param ajouté.
		{"vide → défaut API (pas de param)", "/projects", "", "/projects"},

		// Séparateur correct quand le path porte déjà des query params.
		{"non servi + query existante → &", "/blog/posts?page=1&per_page=10", "de", "/blog/posts?page=1&per_page=10&lang=en"},
		{"servi + query existante → &", "/blog/posts?page=1", "fr", "/blog/posts?page=1&lang=fr"},
		{"vide + query existante → inchangé", "/blog/posts?page=1", "", "/blog/posts?page=1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := langSuffix(tc.path, tc.lang)
			if got != tc.want {
				t.Errorf("langSuffix(%q, %q) = %q, want %q", tc.path, tc.lang, got, tc.want)
			}
		})
	}
}

// --- Index projets pour agents (GET /projects/index) -------------------------
//
// POURQUOI cet index existe : /projects sert ~28 000 caractères rien qu'en descriptions longues.
// Un agent qui cherche « lequel de ces projets fait du Rust » payait tout l'inventaire avant de
// pouvoir trancher. L'index ne garde que de quoi CHOISIR — le détail se lit ensuite sur UN projet.
//
// POURQUOI un test de dégradation : au moment où ce code est écrit, l'endpoint n'est pas encore
// déployé côté maiProFiles. Le chat doit continuer à fonctionner exactement comme avant, sans le
// bloc — pas planter, pas se vider.

func TestGetProjectsIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/index" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"aissia","name":"AISSIA","mini":"Assistant IA en C++17 — 7 modules hot-reload",
			 "category":"Agent IA","status":"active","stack":["c++17","cmake","sqlite"]},
			{"id":"maicivy","name":"maicivy","mini":"Portfolio agentique en Go et Next.js",
			 "category":"Web","status":"production","stack":["go","next.js"]}
		]`))
	}))
	defer srv.Close()

	c := NewMaiProFilesClient()
	c.baseURL = srv.URL

	idx, err := c.GetProjectsIndex(context.Background(), "fr")
	require.NoError(t, err)
	require.Len(t, idx, 2)
	assert.Equal(t, "aissia", idx[0].ID)
	assert.Equal(t, "AISSIA", idx[0].Name)
	assert.Contains(t, idx[0].Mini, "hot-reload")
	assert.Equal(t, "active", idx[0].Status)
	assert.Equal(t, []string{"c++17", "cmake", "sqlite"}, idx[0].Stack)
}

// L'endpoint n'est pas encore déployé : l'erreur doit remonter proprement, sans panique, pour que
// l'appelant puisse simplement omettre le bloc.
func TestGetProjectsIndex_EndpointAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()

	c := NewMaiProFilesClient()
	c.baseURL = srv.URL

	idx, err := c.GetProjectsIndex(context.Background(), "fr")
	require.Error(t, err, "un endpoint absent doit remonter une erreur, pas un index vide silencieux")
	assert.Empty(t, idx)
}
