package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// projetsDeTest fabrique n projets identifiés (SourceID), comme les sert le content provider.
func projetsDeTest(n int) []models.Project {
	out := make([]models.Project, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, models.Project{
			SourceID:     fmt.Sprintf("projet_%02d", i),
			Title:        fmt.Sprintf("Projet %d — plateforme interne de centralisation des accès aux API", i),
			Catchphrase:  "Accroche du projet",
			Status:       "active",
			Technologies: []string{"go", "postgresql", "redis"},
		})
	}
	return out
}

// scoringDeTest monte un faux gateway et le service branché dessus.
// handler reçoit le body de la requête et rend le texte de réponse du LLM + son stop_reason.
func scoringDeTest(t *testing.T, handler func(body map[string]interface{}) (string, string)) (*LLMScoringService, *int) {
	t.Helper()
	appels := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels++
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(raw, &body))
		texte, stop := handler(body)
		resp := map[string]interface{}{
			"content":     []map[string]string{{"type": "text", "text": texte}},
			"stop_reason": stop,
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(srv.Close)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	svc := NewLLMScoringService(srv.URL, "cle-de-test", "modele-epingle", redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	require.NotNil(t, svc)
	return svc, &appels
}

func themeDeTest() *config.CVTheme {
	return &config.CVTheme{ID: "ai", Name: "AI Agent Engineer — Technomancer", Description: "agents en production",
		TagWeights: map[string]float64{"go": 1}}
}

// scoresComplets rend un JSON qui note tous les projets, indexé par identifiant.
func scoresComplets(projets []models.Project) string {
	paires := make([]string, 0, len(projets))
	for i, p := range projets {
		paires = append(paires, fmt.Sprintf("%q: %d", p.SourceID, 100-i))
	}
	return "{" + strings.Join(paires, ",") + "}"
}

// Le budget de sortie suit le NOMBRE de projets. Mesuré le 2026-09-07 : un `max_tokens: 512` en
// dur coupait la réponse après 16 scores sur 79 → repli silencieux sur les poids de tags.
func TestScoreProjects_MaxTokensSuitLeNombreDeProjets(t *testing.T) {
	projets := projetsDeTest(79)
	var maxTokens float64
	svc, _ := scoringDeTest(t, func(body map[string]interface{}) (string, string) {
		maxTokens = body["max_tokens"].(float64)
		return scoresComplets(projets), "end_turn"
	})
	_, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.NoError(t, err)
	require.GreaterOrEqual(t, maxTokens, float64(len(projets)*32))
}

// Le modèle est celui de la configuration (épinglé), plus un identifiant en dur. Ni `temperature`
// (refusée par les modèles 5.x : 400) ni `thinking` (bloc vide via la passerelle).
func TestScoreProjects_ModeleEpingleEtDeterministe(t *testing.T) {
	projets := projetsDeTest(5)
	var body map[string]interface{}
	svc, _ := scoringDeTest(t, func(b map[string]interface{}) (string, string) {
		body = b
		return scoresComplets(projets), "end_turn"
	})
	_, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.NoError(t, err)
	require.Equal(t, "modele-epingle", body["model"])
	_, avecTemperature := body["temperature"]
	require.False(t, avecTemperature, "temperature refusée par les modèles 5.x (mesuré : 400)")
	_, avecThinking := body["thinking"]
	require.False(t, avecThinking, "pas de thinking : budget_tokens rend un bloc vide via la passerelle")
}

// Le modèle reçoit de quoi distinguer un outil d'un système : identifiant, accroche, statut,
// ampleur. Mesuré : sans eux, il classait par mots-clés du titre.
func TestScoreProjects_LePromptPorteLesSignauxDeFond(t *testing.T) {
	projets := []models.Project{{SourceID: "maicivy", Title: "maicivy — Site freelance", Catchphrase: "Site Go/Next.js en production",
		Status: "production", LOC: 93586, TestCount: 1415, Category: "Web", Technologies: []string{"go"}}}
	var prompt string
	svc, _ := scoringDeTest(t, func(b map[string]interface{}) (string, string) {
		prompt = b["messages"].([]interface{})[0].(map[string]interface{})["content"].(string)
		return `{"maicivy": 90}`, "end_turn"
	})
	_, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.NoError(t, err)
	require.Contains(t, prompt, "- maicivy | maicivy | Site Go/Next.js en production | Web | production | 93586 | 1415 | go")
	require.Contains(t, prompt, "AI Agent Engineer — Technomancer")
	require.Contains(t, prompt, "Juge le FOND, pas les mots-clés")
}

// Les scores sont indexés par IDENTIFIANT : un id inventé par le modèle est ignoré, un titre
// réécrit ne perd plus son score.
func TestScoreProjects_IndexeParIdentifiant(t *testing.T) {
	projets := projetsDeTest(3)
	svc, _ := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return `{"projet_00": 90, "projet_01": 40, "projet_02": 10, "invente": 99}`, "end_turn"
	})
	scores, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.NoError(t, err)
	require.Equal(t, map[string]int{"projet_00": 90, "projet_01": 40, "projet_02": 10}, scores)
}

// Une réponse coupée par le budget ne passe jamais pour un succès.
func TestScoreProjects_TroncatureEstUneErreur(t *testing.T) {
	projets := projetsDeTest(79)
	svc, _ := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return `{"projet_00": 80, "projet_01": 75,`, "max_tokens"
	})
	_, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "tronqu")
}

// Un JSON valide qui ne note que quelques projets n'est pas un classement.
func TestScoreProjects_CouvertureIncompleteEstUneErreur(t *testing.T) {
	projets := projetsDeTest(79)
	svc, _ := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return `{"projet_00": 90, "projet_01": 80, "projet_02": 70}`, "end_turn"
	})
	_, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.Error(t, err)
	require.Contains(t, err.Error(), "3/79")
}

// Quelques oublis : les manquants tombent BAS, ils ne s'insèrent pas au milieu.
func TestScoreProjects_CouvertureQuasiCompleteEstAcceptee(t *testing.T) {
	projets := projetsDeTest(20)
	svc, _ := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return scoresComplets(projets[:19]), "end_turn"
	})
	scores, err := svc.ScoreProjectsForTheme(context.Background(), projets, themeDeTest())
	require.NoError(t, err)
	require.Len(t, scores, 20)
	require.Equal(t, scoreProjetNonNote, scores["projet_19"])
}

// Le cache sert tant que rien ne change, et se renouvelle de lui-même quand le CATALOGUE change
// (ex : un projet passe en « retired ») — sans attendre 6 h.
func TestScoreProjects_CacheInvalideParLeCatalogue(t *testing.T) {
	projets := projetsDeTest(4)
	svc, appels := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return scoresComplets(projets), "end_turn"
	})
	ctx := context.Background()
	_, err := svc.ScoreProjectsForTheme(ctx, projets, themeDeTest())
	require.NoError(t, err)
	_, err = svc.ScoreProjectsForTheme(ctx, projets, themeDeTest())
	require.NoError(t, err)
	require.Equal(t, 1, *appels, "même catalogue → cache")

	modifie := append([]models.Project(nil), projets...)
	modifie[0].Status = "retired"
	_, err = svc.ScoreProjectsForTheme(ctx, modifie, themeDeTest())
	require.NoError(t, err)
	require.Equal(t, 2, *appels, "catalogue changé → nouveau classement")
}

func TestScoringCacheKey_DependDuThemeEtDuModele(t *testing.T) {
	p := projetsDeTest(2)
	a := &config.CVTheme{ID: "ai", Name: "AI Engineer", Description: "x"}
	b := &config.CVTheme{ID: "ai", Name: "AI Agent Engineer — Technomancer", Description: "x"}
	require.NotEqual(t, scoringCacheKey(a, "m", p), scoringCacheKey(b, "m", p))
	require.NotEqual(t, scoringCacheKey(a, "m1", p), scoringCacheKey(a, "m2", p))
	require.Equal(t, scoringCacheKey(a, "m", p), scoringCacheKey(a, "m", p))
	require.True(t, strings.HasPrefix(scoringCacheKey(a, "m", p), "llm:project-scores:ai:"))
}
