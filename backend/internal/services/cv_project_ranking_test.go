package services

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// Projets réels du catalogue (statut, ampleur), tels que mesurés le 2026-10-11.
var (
	pMaicivy   = models.Project{SourceID: "maicivy", Status: "production", LOC: 93586, TestCount: 1415, Activity: 0.8}
	pClaudeDuo = models.Project{SourceID: "claudeduo_messaging", Status: "retired", LOC: 2000}
	pRedditMCP = models.Project{SourceID: "reddit_mcp", Status: "production", LOC: 400}
	pGrove     = models.Project{SourceID: "grove_engine", Status: "active", LOC: 124750, TestCount: 1055, Activity: 0.7}
)

func TestPreuveProjet_ProductionEtAmpleur(t *testing.T) {
	require.InDelta(t, 0, preuveProjet(models.Project{}), 1e-9, "rien de connu → aucune preuve")
	require.InDelta(t, 0.5, preuveProjet(models.Project{Status: "production"}), 1e-9)
	require.Greater(t, preuveProjet(pMaicivy), 0.9, "en prod, 94 k lignes, 1,4 k tests")
	require.Less(t, preuveProjet(pRedditMCP), preuveProjet(pMaicivy))
}

func TestEchelleLog_Saturee(t *testing.T) {
	require.Equal(t, 0.0, echelleLog(500, 1000, 100000))
	require.Equal(t, 1.0, echelleLog(500000, 1000, 100000))
	require.InDelta(t, 0.5, echelleLog(10000, 1000, 100000), 1e-9, "log : 10 k est à mi-chemin entre 1 k et 100 k")
}

// La pertinence décide : un gros projet hors sujet ne passe JAMAIS devant un petit projet
// pertinent, quelle que soit sa preuve.
func TestScoreFinal_LaPertinenceDomine(t *testing.T) {
	gros := models.Project{Status: "production", LOC: 200000, TestCount: 5000, Activity: 1}
	petit := models.Project{}
	require.Greater(t, scoreFinalProjet(0.8, petit), scoreFinalProjet(0.3, gros))
}

// À pertinence égale, la preuve et l'activité départagent.
func TestScoreFinal_APertinenceEgaleLaPreuveDepartage(t *testing.T) {
	require.Greater(t, scoreFinalProjet(0.85, pMaicivy), scoreFinalProjet(0.85, pRedditMCP))
}

// Un projet retiré ne prend plus la place d'un projet vivant. Mesuré : ClaudeDuo (retiré) sortait
// 1er sur le thème agents.
func TestScoreFinal_RetireRecule(t *testing.T) {
	require.Less(t, scoreFinalProjet(0.95, pClaudeDuo), scoreFinalProjet(0.6, pMaicivy))
	require.Less(t, scoreFinalProjet(0.95, models.Project{Status: "Archived"}), 0.3, "insensible à la casse")
}

// Scénario reconstitué : avec des pertinences proches, l'ancien classement mettait le petit
// serveur MCP et ClaudeDuo devant maicivy et GroveEngine. Le nouveau les remet dans l'ordre.
func TestScoreFinal_ScenarioReel(t *testing.T) {
	sMaicivy := scoreFinalProjet(0.86, pMaicivy)
	sGrove := scoreFinalProjet(0.80, pGrove)
	sReddit := scoreFinalProjet(0.87, pRedditMCP)
	sDuo := scoreFinalProjet(0.95, pClaudeDuo)
	require.Greater(t, sMaicivy, sReddit)
	require.Greater(t, sGrove, sReddit)
	require.Greater(t, sReddit, sDuo)
}

func TestPertinencesDeRepli_Normalisees(t *testing.T) {
	got := pertinencesDeRepli(map[string]float64{"a": 0.3, "b": 0.15, "c": 0})
	require.InDelta(t, 1.0, got["a"], 1e-9, "le meilleur projet vaut 1 : sinon les scores de tags (≤ 0,3) écrasent tout")
	require.InDelta(t, 0.5, got["b"], 1e-9)
	require.InDelta(t, 0.0, got["c"], 1e-9)
}

// fauxContenu : content provider minimal pour CVService.
type fauxContenu struct{ projets []models.Project }

func (f fauxContenu) GetExperiences(string) []models.Experience { return nil }
func (f fauxContenu) GetSkills(string) []models.Skill           { return nil }
func (f fauxContenu) GetProjects(string) []models.Project       { return f.projets }

// Bout en bout dans CVService : pertinence LLM (faux gateway) × preuve → ordre final du CV.
func TestGetAdaptiveCV_ClasseParPertinenceEtPreuve(t *testing.T) {
	projets := []models.Project{
		{SourceID: "reddit_mcp", Title: "Reddit MCP Server", Status: "production", LOC: 400},
		{SourceID: "claudeduo_messaging", Title: "ClaudeDuo", Status: "retired", LOC: 2000},
		{SourceID: "maicivy", Title: "maicivy", Status: "production", LOC: 93586, TestCount: 1415},
	}
	llm, _ := scoringDeTest(t, func(map[string]interface{}) (string, string) {
		return `{"reddit_mcp": 87, "claudeduo_messaging": 95, "maicivy": 86}`, "end_turn"
	})
	mr := miniredis.RunT(t)
	svc := NewCVService(fauxContenu{projets}, redis.NewClient(&redis.Options{Addr: mr.Addr()}), llm)

	cv, err := svc.GetAdaptiveCV(context.Background(), "ai", "fr")
	require.NoError(t, err)
	ordre := []string{}
	for _, p := range cv.Projects {
		ordre = append(ordre, p.SourceID)
	}
	require.Equal(t, []string{"maicivy", "reddit_mcp", "claudeduo_messaging"}, ordre)
}

// LLM indisponible : le CV sort quand même, classé par les poids de tags normalisés × preuve.
func TestGetAdaptiveCV_RepliSansLLM(t *testing.T) {
	projets := []models.Project{
		{SourceID: "a", Title: "A", Technologies: []string{"claude api", "mcp"}, Status: "production", LOC: 50000},
		{SourceID: "b", Title: "B", Technologies: []string{"excel"}},
	}
	mr := miniredis.RunT(t)
	svc := NewCVService(fauxContenu{projets}, redis.NewClient(&redis.Options{Addr: mr.Addr()}), nil)
	cv, err := svc.GetAdaptiveCV(context.Background(), "ai", "fr")
	require.NoError(t, err)
	require.Equal(t, "a", cv.Projects[0].SourceID)
	require.Greater(t, cv.Projects[0].Score, 0.5)
	_ = config.GetTheme
}

// Régression (mesurée en prod le 09/11/2026) : le CV est mis en cache Redis avec son thème, mais
// NameI18n porte `json:"-"` et disparaît au retour du cache. Le 1er PDF chinois portait le bon
// titre, les suivants (pendant 5 min) retombaient sur l'intitulé anglais. Le thème doit venir de
// la config à chaque appel, cache ou non.
func TestGetAdaptiveCV_ThemeIntactAuRetourDuCache(t *testing.T) {
	projets := []models.Project{{SourceID: "a", Title: "A", Technologies: []string{"mcp"}}}
	mr := miniredis.RunT(t)
	svc := NewCVService(fauxContenu{projets}, redis.NewClient(&redis.Options{Addr: mr.Addr()}), nil)
	attendu := config.GetTheme("ai").NameFor("zh")
	for i := 0; i < 2; i++ { // 1er appel = cache vide, 2e = cache Redis
		cv, err := svc.GetAdaptiveCV(context.Background(), "ai", "zh")
		require.NoError(t, err)
		require.Equal(t, attendu, cv.Theme.NameFor("zh"), "appel %d", i+1)
	}
}
