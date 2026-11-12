package services

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : verrouille le périmètre des tools de composition (set_draft / set_rate) exposés à Claude.
//
// POURQUOI : /chat/stream est PARTAGÉ entre le chat public (visiteurs) et l'agent admin — seul le
// modèle change aujourd'hui selon is_owner (cf. api/chat.go). Les tools qui mutent le draft d'un
// message client ne doivent JAMAIS apparaître dans la liste envoyée à l'API pour un visiteur : un
// visiteur qui pourrait les appeler écrirait dans l'artefact d'une session admin. Le gate est une
// capability explicite (ChatCapabilities) et non une déduction depuis le nom du modèle, pour que le
// compilateur force chaque appelant à se prononcer.
//
// COMMENT : on construit la liste de tools dans les deux modes et on vérifie la présence/absence par
// nom. 1. extraire les noms via le champ OfTool de l'union anthropic, 2. mode visiteur (zero value)
// → aucun tool de composition, 3. mode compose → les deux présents, 4. les tools de données
// maiProFiles restent disponibles dans les deux modes (l'agent en a besoin pour parler du CV).

// toolNames extrait les noms de la liste d'unions renvoyée par buildTools.
func toolNames(t *testing.T, s *ChatService, caps ChatCapabilities) []string {
	t.Helper()
	names := []string{}
	for _, u := range s.buildTools(caps) {
		require.NotNil(t, u.OfTool, "tool union sans OfTool — forme inattendue")
		names = append(names, u.OfTool.Name)
	}
	return names
}

// Un visiteur (capabilities à zéro) ne doit voir aucun tool de composition.
func TestBuildTools_VisitorHasNoComposeTools(t *testing.T) {
	s := &ChatService{}

	names := toolNames(t, s, ChatCapabilities{})

	assert.NotContains(t, names, "set_draft",
		"set_draft exposé à un visiteur : il pourrait écrire dans le draft d'une session admin")
	assert.NotContains(t, names, "set_rate",
		"set_rate exposé à un visiteur")
}

// En mode compose (owner + écran /admin/compose), les deux tools doivent être là.
func TestBuildTools_ComposeExposesDraftTools(t *testing.T) {
	s := &ChatService{}

	names := toolNames(t, s, ChatCapabilities{Compose: true})

	assert.Contains(t, names, "set_draft")
	assert.Contains(t, names, "set_rate")
}

// L'ancrage tarifaire doit atteindre le modèle, sinon il propose un montant inventé — un nombre que
// l'opérateur ne peut ni juger ni défendre face au client. Les deux bornes (tarif pratiqué, cible
// visée) doivent figurer dans la description du tool, seul endroit que le modèle lit avant d'appeler.
func TestBuildTools_AncrageTarifaireDansLaDescription(t *testing.T) {
	s := &ChatService{}

	var setRate *anthropic.ToolParam
	for _, u := range s.buildTools(ChatCapabilities{Compose: true, RateCurrent: 200, RateTarget: 300}) {
		if u.OfTool != nil && u.OfTool.Name == "set_rate" {
			setRate = u.OfTool
		}
	}
	require.NotNil(t, setRate, "set_rate absent du mode compose")

	desc := setRate.Description.Value
	assert.Contains(t, desc, "200", "le tarif pratiqué n'atteint pas le modèle")
	assert.Contains(t, desc, "300", "la cible visée n'atteint pas le modèle")
}

// Sans ancrage configuré, la description ne doit pas inventer de fourchette : mieux vaut que le modèle
// demande à l'opérateur que d'afficher un chiffre sorti de nulle part avec une fausse caution.
func TestBuildTools_SansAncrageAucuneFourchette(t *testing.T) {
	s := &ChatService{}

	var setRate *anthropic.ToolParam
	for _, u := range s.buildTools(ChatCapabilities{Compose: true}) {
		if u.OfTool != nil && u.OfTool.Name == "set_rate" {
			setRate = u.OfTool
		}
	}
	require.NotNil(t, setRate)

	assert.NotContains(t, setRate.Description.Value, "fourchette")
}

// --- Dispatch des tools de composition (executeTool) ---
//
// QUOI : vérifie que set_draft / set_rate sont traités AVANT la validation de langue et qu'ils
// renvoient un accusé, pas le contenu.
// POURQUOI : ils ne reçoivent pas de paramètre `language` (ils ne servent aucun contenu localisé) ;
// s'ils tombaient dans la validation commune, tout appel échouerait avec une erreur de langue. Et
// l'accusé (plutôt que le draft renvoyé au LLM) évite de doubler le coût en contexte du message que
// le modèle vient lui-même d'écrire.

func TestExecuteTool_SetDraft_SansLangue(t *testing.T) {
	s := &ChatService{}

	// Aucun champ "language" : c'est précisément le cas qui casserait si le dispatch passait après
	// la validation commune.
	res, err := s.executeTool("set_draft", map[string]interface{}{"message": "Bonjour,\n\nAlexis"}, ChatCapabilities{})

	require.NoError(t, err)
	m, ok := res.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["ok"])
	assert.NotContains(t, m, "message", "le draft ne doit pas être renvoyé au LLM (coût contexte)")
}

func TestExecuteTool_SetDraft_RejetteVide(t *testing.T) {
	s := &ChatService{}

	_, err := s.executeTool("set_draft", map[string]interface{}{"message": "   \n  "}, ChatCapabilities{})

	require.Error(t, err, "un draft vide effacerait le message sans que l'opérateur puisse s'en rendre compte")
}

// Le comptage est en RUNES, pas en octets : un message français est plein d'accents, un compteur en
// octets afficherait une longueur fausse à l'opérateur.
func TestExecuteTool_SetDraft_CompteEnRunes(t *testing.T) {
	s := &ChatService{}

	res, err := s.executeTool("set_draft", map[string]interface{}{"message": "éàü"}, ChatCapabilities{})

	require.NoError(t, err)
	assert.Equal(t, 3, res.(map[string]interface{})["chars"])
}

func TestExecuteTool_SetRate_BornesEtRetour(t *testing.T) {
	s := &ChatService{}

	for _, tc := range []struct {
		label   string
		amount  float64
		wantErr bool
	}{
		{"sous la borne basse", 40, true},
		{"borne basse", 50, false},
		{"nominal", 300, false},
		{"borne haute", 5000, false},
		{"au-dessus de la borne haute", 5001, true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			// amount arrive en float64 : c'est ce que produit le décodage JSON de l'input du tool.
			res, err := s.executeTool("set_rate", map[string]interface{}{
				"amount": tc.amount, "rationale": "stack agentique + prod",
			}, ChatCapabilities{})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			m := res.(map[string]interface{})
			assert.Equal(t, int(tc.amount), m["amount"])
			assert.Equal(t, "stack agentique + prod", m["rationale"])
		})
	}
}

// Les tools de données maiProFiles restent disponibles dans les deux modes : l'agent compose a besoin
// du CV pour rédiger, le chat public en a besoin pour répondre. Le gate ne doit toucher QUE la
// composition.
func TestBuildTools_DataToolsAvailableInBothModes(t *testing.T) {
	s := &ChatService{}

	for _, tc := range []struct {
		label string
		caps  ChatCapabilities
	}{
		{"visiteur", ChatCapabilities{}},
		{"compose", ChatCapabilities{Compose: true}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			names := toolNames(t, s, tc.caps)
			for _, expected := range []string{"get_project", "list_skills", "get_experience"} {
				assert.Contains(t, names, expected)
			}
		})
	}
}

// QUOI : plancher DUR du TJM — set_rate refuse tout montant sous ComposeRateFloor, pour l'owner comme
// pour un prospecteur, et l'erreur (renvoyée au modèle) nomme le plancher.
// POURQUOI : l'ancrage du prompt oriente sans contraindre ; un prospecteur qui négocie librement ne
// doit jamais pouvoir engager Alexis sous 260 €/jour, quel que soit ce que le modèle décide.
func TestExecuteTool_SetRate_PlancherDur(t *testing.T) {
	s := &ChatService{}
	for _, tc := range []struct {
		label string
		caps  ChatCapabilities
	}{
		{"owner", ChatCapabilities{Compose: true, RateFloor: 260}},
		{"prospecteur", ChatCapabilities{Compose: true, RateFloor: 260, Prospect: &ComposeProspect{Name: "Marc", Ref: "abc123"}}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			_, err := s.executeTool("set_rate", map[string]interface{}{"amount": float64(259), "rationale": "r"}, tc.caps)
			require.Error(t, err, "259 doit être refusé")
			assert.Contains(t, err.Error(), "260")

			res, err := s.executeTool("set_rate", map[string]interface{}{"amount": float64(260), "rationale": "r"}, tc.caps)
			require.NoError(t, err, "le plancher lui-même est accepté")
			assert.Equal(t, 260, res.(map[string]interface{})["amount"])
		})
	}
}

// Le plancher doit aussi être ANNONCÉ au modèle (description du tool) : sinon il le découvre par
// l'erreur, après avoir déjà écrit un message autour d'un tarif refusé.
func TestBuildTools_PlancherAnnonce(t *testing.T) {
	s := &ChatService{}
	for _, u := range s.buildTools(ChatCapabilities{Compose: true, RateCurrent: 200, RateTarget: 300, RateFloor: 260}) {
		if u.OfTool != nil && u.OfTool.Name == "set_rate" {
			desc := u.OfTool.Description.Value
			assert.Contains(t, desc, "ne descends jamais sous 260", "le bas de fourchette doit être relevé au plancher")
			assert.Contains(t, desc, "PLANCHER DUR")
			return
		}
	}
	t.Fatal("set_rate absent")
}

// Le prompt prospecteur : écrit au nom du prospecteur, présente Alexis, ne parle pas de Malt, et
// impose le ?ref= sur les liens maicivy.
func TestProspectComposeGuidance(t *testing.T) {
	g := prospectComposeGuidance(ComposeProspect{Name: "Marc Dupont", Ref: "k7q2zx"})
	assert.Contains(t, g, "Marc Dupont")
	assert.Contains(t, g, "?ref=k7q2zx")
	assert.Contains(t, g, "N'écris jamais « je suis Alexis »")
	assert.NotContains(t, strings.ToLower(strings.ReplaceAll(g, "Ne mentionne JAMAIS Malt", "")), "malt")
	assert.Contains(t, rateFloorGuidance(260), "260")
	assert.Empty(t, rateFloorGuidance(0))
}

// « Même composer pour tout le monde » (Alexi, 2026-10-16) : le prompt prospecteur suit la même
// stratégie que celui d'Alexis — 2 ou 3 projets avec leur matière, 3 à 5 liens de référence (tous
// avec ?ref=), une ouverture et non une injonction, 250 à 350 mots.
func TestProspectComposeGuidance_MemeStrategieQueLOwner(t *testing.T) {
	g := prospectComposeGuidance(ComposeProspect{Name: "Marc Dupont", Ref: "k7q2zx"})
	assert.Contains(t, g, "2 OU 3 PROJETS")
	assert.Contains(t, g, "3 À 5 LIENS DE RÉFÉRENCE")
	assert.Contains(t, g, "https://maicivy.etheryale.com/fr/gitstats?ref=k7q2zx")
	assert.Contains(t, g, "https://maicivy.etheryale.com/fr/portfolio?ref=k7q2zx")
	assert.Contains(t, g, "250 à 350 mots")
	assert.Contains(t, g, "OUVERTURE, PAS UNE INJONCTION")
	assert.NotContains(t, g, "Un ou deux liens au plus")
	assert.NotContains(t, g, "UN seul appel à l'action")
}
