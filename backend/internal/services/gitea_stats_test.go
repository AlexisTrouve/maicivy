// backend/internal/services/gitea_stats_test.go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
// CommitDays (source du scoring vedette : âge, récence, régularité) n'est plus mergé jour par jour
// par une fonction dédiée — il se DÉRIVE de l'ensemble SHA→commitStat du repo, comme tout le reste.
// mergeCommitDays a donc disparu ; son invariant, lui, reste verrouillé ici : jours distincts, triés
// croissant, élagués hors fenêtre 6 mois. Un jour porté par plusieurs commits ne compte qu'une fois,
// sans quoi la « régularité » du scoring gonflerait avec le volume au lieu de mesurer la fréquence.
func TestDeriveStats_CommitDaysDistinctSortedAndRolledOff(t *testing.T) {
	const cutoff6m, cutoff30d = "2026-01-01", "2026-05-25"
	repos := []RepoStat{{Name: "alpha"}}
	commits := map[string]map[string]commitStat{
		"alpha": {
			"s1": {Day: "2026-06-10"},
			"s2": {Day: "2026-06-01"},
			"s3": {Day: "2026-06-10"}, // même jour que s1 → un seul jour distinct
			"s4": {Day: "2025-11-01"}, // hors fenêtre 6 mois → élagué
		},
	}
	resp := deriveStats(repos, commits, cutoff6m, cutoff30d)

	require.Len(t, resp.Repos, 1)
	assert.Equal(t, []string{"2026-06-01", "2026-06-10"}, resp.Repos[0].CommitDays,
		"jours distincts, triés, hors-fenêtre élagué")
	assert.Equal(t, 3, resp.Repos[0].Commits, "le jour est dédupliqué, PAS le commit : s1 et s3 comptent 2")
	assert.Equal(t, 3, resp.TotalCommits, "et le total dit la même chose")
}

// Un repo sans aucun commit ne doit pas être compté actif ni polluer la série journalière.
func TestDeriveStats_EmptyRepo(t *testing.T) {
	resp := deriveStats([]RepoStat{{Name: "vide"}}, map[string]map[string]commitStat{}, "2026-01-01", "2026-05-25")
	require.Len(t, resp.Repos, 1)
	assert.Equal(t, 0, resp.Repos[0].Commits)
	assert.Nil(t, resp.Repos[0].CommitDays, "aucun jour → nil, pas une slice vide")
	assert.Equal(t, 0, resp.ActiveRepos, "un repo sans commit n'est pas actif")
	assert.Empty(t, resp.Daily)
}

// Verrouille le merge des commits PAR REPO (clé SHA) — le correctif du bug d'empilement. L'ancien code
// faisait `r.Commits += c.Commits` à chaque fetch incrémental (30 min) avec un recouvrement de 1h → le
// même commit était recompté en boucle, sans rolloff 6 mois (cas réel : blender-mcp affichait 9300
// commits pour 0 jour d'activité). La clé SHA déduplique : un commit re-fetché = no-op.
func TestMergeRepoCommits(t *testing.T) {
	const cutoff = "2026-01-01"
	cases := []struct {
		name          string
		cached, fresh map[string]commitStat
		want          map[string]commitStat
	}{
		{
			name:   "union par SHA + dédup : re-fetch du même commit n'empile pas",
			cached: map[string]commitStat{"a": {Day: "2026-06-01"}, "b": {Day: "2026-06-02"}},
			fresh:  map[string]commitStat{"b": {Day: "2026-06-02"}, "c": {Day: "2026-06-03"}}, // b déjà connu
			want:   map[string]commitStat{"a": {Day: "2026-06-01"}, "b": {Day: "2026-06-02"}, "c": {Day: "2026-06-03"}},
		},
		{
			name:   "rolloff : les SHA dont le jour < cutoff sont élagués",
			cached: map[string]commitStat{"old": {Day: "2025-11-01"}, "keep": {Day: "2026-06-10"}},
			fresh:  map[string]commitStat{"new": {Day: "2026-06-15"}},
			want:   map[string]commitStat{"keep": {Day: "2026-06-10"}, "new": {Day: "2026-06-15"}},
		},
		{
			name:   "tout vide → map vide (non nil)",
			cached: nil, fresh: nil,
			want: map[string]commitStat{},
		},
		{
			// Le cache ne doit RIEN perdre quand la fenêtre fraîche est partielle : c'est l'union qui
			// interdit l'érosion du journalier (cf. gitea_stats_consistency_test.go).
			name:   "fenêtre fraîche partielle : le cache n'est jamais retranché",
			cached: map[string]commitStat{"a": {Day: "2026-06-01"}, "b": {Day: "2026-06-01"}, "c": {Day: "2026-06-01"}},
			fresh:  map[string]commitStat{"c": {Day: "2026-06-01"}},
			want: map[string]commitStat{
				"a": {Day: "2026-06-01"}, "b": {Day: "2026-06-01"}, "c": {Day: "2026-06-01"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, mergeRepoCommits(c.cached, c.fresh, cutoff))
		})
	}

	// Anti-régression du bug d'empilement : merger N fois le MÊME lot frais ne fait JAMAIS grossir le set.
	cached := map[string]commitStat{"a": {Day: "2026-06-01"}}
	fresh := map[string]commitStat{"a": {Day: "2026-06-01"}, "b": {Day: "2026-06-02"}} // recouvrement
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
	sha := map[string]commitStat{
		"a": {Day: "2026-06-20"}, // dans 30j
		"b": {Day: "2026-06-01"}, // dans 30j
		"c": {Day: "2026-05-26"}, // dans 30j (>= cutoff)
		"d": {Day: "2026-05-24"}, // hors 30j (< cutoff) mais dans 6 mois
		"e": {Day: "2026-02-01"}, // hors 30j
	}
	total, last30 := repoCommitCounts(sha, cutoff30d)
	assert.Equal(t, 5, total, "total = tous les SHA de la fenêtre 6 mois")
	assert.Equal(t, 3, last30, "30j = SHA dont le jour >= cutoff30d")

	t0, l0 := repoCommitCounts(nil, cutoff30d)
	assert.Equal(t, 0, t0, "nil (repo sans commit / fork) → 0")
	assert.Equal(t, 0, l0, "nil → 0")
}
