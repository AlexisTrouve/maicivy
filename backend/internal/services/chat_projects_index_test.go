package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : le bloc « inventaire projets » injecté dans le prompt système du chat.
//
// POURQUOI il existe : sans lui, l'agent ignore TOUT des projets au démarrage. Une question aussi
// simple que « qu'est-ce qu'il a fait en Rust ? » exigeait un aller-retour d'outil avant de pouvoir
// seulement dire ce qui existe. Avec l'inventaire, il sait quoi proposer et ne fetche que le projet
// qu'il va réellement détailler.
//
// POURQUOI ces tests portent sur la TAILLE autant que sur le contenu : ce bloc part à CHAQUE tour de
// conversation. Mesuré sur les 79 projets réels — le JSON brut de /projects/index pèse ~9 600 tokens,
// le rendu `id | nom — mini` ~3 900. La différence est du budget pur, sur chaque message, pour la
// même information utile. Un rendu qui regrosserait sans qu'on le voie coûterait en silence.

func entreesDeTest(n int) []MPFProjectIndexEntry {
	out := make([]MPFProjectIndexEntry, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, MPFProjectIndexEntry{
			ID:       "projet_" + strings.Repeat("x", 6),
			Name:     "Projet de démonstration numéro " + strings.Repeat("y", 10),
			Mini:     "Une ligne qui dit quoi, avec quelle techno structurante, et ce qui le distingue.",
			Category: "Catégorie assez longue / sous-catégorie",
			Status:   "production",
			Stack:    []string{"go", "next.js"},
		})
	}
	return out
}

func TestRenderProjectsIndex_ContientLEssentiel(t *testing.T) {
	bloc := renderProjectsIndex([]MPFProjectIndexEntry{{
		ID: "aissia", Name: "AISSIA", Mini: "Assistant IA en C++17 — 7 modules hot-reload",
		Category: "Agent IA", Status: "active", Stack: []string{"c++17", "cmake"},
	}})

	// L'identifiant DOIT y être : c'est ce que l'agent passe ensuite pour demander le détail.
	assert.Contains(t, bloc, "aissia")
	assert.Contains(t, bloc, "AISSIA")
	assert.Contains(t, bloc, "hot-reload")
}

// Le bloc rend `id | nom [production] — mini`. Catégorie et stack restent ÉCARTÉES : le mini porte
// déjà la techno structurante, et réémettre la stack doublonnait pour 35 % du poids du bloc.
//
// Le STATUT, lui, a été réintégré après coup — je l'avais écarté avec les deux autres, à tort.
// Ce n'est pas une redondance technique mais une PREUVE DE RÉALITÉ : mesuré sur les 79 fiches, un
// seul mini dit « en production » alors que 39 projets le sont. C'est le seul signal disponible qui
// distingue « j'ai construit ça » de « ça tourne pour de vrai » — exactement ce qu'un client cherche.
// Coût : ~145 tokens.
func TestRenderProjectsIndex_RendIdNomStatutEtMini(t *testing.T) {
	bloc := renderProjectsIndex([]MPFProjectIndexEntry{{
		ID: "x", Name: "Projet X", Mini: "une ligne utile", Status: "production",
		Category: "CATEGORIE_A_NE_PAS_RENDRE",
		Stack:    []string{"STACK_A_NE_PAS_RENDRE"},
	}})

	assert.Equal(t, "x | Projet X [production] — une ligne utile", bloc)
	assert.NotContains(t, bloc, "CATEGORIE_A_NE_PAS_RENDRE")
	assert.NotContains(t, bloc, "STACK_A_NE_PAS_RENDRE")
}

// SEUL « production » est rendu. Les autres statuts (active, wip, R&D) n'apportent aucune preuve au
// client — « wip » en dirait même du mal — et occuperaient des tokens sur chaque message pour ça.
func TestRenderProjectsIndex_SeulProductionEstMarque(t *testing.T) {
	for _, statut := range []string{"active", "wip", "R&D", ""} {
		bloc := renderProjectsIndex([]MPFProjectIndexEntry{{
			ID: "x", Name: "Projet X", Mini: "m", Status: statut,
		}})
		assert.Equal(t, "x | Projet X — m", bloc, "statut %q ne doit pas être rendu", statut)
	}
}

// LE garde-fou de coût. 79 projets réels rendus doivent tenir sous un plafond franc.
func TestRenderProjectsIndex_ResteSousLePlafond(t *testing.T) {
	bloc := renderProjectsIndex(entreesDeTest(79))
	// Mesuré sur les 79 fiches réelles : id + nom + mini ≈ 13 800 caractères (~3 900 tokens). Les
	// entrées de test sont volontairement plus longues que la moyenne. 18 000 laisse de la marge
	// éditoriale sans laisser passer un doublement silencieux.
	require.Less(t, len(bloc), 18000,
		"le bloc part à chaque tour : au-delà, c'est du budget perdu sur chaque message")
}

// Un index vide ne doit pas produire un en-tête orphelin qui ferait croire à l'agent qu'il n'y a
// aucun projet — c'est faux, et il répondrait « il n'a rien fait » avec aplomb.
func TestRenderProjectsIndex_VideNeRendRien(t *testing.T) {
	assert.Empty(t, renderProjectsIndex(nil))
	assert.Empty(t, renderProjectsIndex([]MPFProjectIndexEntry{}))
}

// Une entrée sans mini n'a aucune valeur pour choisir : on l'ignore plutôt que de rendre une ligne
// creuse qui occupe des tokens sans rien apprendre.
func TestRenderProjectsIndex_IgnoreLesEntreesSansMini(t *testing.T) {
	bloc := renderProjectsIndex([]MPFProjectIndexEntry{
		{ID: "vide", Name: "Sans mini", Mini: "   "},
		{ID: "plein", Name: "Avec mini", Mini: "une ligne utile"},
	})
	assert.NotContains(t, bloc, "Sans mini")
	assert.Contains(t, bloc, "Avec mini")
}

// --- Intégration dans le prompt système -------------------------------------

// promptAvecServeur monte un faux maiProFiles et rend le prompt système complet.
// `indexOK=false` simule le déploiement ACTUEL, où /projects/index n'existe pas encore.
func promptAvecServeur(t *testing.T, indexOK bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/projects/index"):
			if !indexOK {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[{"id":"aissia","name":"AISSIA","mini":"Assistant IA en C++17, 7 modules hot-reload","category":"Agent IA","status":"active","stack":["c++17","cmake"]}]`))
		case strings.HasPrefix(r.URL.Path, "/profile"):
			_, _ = w.Write([]byte(`{"name":"Alexi","headline":"Dev","experience_years":8}`))
		case strings.HasPrefix(r.URL.Path, "/stats"):
			_, _ = w.Write([]byte(`{"projects":79,"total_loc":1000,"total_tests":10,"stack":{"go":12}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewMaiProFilesClient()
	client.baseURL = srv.URL
	svc := &ChatService{portfolio: &PortfolioService{client: client}}
	return svc.buildSystemPrompt(context.Background(), ChatCapabilities{})
}

// L'inventaire doit VRAIMENT arriver dans le prompt — c'est tout l'objet de la fonctionnalité.
func TestBuildSystemPrompt_ContientLInventaire(t *testing.T) {
	p := promptAvecServeur(t, true)
	assert.Contains(t, p, "Inventaire des projets")
	assert.Contains(t, p, "aissia", "l'ID doit y être : c'est ce que l'agent repasse à get_project")
	assert.Contains(t, p, "hot-reload")
}

// RÉGRESSION — au moment où ce code est écrit, /projects/index n'est PAS déployé côté maiProFiles.
// Le chat doit fonctionner exactement comme avant : profil et stats présents, inventaire absent,
// aucune erreur. Sans cette garantie, livrer maicivy avant maiProFiles casserait le chat en prod.
func TestBuildSystemPrompt_SansIndexResteFonctionnel(t *testing.T) {
	p := promptAvecServeur(t, false)
	assert.NotContains(t, p, "Inventaire des projets",
		"pas d'en-tête orphelin : l'agent conclurait qu'il n'y a aucun projet")
	assert.Contains(t, p, "Alexi", "le reste du prompt doit être intact")
	assert.Contains(t, p, "Portfolio : 79 projets")
	assert.NotEmpty(t, p)
}

// --- Nom réduit à son identité, le mini porte la description -----------------

// POURQUOI : partage des rôles. Le NOM désigne le projet devant l'utilisateur, le MINI (gardé
// complet) le décrit. 69 des 79 fiches portent un sous-titre après le nom, qui refait le travail du
// mini en moins bien — 3 434 caractères (~980 tokens) sur un bloc renvoyé à chaque tour.
//
// ⚠️ Ne PAS écrire que ces sous-titres « répètent le mini » : vérifié, la redondance textuelle ne
// concerne que 32 fiches (712 caractères). La coupe se justifie par les rôles, pas par la redondance.
func TestNomProjet_CoupeALAccroche(t *testing.T) {
	cas := []struct{ nom, attendu string }{
		{"AISSIA (CELUNA) — Moteur d'assistant personnel IA en C++17", "AISSIA (CELUNA)"},
		{"Aria – Assistant IA desktop", "Aria"},
		{"maicivy - Portfolio agentique", "maicivy"},
		// Pas de séparateur : le nom est déjà nu, on n'y touche pas.
		{"GroveEngine", "GroveEngine"},
		// Un tiret NON espacé fait partie du nom (e-commerce, C-Sharp…) : ne pas le couper.
		{"wechat-homework-bot", "wechat-homework-bot"},
		// Un nom qui ne serait QUE l'accroche ne doit pas devenir vide.
		{" — accroche seule", " — accroche seule"},
	}
	for _, c := range cas {
		assert.Equal(t, c.attendu, nomProjetCourt(c.nom), "nom source: %q", c.nom)
	}
}

// Le rendu complet applique bien la coupe, et laisse le mini intact.
func TestRenderProjectsIndex_NomCourtMiniComplet(t *testing.T) {
	long := "Assistant personnel IA en C++17 — 7 modules hot-reload en pub/sub, pipeline vocal STT/TTS, serveur MCP"
	bloc := renderProjectsIndex([]MPFProjectIndexEntry{{
		ID: "aissia", Name: "AISSIA (CELUNA) — Moteur d'assistant personnel IA en C++17", Mini: long,
	}})
	assert.Equal(t, "aissia | AISSIA (CELUNA) — "+long, bloc)
}

// Les projets qui ont une page publique sont marqués [page] — c'est ce qui permet au Composer de
// lier la page du projet cité sans inventer d'adresse. Sans liste (nil), aucune marque.
func TestRenderProjectsIndex_MarquePage(t *testing.T) {
	entrees := []MPFProjectIndexEntry{
		{ID: "melodicode", Name: "Melodicode", Mini: "Orchestrateur d'agents", Status: "production"},
		{ID: "outil", Name: "Outil", Mini: "Petit outil", Status: "active"},
	}
	bloc := renderProjectsIndexAvecPages(entrees, map[string]bool{"melodicode": true})
	assert.Contains(t, bloc, "melodicode | Melodicode [production] [page] — Orchestrateur d'agents")
	assert.Contains(t, bloc, "outil | Outil — Petit outil")
	assert.NotContains(t, renderProjectsIndex(entrees), "[page]")
}
