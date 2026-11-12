package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// Verrouille la LANGUE du CV taillé (POST /cv/generate).
//
// CONTEXTE (mesuré sur la prod le 11/11/2026) : un CV demandé en anglais sortait avec un résumé
// et les accroches des trois premières expériences en FRANÇAIS — les seuls textes rédigés par le
// modèle. Le prompt est écrit en français, ses exemples de ton aussi, et seule la langue zh
// recevait une consigne de sortie : en/fr laissaient le modèle choisir, et il suivait le prompt.
// Le reste du CV (contenu maiProFiles, déjà dans la bonne langue) était correct.
//
// Deux verrous, aux deux endroits où ça casse : le prompt (consigne pour CHAQUE langue de PDF) et
// la réponse (une rédaction dans la mauvaise langue n'atteint jamais le CV visible).

// fauxContenuDansLaLangue : contenu maiProFiles minimal, servi DANS la langue demandée — comme le
// vrai provider, qui garde un cache par langue (cf. content_provider.go).
type fauxContenuDansLaLangue struct{}

// expsParLangue : quatre expériences dont l'accroche d'origine est dans la langue de la clé.
var expsParLangue = map[string][]models.Experience{
	"en": {
		{Title: "AI Agent Engineer", Company: "Start-up SaaS B2B", Catchphrase: "A team of AI agents that delivers the platform code"},
		{Title: "AI Infrastructure Developer", Company: "Awen", Catchphrase: "Deployed the infrastructure of an AI API distribution service"},
		{Title: "AI Agent & Automation Engineer", Company: "Self-employed", Catchphrase: "MCP servers and automations for small businesses"},
		{Title: "SEO System Developer", Company: "Cogesco", Catchphrase: "Modernizing an Office/VBA information system"},
	},
	"fr": {
		{Title: "Ingénieur agents IA", Company: "Start-up SaaS B2B", Catchphrase: "Une équipe d'agents IA qui livre le code de la plateforme"},
		{Title: "Développeur infrastructure IA", Company: "Awen", Catchphrase: "Déploiement de l'infrastructure d'un service de distribution d'API IA"},
		{Title: "Ingénieur agents IA et automatisation", Company: "Indépendant", Catchphrase: "Serveurs MCP et automatisations pour les PME"},
		{Title: "Développeur système SEO", Company: "Cogesco", Catchphrase: "Modernisation d'un système d'information Office/VBA"},
	},
}

func (fauxContenuDansLaLangue) GetExperiences(lang string) []models.Experience {
	src := expsParLangue[lang]
	out := make([]models.Experience, len(src))
	copy(out, src)
	return out
}
func (fauxContenuDansLaLangue) GetSkills(string) []models.Skill     { return nil }
func (fauxContenuDansLaLangue) GetProjects(string) []models.Project { return nil }

// redactionModele : ce que le faux modèle écrit pour un CV demandé dans une langue donnée.
// `mauvaises` : rédactions dans l'AUTRE langue (titre, résumé, trois accroches) — les formes
// réellement observées en prod pour en. `bonne` : une accroche dans la bonne langue, qui doit,
// elle, atteindre le CV (une garde qui jetterait tout passerait sinon le test).
type redactionModele struct {
	titre, resume string
	accroches     [3]string
	bonne         string
}

var redactionsDansLaMauvaiseLangue = map[string]redactionModele{
	// CV demandé en anglais, modèle qui répond en français (mesuré en prod le 11/11/2026).
	"en": {
		titre: "Ingénieur des systèmes multi-agents pour la production",
		resume: "Je construis des systèmes multi-agents qui vont en prod. Une équipe d'agents qui livre du code " +
			"avec preuves E2E et mutation testing, des serveurs MCP en série. Le tool use et l'orchestration " +
			"sont pour moi des briques d'architecture, pas des démos.",
		accroches: [3]string{
			"Agents IA qui livrent du code, validés par preuves E2E et mutation testing",
			"Infrastructure de distribution d'API LLM (Claude), Docker, Node.js",
			"Agents LLM, serveurs MCP et automatisations en production pour PME et startups",
		},
		bonne: "Rust microservices that replace an Office/VBA system",
	},
	// CV demandé en français, modèle qui répond en anglais (même panne, sens inverse : les
	// données projet de la source fr sont en anglais, elles tirent le modèle vers l'anglais).
	"fr": {
		titre: "Senior engineer for multi-agent systems in production",
		resume: "I build multi-agent systems that ship to production. A team of agents delivers the code " +
			"with E2E proofs and mutation testing, and the tool use is a building block of the architecture.",
		accroches: [3]string{
			"AI agents that ship code, validated by E2E proofs and mutation testing",
			"Distribution infrastructure for the Claude API, with Docker and Node.js",
			"LLM agents and MCP servers in production for small businesses and startups",
		},
		bonne: "Microservices Rust qui remplacent un système Office/VBA",
	},
}

// fausseReponseModele assemble la réponse du faux modèle : un raisonnement, puis le JSON — la
// forme réelle (cf. parseLLMResponse). Les slugs sont ceux que le service met dans le prompt.
func fausseReponseModele(t *testing.T, lang string) string {
	t.Helper()
	r := redactionsDansLaMauvaiseLangue[lang]
	exps := expsParLangue[lang]
	type exp struct {
		Slug        string `json:"slug"`
		Score       int    `json:"score"`
		Catchphrase string `json:"catchphrase"`
	}
	// Struct et non map : l'ordre des clés doit être celui du schéma demandé au modèle
	// (job_title en tête) — parseLLMResponse ancre le JSON sur le dernier "job_title".
	payload := struct {
		JobTitle    string `json:"job_title"`
		Summary     string `json:"summary"`
		Location    string `json:"location"`
		Experiences []exp  `json:"experiences"`
		Projects    []any  `json:"projects"`
		Skills      []any  `json:"skills"`
	}{
		JobTitle: r.titre,
		Summary:  r.resume,
		Experiences: []exp{
			{toSlug(exps[0].Company + "-" + exps[0].Title), 95, r.accroches[0]},
			{toSlug(exps[1].Company + "-" + exps[1].Title), 90, r.accroches[1]},
			{toSlug(exps[2].Company + "-" + exps[2].Title), 88, r.accroches[2]},
			{toSlug(exps[3].Company + "-" + exps[3].Title), 70, r.bonne},
		},
		Projects: []any{},
		Skills:   []any{},
	}
	js, err := json.Marshal(payload)
	require.NoError(t, err)
	return "<analysis>L'offre cherche un ingénieur agents.</analysis>\n" + string(js)
}

// servicePasserelleFactice : CVGenerationService dont la passerelle est un faux serveur
// renvoyant `texte` — aucun appel réel au modèle.
func servicePasserelleFactice(t *testing.T, texte string) *CVGenerationService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body, _ := json.Marshal(map[string]any{"content": []map[string]string{{"text": texte}}})
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	s := NewCVGenerationService(fauxContenuDansLaLangue{}, srv.URL, "test-key", "modele-test", nil)
	s.httpClient = &http.Client{Timeout: 5 * time.Second}
	return s
}

// textesVisiblesRediges : tout ce que le modèle peut faire apparaître à l'écran dans un CV taillé
// — titre du bandeau, résumé, accroches d'expérience.
func textesVisiblesRediges(cv *AdaptiveCVResponse, lang string) []string {
	out := []string{cv.Theme.NameFor(lang), cv.Summary}
	for _, e := range cv.Experiences {
		out = append(out, e.Catchphrase)
	}
	return out
}

// TestGenerateDynamicCV_RedactionDansLaMauvaiseLangueNAtteintPasLeCV : quelle que soit la langue
// dans laquelle le modèle répond, le CV visible reste dans la langue demandée. Une accroche
// réécrite dans la mauvaise langue cède la place à l'accroche maiProFiles (déjà dans la bonne) ;
// une accroche dans la bonne langue est conservée.
func TestGenerateDynamicCV_RedactionDansLaMauvaiseLangueNAtteintPasLeCV(t *testing.T) {
	for _, lang := range []string{"en", "fr"} {
		t.Run(lang, func(t *testing.T) {
			s := servicePasserelleFactice(t, fausseReponseModele(t, lang))
			cv, err := s.GenerateDynamicCV(context.Background(), "Senior AI Engineer, multi-agent systems", lang)
			require.NoError(t, err)

			r := redactionsDansLaMauvaiseLangue[lang]
			mauvaises := append([]string{r.titre, r.resume}, r.accroches[:]...)
			visibles := strings.Join(textesVisiblesRediges(cv, lang), "\n")
			for _, m := range mauvaises {
				require.NotContains(t, visibles, m,
					"rédaction dans la mauvaise langue visible sur un CV %s", lang)
			}

			// La bonne rédaction passe, et les accroches écartées retombent sur maiProFiles.
			require.Contains(t, visibles, r.bonne, "une rédaction dans la bonne langue doit être gardée")
			parEntreprise := map[string]string{}
			for _, e := range cv.Experiences {
				parEntreprise[e.Company] = e.Catchphrase
			}
			for _, src := range expsParLangue[lang][:3] {
				require.Equal(t, src.Catchphrase, parEntreprise[src.Company],
					"accroche écartée : l'accroche maiProFiles (bonne langue) doit la remplacer")
			}
		})
	}
}

// TestPromptGeneration_ConsigneDeLanguePourChaqueLangueDuPDF : chaque langue de PDF taillé reçoit
// une consigne de sortie, distincte des autres, présente dans le prompt. Avant le 11/11/2026, seule
// zh en avait une — en et fr laissaient le modèle suivre la langue du prompt (le français).
func TestPromptGeneration_ConsigneDeLanguePourChaqueLangueDuPDF(t *testing.T) {
	s := &CVGenerationService{}
	vues := map[string]string{}
	for _, lang := range PDFLanguages {
		c := consigneLangueSortie(lang)
		require.NotEmpty(t, strings.TrimSpace(c), "aucune consigne de langue de sortie pour %s", lang)
		require.Contains(t, s.buildGenerationPrompt("offre", nil, nil, nil, lang), c)
		for autre, vue := range vues {
			require.NotEqual(t, vue, c, "%s et %s reçoivent la même consigne", lang, autre)
		}
		vues[lang] = c
	}
}

// TestRedigeHorsLangue_PitchsDesThemes : la détection sur le corpus RÉEL rédigé à la main dans
// les trois langues (pitchs des thèmes, config/themes.go) — chaque pitch est reconnu dans sa
// langue et écarté dans les deux autres. Aucune phrase choisie pour l'occasion.
func TestRedigeHorsLangue_PitchsDesThemes(t *testing.T) {
	vus := 0
	for id, th := range config.GetAvailableThemes() {
		for langPitch, pitch := range th.Pitch {
			for _, langCV := range PDFLanguages {
				require.Equal(t, langCV != langPitch, redigeHorsLangue(pitch, langCV),
					"thème %s : pitch %s lu pour un CV %s", id, langPitch, langCV)
			}
			vus++
		}
	}
	require.NotZero(t, vus, "aucun pitch de thème : le test ne vérifierait rien")
}

// TestRedigeHorsLangue_SansPreuveRienNEstEcarte : un texte sans mot-outil (titre fait de noms
// propres et de technos, chaîne vide) n'est jamais écarté — la garde ne retire que sur preuve.
func TestRedigeHorsLangue_SansPreuveRienNEstEcarte(t *testing.T) {
	for _, texte := range []string{"", "Senior AI Engineer — Multi-agent systems & MCP", "Python, Go, Postgres, Docker", "Lead Go"} {
		for _, lang := range PDFLanguages {
			require.False(t, redigeHorsLangue(texte, lang), "%q écarté pour %s", texte, lang)
		}
	}
}
