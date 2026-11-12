package services

// Vitrine des projets sur le CV (décision du 2026-10-11) : chaque carte affiche une accroche
// (le `mini` de maiProFiles) et une ligne de PREUVE (« en production · 94 k lignes · 1,4 k tests »),
// ses technos sont plafonnées à 5 à l'écran, et la grille est bornée pour tenir en 2 pages.
// Contexte : le seul prospect Malt qui a vraiment échangé demandait des preuves vérifiables ; le
// CV n'affichait que des titres et des murs de pastilles.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

func TestLigneDePreuve_FrEtEn(t *testing.T) {
	stats := MPFStats{LOC: 93586, Tests: 1415}
	require.Equal(t, "en production · 94 k lignes · 1,4 k tests", ligneDePreuve("production", stats, "fr"))
	require.Equal(t, "in production · 94k lines · 1.4k tests", ligneDePreuve("production", stats, "en"))
	require.Equal(t, "已投产 · 94k 行代码 · 1.4k 项测试", ligneDePreuve("production", stats, "zh"), "zh : formulation chinoise, jamais française")
}

func TestLigneDePreuve_SeulProductionEstUnStatutEtZeroNestJamaisAffiche(t *testing.T) {
	// « active » / « wip » n'apportent aucune preuve : rien n'est rendu pour eux.
	require.Equal(t, "333 lignes", ligneDePreuve("active", MPFStats{LOC: 333}, "fr"))
	// Une stat inconnue (0) ne devient pas « 0 tests ».
	require.Equal(t, "en production", ligneDePreuve("production", MPFStats{}, "fr"))
	require.Equal(t, "", ligneDePreuve("wip", MPFStats{}, "fr"))
}

func TestCompteCourt(t *testing.T) {
	cas := []struct {
		n      int
		en     bool
		attend string
	}{
		{999, false, "999"},
		{1000, false, "1 k"},
		{1415, false, "1,4 k"},
		{1415, true, "1.4k"},
		{5423, false, "5,4 k"},
		{124750, false, "125 k"},
		{124750, true, "125k"},
	}
	for _, c := range cas {
		require.Equal(t, c.attend, compteCourt(c.n, c.en), "compteCourt(%d, en=%v)", c.n, c.en)
	}
}

func TestMapProjects_AccrocheEtPreuve(t *testing.T) {
	fr := []MPFProject{{ID: "maicivy", Name: "maicivy — Site", Status: "production",
		Stats: MPFStats{LOC: 93586, Tests: 1415}}}
	out := mapProjects(fr, nil, nil, vitrineProjets{Minis: map[string]string{"maicivy": " Site Go/Next.js en production "}, Lang: "fr"})
	require.Len(t, out, 1)
	require.Equal(t, "Site Go/Next.js en production", out[0].Catchphrase)
	require.Equal(t, "en production · 94 k lignes · 1,4 k tests", out[0].Proof)
}

// fausseSourceIndex : index par langue, ou erreur pour une langue absente.
type fausseSourceIndex map[string][]MPFProjectIndexEntry

func (f fausseSourceIndex) GetProjectsIndex(_ context.Context, lang string) ([]MPFProjectIndexEntry, error) {
	if idx, ok := f[lang]; ok {
		return idx, nil
	}
	return nil, errors.New("indisponible")
}

// Un CV anglais ne doit JAMAIS afficher une accroche française. maiProFiles renvoie la source FR
// tant qu'une traduction n'est pas en cache — mesuré le 2026-10-11 : /projects/index?lang=en
// servait le mini français de maicivy.
func TestMinisParLangue_JamaisDeFrancaisSurUneAutreLangue(t *testing.T) {
	src := fausseSourceIndex{
		"fr": {{ID: "a", Mini: "Accroche A"}, {ID: "b", Mini: "Accroche B"}},
		"en": {{ID: "a", Mini: "Accroche A"}, {ID: "b", Mini: "Hook B"}},
	}
	got := minisParLangue(context.Background(), src)
	require.Equal(t, "Accroche A", got["fr"]["a"])
	require.Equal(t, "Hook B", got["en"]["b"])
	_, present := got["en"]["a"]
	require.False(t, present, "mini EN identique au FR = non traduit → aucune accroche plutôt que du français")
}

func TestMinisParLangue_IndexIndisponibleNeCassePasLeCV(t *testing.T) {
	got := minisParLangue(context.Background(), fausseSourceIndex{})
	require.Empty(t, got["fr"])
	require.Empty(t, got["en"])
}

// Rendu réel du gabarit (sans Chromium) : la carte montre le titre court, l'accroche, la preuve,
// au plus 5 technos — et la couche ATS garde la liste COMPLÈTE.
func TestGabaritCV_CarteProjetCompacteAvecPreuve(t *testing.T) {
	svc := nouveauPDFServiceDeTest(t)
	techs := []string{"go", "fiber", "gorm", "postgresql", "redis", "next.js", "playwright"}
	cv := &AdaptiveCVResponse{
		Theme: *config.GetTheme("ai"),
		Projects: []ScoredProjectResponse{{Score: 0.9, Project: models.Project{
			Title: "maicivy — Site freelance en production", Catchphrase: "Site Go/Next.js en production",
			Proof: "en production · 94 k lignes · 1,4 k tests", Technologies: techs,
		}}},
	}
	html, err := svc.renderCVHTML(cv, "fr", "")
	require.NoError(t, err)

	require.Contains(t, html, `<div class="proj-title">maicivy</div>`, "titre réduit à son identité")
	require.Contains(t, html, `<div class="proj-catch">Site Go/Next.js en production</div>`)
	require.Contains(t, html, `<div class="proj-proof">en production · 94 k lignes · 1,4 k tests</div>`)

	carte := html[strings.Index(html, `<div class="proj-card">`):]
	carte = carte[:strings.Index(carte, `<div class="proj-catch">`)]
	require.Equal(t, 5, strings.Count(carte, `<span class="pill">`), "5 technos max à l'écran")
	require.NotContains(t, carte, "playwright")
	require.Contains(t, html, "playwright", "la liste complète reste dans la couche ATS")
}

func TestGabaritCV_PitchDuThemeDansLaLangueDuCV(t *testing.T) {
	svc := nouveauPDFServiceDeTest(t)
	// Fragments sans caractère échappé par html/template (' → &#39;, + → &#43;).
	theme := *config.GetTheme("ai")

	fr, err := svc.renderCVHTML(&AdaptiveCVResponse{Theme: theme}, "fr", "")
	require.NoError(t, err)
	require.Contains(t, fr, "Je conçois et mets en production des systèmes")
	require.Contains(t, fr, "AI Orchestration Engineer — Technomancer")

	en, err := svc.renderCVHTML(&AdaptiveCVResponse{Theme: theme}, "en", "")
	require.NoError(t, err)
	require.Contains(t, en, "I design and ship AI agent systems")
	require.NotContains(t, en, "Je conçois et mets en production des systèmes", "jamais le pitch français sur un CV anglais")

	// zh : pitch natif — avant le 28/10 il repliait sur l'anglais, en tête du CV chinois.
	zh, err := svc.renderCVHTML(&AdaptiveCVResponse{Theme: theme}, "zh", "")
	require.NoError(t, err)
	require.Contains(t, zh, "我为初创公司和成长型企业设计并交付")
	require.NotContains(t, zh, "I design and ship AI agent systems", "jamais le pitch anglais sur un CV chinois")

	// Le résumé LLM d'un CV taillé reste prioritaire sur le pitch du thème.
	taille, err := svc.renderCVHTML(&AdaptiveCVResponse{Theme: theme, Summary: "Résumé taillé"}, "fr", "")
	require.NoError(t, err)
	require.Contains(t, taille, "Résumé taillé")
	require.NotContains(t, taille, "Je conçois et mets en production des systèmes")
}

// nouveauPDFServiceDeTest : le VRAI gabarit, chargé depuis la racine backend (cwd des tests =
// internal/services).
func nouveauPDFServiceDeTest(t *testing.T) *PDFService {
	t.Helper()
	svc := newPDFServiceFrom("../../templates/cv")
	require.True(t, svc.hasTemplates, "gabarit CV introuvable")
	return svc
}

func TestPitchFor_RepliAnglaisJamaisFrancais(t *testing.T) {
	th := config.CVTheme{Pitch: map[string]string{"fr": "FR", "en": "EN"}}
	require.Equal(t, "FR", th.PitchFor("fr"))
	require.Equal(t, "EN", th.PitchFor("de"))
	require.Equal(t, "", config.CVTheme{}.PitchFor("fr"))
}
