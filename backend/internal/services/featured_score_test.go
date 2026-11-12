// backend/internal/services/featured_score_test.go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// makeDays construit un slice de `count` jours de commit distincts, triés croissant, dont le plus
// ancien est now-firstAgo et le plus récent now-lastAgo. featuredScore ne lit que [0], [len-1] et
// len, donc les jours intermédiaires servent juste à atteindre le bon `count`.
func makeDays(now time.Time, firstAgo, lastAgo, count int) []string {
	const fmtDay = "2006-01-02"
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []string{now.AddDate(0, 0, -lastAgo).Format(fmtDay)}
	}
	out := make([]string, 0, count)
	out = append(out, now.AddDate(0, 0, -firstAgo).Format(fmtDay)) // plus ancien
	// fillers : on remonte depuis le plus récent, count-2 jours consécutifs (sans toucher les bornes)
	for i := 1; i <= count-2; i++ {
		out = append(out, now.AddDate(0, 0, -lastAgo-i).Format(fmtDay))
	}
	out = append(out, now.AddDate(0, 0, -lastAgo).Format(fmtDay)) // plus récent
	// tri croissant pour respecter le contrat (CommitDays est trié)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestFeaturedScore(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	// Score de référence "calibre-phare" : repère pour valider que la formule sépare bien les
	// vrais phares des dormants/maigres. (La sélection réelle est un top-N, pas un seuil — cf.
	// autoFeaturedSet ; ce test verrouille les VALEURS du score, pas le mécanisme de choix.)
	const flagshipScore = 0.35

	cases := []struct {
		name        string
		days        []string
		flagshipish bool // score >= flagshipScore ?
		approx      float64
	}{
		{"aucun commit", nil, false, 0},
		// Vieux projet ABANDONNÉ : âgé + régulier autrefois, mais dernier commit 120j → récence=0 → écrasé.
		{"vieux abandonné", makeDays(now, 200, 120, 25), false, 0},
		// Vieux repo mais SEULEMENT 2 jours actifs : l'âge ne rachète pas la régularité nulle.
		{"vieux non régulier", makeDays(now, 150, 0, 2), false, 0.056},
		// "mid" récent-ish mais peu régulier (type video→mp3) : sous le seuil.
		{"mid sparse", makeDays(now, 203, 32, 12), false, 0.258},
		// Phare : ancien, très régulier, actif maintenant.
		{"phare établi régulier actif", makeDays(now, 182, 0, 35), true, 1.0},
		// Pin-like (type confluent) : ancien, 24 jours actifs, commit il y a 3j.
		{"établi régulier", makeDays(now, 182, 3, 24), true, 0.774},
		// Tout neuf (1er commit aujourd'hui) : pas encore établi → exclu.
		{"tout neuf", makeDays(now, 0, 0, 1), false, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := featuredScore(c.days, now)
			assert.InDelta(t, c.approx, got, 0.02, "score")
			assert.Equal(t, c.flagshipish, got >= flagshipScore, "calibre-phare ?")
		})
	}
}
