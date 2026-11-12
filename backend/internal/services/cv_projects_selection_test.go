package services

import (
	"fmt"
	"strings"
	"testing"

	"maicivy/internal/models"

	"github.com/stretchr/testify/require"
)

// projetsScores fabrique n projets déjà triés par score décroissant (invariant en amont).
func projetsScores(scores ...float64) []ScoredProjectResponse {
	out := make([]ScoredProjectResponse, 0, len(scores))
	for i, sc := range scores {
		out = append(out, ScoredProjectResponse{
			Project: models.Project{
				Title:        fmt.Sprintf("Projet %02d", i),
				Technologies: []string{"go", "postgresql"},
			},
			Score: sc,
		})
	}
	return out
}

func scoresDegressifs(n int, haut, bas float64) []float64 {
	out := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, haut-(haut-bas)*float64(i)/float64(n-1))
	}
	return out
}

// TestSelectionProjets_CoupeLaQueue : le CV déroulait le catalogue entier — 79 projets, dont 73
// en grille compacte sur 4 pages d'un PDF de 6. Or les projets sont triés par score décroissant :
// cette queue est, par construction, ce qui correspond le MOINS à l'offre.
func TestSelectionProjets_CoupeLaQueue(t *testing.T) {
	projets := projetsScores(scoresDegressifs(79, 0.95, 0.05)...)

	detailles, compacts, ecartes := selectionnerProjets(projets)

	require.Len(t, detailles, projetsDetailles)
	require.LessOrEqual(t, len(compacts), compactsMax, "la grille compacte doit rester bornée")
	require.Equal(t, len(projets), len(detailles)+len(compacts)+len(ecartes), "aucun projet perdu")
	for _, p := range compacts {
		require.GreaterOrEqual(t, p.Score, seuilProjetCompact,
			"un projet sous le seuil ne doit pas figurer dans la grille")
	}
	require.Equal(t, "Projet 00", detailles[0].Title, "l'ordre par score doit être préservé")
}

// TestSelectionProjets_ScoringDegenereNeVidePasLeCV : si le scoring ne discrimine plus (tous les
// projets au même score — l'état réel mesuré en prod le 2026-09-07, 76 sur 79 à 0.50), le seuil
// ne veut plus rien dire. La règle a une précondition : hors de son domaine, elle ne doit pas
// décider seule. Le plafond, lui, s'applique toujours.
func TestSelectionProjets_ScoringDegenereNeVidePasLeCV(t *testing.T) {
	scores := make([]float64, 79)
	for i := range scores {
		scores[i] = 0.50
	}
	_, compacts, _ := selectionnerProjets(projetsScores(scores...))

	require.Equal(t, compactsMax, len(compacts), "tous au-dessus du seuil → le plafond tranche")
}

// TestSelectionProjets_ScoringBasGardeUnPlancher : si le modèle n'a noté personne (tous au score
// plancher), le seuil couperait TOUT. Un CV sans aucun projet est pire que trop de projets.
func TestSelectionProjets_ScoringBasGardeUnPlancher(t *testing.T) {
	scores := make([]float64, 40)
	for i := range scores {
		scores[i] = float64(scoreProjetNonNote) / 100.0
	}
	_, compacts, _ := selectionnerProjets(projetsScores(scores...))

	require.Equal(t, compactsMin, len(compacts), "plancher : le CV garde une queue minimale")
}

// TestSelectionProjets_CatalogueRikiki : moins de projets que de cartes détaillées.
func TestSelectionProjets_CatalogueRikiki(t *testing.T) {
	detailles, compacts, ecartes := selectionnerProjets(projetsScores(0.9, 0.8, 0.7))

	require.Len(t, detailles, 3)
	require.Empty(t, compacts)
	require.Empty(t, ecartes)
}

// TestStripATS_TechnosEcartesDansLeBudget : couper des projets retire leurs mots-clés du PDF.
// On récupère les technos ORPHELINES (absentes du reste du strip) — mais sous un budget dur.
// Mesuré le 2026-09-07 : au-delà d'environ 900 caractères, Chromium émet bien le texte du strip
// mais en LAISSANT TOMBER des glyphes ("P j tT i O ( t l di d k )"). Un strip trop long ne donne
// donc pas plus de mots-clés à l'ATS : il lui donne de la bouillie.
func TestStripATS_TechnosEcartesDansLeBudget(t *testing.T) {
	ecartes := make([]ScoredProjectResponse, 0, 60)
	for i := 0; i < 60; i++ {
		ecartes = append(ecartes, ScoredProjectResponse{Project: models.Project{
			Title:        fmt.Sprintf("Ecarte %02d", i),
			Technologies: []string{fmt.Sprintf("techno-orpheline-%02d", i), "go"},
		}})
	}
	base := buildStealthText([]string{"Go"}, "fr")

	strip := enrichirStripAvecEcartes(base, ecartes)

	require.LessOrEqual(t, len(strip), budgetStripATS, "le strip doit tenir dans son budget")
	require.Contains(t, strip, base, "le strip d'origine ne doit pas être amputé")
	require.Contains(t, strip, "techno-orpheline-00", "au moins une techno orpheline doit passer")
	require.Equal(t, 1, strings.Count(strip, "techno-orpheline-00"), "pas de doublon")
}

// TestStripATS_SansEcartesNeChangeRien : un catalogue entièrement affiché ne doit rien ajouter.
func TestStripATS_SansEcartesNeChangeRien(t *testing.T) {
	base := buildStealthText([]string{"Go"}, "fr")
	require.Equal(t, base, enrichirStripAvecEcartes(base, nil))
}

// TestStripATS_StripVideResteVide : GenerateCVPDF n'injecte aucun strip. Ne pas en fabriquer un.
func TestStripATS_StripVideResteVide(t *testing.T) {
	ecartes := projetsScores(0.1, 0.1)
	require.Equal(t, "", enrichirStripAvecEcartes("", ecartes))
}
