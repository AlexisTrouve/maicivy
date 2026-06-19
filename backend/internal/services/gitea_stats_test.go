// backend/internal/services/gitea_stats_test.go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Verrouille l'exclusion des commits "tuyauterie" des agrégats LOC. Le filtre doit être SYMÉTRIQUE :
// un vendoring/import gonfle les additions, une migration LFS / purge de dossier généré gonfle les
// suppressions. Régression réelle verrouillée ici : la migration LFS de ChineseClass (1599 ajouts,
// 1 187 366 suppressions) passait l'ancien filtre additions-seul et faussait TotalDeleted.
func TestIsMassiveCommit(t *testing.T) {
	mk := func(add, del int) giteaCommit {
		c := giteaCommit{}
		c.Stats.Additions = add
		c.Stats.Deletions = del
		return c
	}
	cases := []struct {
		name string
		c    giteaCommit
		want bool
	}{
		{"commit de dev normal", mk(120, 45), false},
		{"gros ajout (vendoring/import)", mk(60000, 0), true},
		{"migration LFS ChineseClass — grosse suppression, petit ajout", mk(1599, 1187366), true},
		{"pile au seuil → gardé (strictement >)", mk(50000, 50000), false},
		{"juste au-dessus du seuil en suppression", mk(0, 50001), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isMassiveCommit(c.c))
		})
	}
}

// Verrouille le merge incrémental des jours de commit : UNION (pas addition → pas de gonflement
// de la régularité à chaque fetch de 30 min), dédup, et rolloff des jours antérieurs au cutoff 6 mois.
func TestMergeCommitDays(t *testing.T) {
	cases := []struct {
		name           string
		cached, fresh  []string
		cutoff         string
		want           []string
	}{
		{
			name:   "union + dédup + tri",
			cached: []string{"2026-06-10", "2026-06-01"},
			fresh:  []string{"2026-06-15", "2026-06-10"}, // 06-10 en double → 1 seule fois
			cutoff: "2026-01-01",
			want:   []string{"2026-06-01", "2026-06-10", "2026-06-15"},
		},
		{
			name:   "rolloff : les jours < cutoff sont élagués",
			cached: []string{"2025-11-01", "2026-06-10"}, // 2025-11-01 hors fenêtre
			fresh:  []string{"2026-06-15"},
			cutoff: "2026-01-01",
			want:   []string{"2026-06-10", "2026-06-15"},
		},
		{
			name:   "tout vide",
			cached: nil, fresh: nil, cutoff: "2026-01-01",
			want: nil,
		},
		{
			name:   "rolloff total → nil",
			cached: []string{"2025-01-01"}, fresh: []string{"2025-02-01"}, cutoff: "2026-01-01",
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, mergeCommitDays(c.cached, c.fresh, c.cutoff))
		})
	}
}
