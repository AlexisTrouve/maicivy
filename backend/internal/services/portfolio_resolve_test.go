package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : résolution d'un projet à partir de l'identifiant fourni par l'agent.
//
// POURQUOI ces tests existent — INCIDENT DE PRODUCTION du 2026-08-01 : depuis que l'inventaire des
// projets est dans le prompt système, l'agent cite des identifiants exacts. Constaté sur le chat
// public (Haiku) : il a écrit `groove_engine` là où l'identifiant réel est `grove_engine`. UNE lettre.
//
// Conséquence avant correctif : aucune correspondance (ni exacte, ni par sous-chaîne, ni via
// /search) → « Projet non trouvé » → l'agent annonce à l'utilisateur qu'un projet qui EXISTE n'existe
// pas. Le pire cas possible pour un portfolio.
//
// Le correctif est MÉCANIQUE et non un durcissement de prompt : un modèle qui recopie un identifiant
// se trompera de nouveau, et on ne peut pas le tester. Une résolution tolérante, si.

func portfolioDeTest(t *testing.T) *PortfolioService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(`[
				{"id":"grove_engine","name":"GroveEngine"},
				{"id":"maicivy","name":"maicivy"},
				{"id":"aissia_celuna","name":"AISSIA (CELUNA)"}
			]`))
		case "/projects/grove_engine":
			_, _ = w.Write([]byte(`{"id":"grove_engine","name":"GroveEngine"}`))
		case "/projects/maicivy":
			_, _ = w.Write([]byte(`{"id":"maicivy","name":"maicivy"}`))
		case "/projects/aissia_celuna":
			_, _ = w.Write([]byte(`{"id":"aissia_celuna","name":"AISSIA (CELUNA)"}`))
		default:
			// /search ne trouve rien : on veut éprouver la résolution, pas le repli recherche.
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(srv.Close)

	c := NewMaiProFilesClient()
	c.baseURL = srv.URL
	return &PortfolioService{client: c}
}

// LE cas de l'incident : une lettre en trop dans l'identifiant.
func TestGetProject_ToleereUneCoquilleDIdentifiant(t *testing.T) {
	p := portfolioDeTest(t)

	entry, found := p.GetProject("groove_engine", "fr")
	require.True(t, found, "une coquille d'une lettre ne doit pas faire disparaître un projet existant")
	// PortfolioEntry.Name porte l'ID (slug), Title le nom affiché — cf. mapProject.
	assert.Equal(t, "grove_engine", entry.Name)
	assert.Equal(t, "GroveEngine", entry.Title)
}

// L'identifiant exact continue évidemment de marcher — la tolérance ne doit rien casser.
func TestGetProject_IdentifiantExact(t *testing.T) {
	p := portfolioDeTest(t)
	entry, found := p.GetProject("grove_engine", "fr")
	require.True(t, found)
	// PortfolioEntry.Name porte l'ID (slug), Title le nom affiché — cf. mapProject.
	assert.Equal(t, "grove_engine", entry.Name)
	assert.Equal(t, "GroveEngine", entry.Title)
}

// GARDE-FOU : la tolérance ne doit PAS inventer un projet. Un terme sans rapport reste introuvable —
// mieux vaut dire « je ne trouve pas » que de servir la fiche d'un autre projet avec assurance.
func TestGetProject_NInventePasUnProjet(t *testing.T) {
	p := portfolioDeTest(t)
	_, found := p.GetProject("kubernetes_operator_rust", "fr")
	assert.False(t, found, "un terme sans rapport ne doit correspondre à aucun projet")
}

// GARDE-FOU : deux candidats aussi proches l'un que l'autre = ambiguïté → on refuse plutôt que de
// tirer à pile ou face sur le projet qu'on montre à l'utilisateur.
func TestGetProject_RefuseSiAmbigu(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/projects" {
			_, _ = w.Write([]byte(`[{"id":"projet_a"},{"id":"projet_b"}]`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	c := NewMaiProFilesClient()
	c.baseURL = srv.URL
	p := &PortfolioService{client: c}

	// "projet_c" est à distance 1 de projet_a ET de projet_b : indécidable.
	_, found := p.GetProject("projet_c", "fr")
	assert.False(t, found, "deux candidats équidistants doivent être refusés, pas départagés au hasard")
}
