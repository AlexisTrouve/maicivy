package services

import (
	"testing"
	"time"

	"maicivy/internal/models"
)

// Régression du 28/10/2026 : le CV rangeait les expériences par score de thème. Sur le PDF de prod
// (thème ai), CoconSystem (02/2026 → 03/2026) passait AVANT la mission SaaS en cours (04/2026 →
// présent), qui n'arrivait qu'en 3e position — alors que l'accroche du CV ne parle que d'elle.
// Un lecteur de CV lit une chronologie : l'ordre doit être chronologique inverse.

func mois(a, m int) time.Time { return time.Date(a, time.Month(m), 1, 0, 0, 0, 0, time.UTC) }

func exp(titre string, debut time.Time, fin *time.Time, score float64) ScoredExperienceResponse {
	return ScoredExperienceResponse{
		Experience: models.Experience{Title: titre, StartDate: debut, EndDate: fin},
		Score:      score,
	}
}

func titres(e []ScoredExperienceResponse) []string {
	out := make([]string, len(e))
	for i := range e {
		out[i] = e[i].Title
	}
	return out
}

func TestOrdreExperiencesCV_EnCoursPuisFinDecroissante(t *testing.T) {
	finCocon := mois(2026, 3)
	finTaglabs := mois(2020, 10)
	// Scores choisis pour que l'ancien tri (score DESC) donne l'ordre FAUX observé en prod.
	e := []ScoredExperienceResponse{
		exp("freelance", mois(2024, 9), nil, 0.9),
		exp("cocon", mois(2026, 2), &finCocon, 0.8),
		exp("saas", mois(2026, 4), nil, 0.7),
		exp("awen", mois(2025, 12), nil, 0.6),
		exp("taglabs", mois(2018, 9), &finTaglabs, 0.95),
	}

	ordonnerExperiencesCV(e)

	attendu := []string{"saas", "awen", "freelance", "cocon", "taglabs"}
	got := titres(e)
	for i := range attendu {
		if got[i] != attendu[i] {
			t.Fatalf("ordre = %v, attendu %v", got, attendu)
		}
	}
}

func TestOrdreExperiencesCV_MemeFinDepartageParDebut(t *testing.T) {
	fin := mois(2025, 6)
	e := []ScoredExperienceResponse{
		exp("ancienne", mois(2021, 1), &fin, 0.9),
		exp("recente", mois(2024, 1), &fin, 0.1),
	}
	ordonnerExperiencesCV(e)
	if got := titres(e); got[0] != "recente" {
		t.Fatalf("ordre = %v, la plus récemment commencée d'abord", got)
	}
}
