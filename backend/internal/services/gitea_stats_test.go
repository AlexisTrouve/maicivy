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
		name          string
		cached, fresh []string
		cutoff        string
		want          []string
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

// Verrouille le merge des commits PAR REPO (clé SHA) — le correctif du bug d'empilement. L'ancien code
// faisait `r.Commits += c.Commits` à chaque fetch incrémental (30 min) avec un recouvrement de 1h → le
// même commit était recompté en boucle, sans rolloff 6 mois (cas réel : blender-mcp affichait 9300
// commits pour 0 jour d'activité). La clé SHA déduplique : un commit re-fetché = no-op.
func TestMergeRepoCommits(t *testing.T) {
	const cutoff = "2026-01-01"
	cases := []struct {
		name          string
		cached, fresh map[string]string
		want          map[string]string
	}{
		{
			name:   "union par SHA + dédup : re-fetch du même commit n'empile pas",
			cached: map[string]string{"a": "2026-06-01", "b": "2026-06-02"},
			fresh:  map[string]string{"b": "2026-06-02", "c": "2026-06-03"}, // b déjà connu
			want:   map[string]string{"a": "2026-06-01", "b": "2026-06-02", "c": "2026-06-03"},
		},
		{
			name:   "rolloff : les SHA dont le jour < cutoff sont élagués",
			cached: map[string]string{"old": "2025-11-01", "keep": "2026-06-10"},
			fresh:  map[string]string{"new": "2026-06-15"},
			want:   map[string]string{"keep": "2026-06-10", "new": "2026-06-15"},
		},
		{
			name:   "tout vide → map vide (non nil)",
			cached: nil, fresh: nil,
			want: map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, mergeRepoCommits(c.cached, c.fresh, cutoff))
		})
	}

	// Anti-régression du bug d'empilement : merger N fois le MÊME lot frais ne fait JAMAIS grossir le set.
	cached := map[string]string{"a": "2026-06-01"}
	fresh := map[string]string{"a": "2026-06-01", "b": "2026-06-02"} // recouvrement avec le cache
	merged := mergeRepoCommits(cached, fresh, cutoff)
	for i := 0; i < 5; i++ {
		merged = mergeRepoCommits(merged, fresh, cutoff)
	}
	assert.Len(t, merged, 2, "re-merge répété du même lot ne doit pas accumuler (bug r.Commits += c.Commits)")
}

// Verrouille la dérivation des compteurs affichés depuis l'ensemble SHA→jour : total 6 mois + 30 jours
// glissants (ce qu'affiche le badge "Repos chauds en ce moment").
func TestRepoCommitCounts(t *testing.T) {
	const cutoff30d = "2026-05-25"
	sha := map[string]string{
		"a": "2026-06-20", // dans 30j
		"b": "2026-06-01", // dans 30j
		"c": "2026-05-26", // dans 30j (>= cutoff)
		"d": "2026-05-24", // hors 30j (< cutoff) mais dans 6 mois
		"e": "2026-02-01", // hors 30j
	}
	total, last30 := repoCommitCounts(sha, cutoff30d)
	assert.Equal(t, 5, total, "total = tous les SHA de la fenêtre 6 mois")
	assert.Equal(t, 3, last30, "30j = SHA dont le jour >= cutoff30d")

	t0, l0 := repoCommitCounts(nil, cutoff30d)
	assert.Equal(t, 0, t0, "nil (repo sans commit / fork) → 0")
	assert.Equal(t, 0, l0, "nil → 0")
}
