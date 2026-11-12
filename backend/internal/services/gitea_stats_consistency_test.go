package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Invariant central de ce fichier ---------------------------------------------------------
//
// QUOI : les trois compteurs servis par /cv/gitstats — la série journalière (Daily), les compteurs
// par repo (Repos[].Commits) et le total (TotalCommits) — doivent compter LA MÊME CHOSE : un commit
// = un SHA distinct dans la fenêtre 6 mois. Aucun des trois n'a le droit de diverger des autres.
//
// POURQUOI ce fichier existe : ils divergeaient en production (mesuré le 2026-08-11 : 8949 commits
// côté compteurs par repo contre 8026 côté journalier). Deux causes distinctes, toutes deux muettes :
//
//  1. ÉROSION du journalier. Le fetch incrémental ne demande à Gitea que les commits depuis
//     `fetchedAt - 1h`, puis ÉCRASE le jour entier avec ce résultat partiel. Le jour courant perdait
//     donc tous ses commits antérieurs à cette fenêtre, définitivement (une fois minuit passé, ce jour
//     n'est plus jamais refetché). Les compteurs par repo, eux, avaient déjà été corrigés en union
//     dédupliquée par SHA — d'où le désaccord, et d'où le fait que seul le journalier mentait.
//
//  2. DÉFINITION DIVERGENTE d'« un commit ». Le filtre isMassiveCommit (commits de tuyauterie :
//     vendoring, migration LFS) doit exclure ces commits des agrégats de LIGNES. La boucle journalière
//     faisait `continue`, ce qui les retirait AUSSI du comptage de commits ; l'ensemble SHA des repos
//     les gardait. Un commit massif comptait donc 1 d'un côté et 0 de l'autre.
//
// COMMENT on le verrouille : bout en bout contre un faux serveur Gitea, à travers GetStats — le bug
// vit dans la glue du merge incrémental, un test sur les fonctions pures seules le raterait.

// fakeGitea sert un unique repo `alpha` et une liste de commits qu'on peut changer entre deux phases,
// pour rejouer exactement le scénario réel : un fetch complet, puis un fetch incrémental dont la
// fenêtre étroite ne ramène qu'une PARTIE des commits d'un jour déjà connu.
type fakeGitea struct {
	mu      sync.Mutex
	commits []string // fragments JSON de commits, servis tels quels
}

func (f *fakeGitea) setCommits(jsonCommits ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commits = jsonCommits
}

func (f *fakeGitea) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liste des repos : un seul repo non vide, branche par défaut `main`.
		if strings.Contains(r.URL.Path, "/repos/search") {
			_, _ = fmt.Fprintf(w, `{"ok":true,"data":[{"name":"alpha","description":"d","language":"Go",
				"stars_count":1,"updated_at":"2026-08-01T00:00:00Z","empty":false,"fork":false,
				"default_branch":"main"}]}`)
			return
		}
		if strings.Contains(r.URL.Path, "/commits") {
			f.mu.Lock()
			body := "[" + strings.Join(f.commits, ",") + "]"
			f.mu.Unlock()
			// listCommits s'arrête dès qu'une page renvoie < 50 éléments → une seule page suffit,
			// mais la pagination demande la page 2 si on en renvoyait 50. On reste sous le seuil.
			if strings.Contains(r.URL.RawQuery, "page=1") {
				_, _ = w.Write([]byte(body))
				return
			}
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}
}

// commitJSON fabrique un commit Gitea au format attendu par giteaCommit.
func commitJSON(sha, day string, additions, deletions int) string {
	return fmt.Sprintf(`{"sha":%q,"commit":{"author":{"date":"%sT12:00:00Z"}},
		"stats":{"additions":%d,"deletions":%d}}`, sha, day, additions, deletions)
}

// newConsistencyService monte le service sur miniredis + le faux Gitea.
func newConsistencyService(t *testing.T) (*GiteaStatsService, *miniredis.Miniredis, *fakeGitea) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	fake := &fakeGitea{}
	server := httptest.NewServer(fake.handler())
	t.Cleanup(server.Close)
	svc := NewGiteaStatsService(rdb, server.URL, "tok", "StillHammer")
	require.NotNil(t, svc)
	return svc, mr, fake
}

// backdateCache vieillit le cache pour rendre le prochain GetStats "périmé" (> minFetchInterval),
// ce qui déclenche le chemin incrémental. On manipule le JSON en map générique et NON via
// gitStatsCache : le test reste ainsi indépendant du format interne du cache, qu'on fait justement
// évoluer. Retourne la valeur de fetchedAt posée, pour pouvoir détecter la fin du refresh de fond.
func backdateCache(t *testing.T, mr *miniredis.Miniredis, age time.Duration) string {
	t.Helper()
	raw, err := mr.Get(cacheKey)
	require.NoError(t, err, "le cache doit exister avant d'être vieilli")
	var blob map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &blob))
	stamp := time.Now().Add(-age).Format(time.RFC3339Nano)
	blob["fetchedAt"] = stamp
	out, err := json.Marshal(blob)
	require.NoError(t, err)
	require.NoError(t, mr.Set(cacheKey, string(out)))
	return stamp
}

// waitForRefresh attend que le rafraîchissement de FOND ait réécrit le cache (fetchedAt change).
// POURQUOI : GetStats rend le stale immédiatement et refetche en tâche de fond (stale-while-
// revalidate) ; sans cette attente on assertait sur l'ancienne valeur et le test serait un faux vert.
func waitForRefresh(t *testing.T, mr *miniredis.Miniredis, previousStamp string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := mr.Get(cacheKey)
		if err == nil {
			var blob map[string]any
			if json.Unmarshal([]byte(raw), &blob) == nil {
				if stamp, _ := blob["fetchedAt"].(string); stamp != previousStamp {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("le rafraîchissement de fond n'a jamais réécrit le cache")
}

// sumDaily / sumRepos — les deux façons de compter, qui doivent tomber sur le même nombre.
func sumDaily(resp *GitStatsResponse) int {
	n := 0
	for _, d := range resp.Daily {
		n += d.Commits
	}
	return n
}

func sumRepos(resp *GitStatsResponse) int {
	n := 0
	for _, r := range resp.Repos {
		n += r.Commits
	}
	return n
}

// TestIncrementalMerge_PartialWindowDoesNotErodeDay — LE bug de production.
//
// Scénario : un jour D porte 4 commits, connus du cache. Le fetch incrémental suivant a une fenêtre
// étroite (fetchedAt - 1h) et ne ramène que 2 commits de ce jour (1 déjà connu — le recouvrement
// volontaire de 1h — et 1 nouveau). Le jour D doit alors valoir 5 commits (union dédupliquée par SHA),
// et surtout JAMAIS 2 : un fetch incrémental ne peut pas faire DIMINUER un jour déjà observé.
//
// Avant le fix : `dailyMap[d.Date] = d` écrasait le jour avec le résultat partiel → 2. Les compteurs
// par repo, en union SHA, disaient 5. C'est exactement l'écart mesuré en production.
func TestIncrementalMerge_PartialWindowDoesNotErodeDay(t *testing.T) {
	svc, mr, fake := newConsistencyService(t)
	day := time.Now().AddDate(0, 0, -2).Format("2006-01-02") // jour fixe, à l'abri du passage de minuit

	// Phase 1 — fetch complet : les 4 commits du jour D sont connus.
	fake.setCommits(
		commitJSON("sha1", day, 10, 1),
		commitJSON("sha2", day, 20, 2),
		commitJSON("sha3", day, 30, 3),
		commitJSON("sha4", day, 40, 4),
	)
	full, err := svc.GetStats(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 4, full.TotalCommits, "prérequis : le fetch complet voit les 4 commits")

	// Phase 2 — la fenêtre incrémentale ne ramène qu'une partie du jour D : sha4 (recouvrement) + sha5.
	fake.setCommits(
		commitJSON("sha4", day, 40, 4),
		commitJSON("sha5", day, 50, 5),
	)
	stamp := backdateCache(t, mr, time.Hour)

	_, err = svc.GetStats(context.Background(), false) // rend le stale, refetche en fond
	require.NoError(t, err)
	waitForRefresh(t, mr, stamp)

	merged, err := svc.GetStats(context.Background(), false) // cache frais → réponse mergée
	require.NoError(t, err)

	assert.Equal(t, 5, merged.TotalCommits, "union dédupliquée : 4 connus + sha5, sha4 ne compte qu'une fois")
	assert.Equal(t, 5, sumDaily(merged), "la série journalière ne doit pas être érodée par une fenêtre partielle")
	assert.Equal(t, 5, sumRepos(merged), "les compteurs par repo doivent dire la même chose")
	assert.Equal(t, sumDaily(merged), sumRepos(merged), "INVARIANT : journalier et par-repo comptent la même chose")
}

// TestMassiveCommit_SameDefinitionEverywhere — un commit massif reste UN COMMIT.
//
// Le filtre isMassiveCommit sert à ne pas fausser les compteurs de LIGNES (un vendoring de 60k lignes
// n'est pas du travail d'écriture). Il ne doit pas faire disparaître le commit du COMPTAGE : sinon
// Repos[].Commits (ensemble SHA, qui les garde) et Daily (qui les jetait) ne parlent plus de la même
// chose — la seconde cause de l'écart de production.
func TestMassiveCommit_SameDefinitionEverywhere(t *testing.T) {
	svc, _, fake := newConsistencyService(t)
	day := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

	fake.setCommits(
		commitJSON("normal", day, 100, 10),
		commitJSON("vendoring", day, massiveCommitThreshold+1, 0), // tuyauterie : lignes exclues, commit compté
	)
	resp, err := svc.GetStats(context.Background(), true)
	require.NoError(t, err)

	assert.Equal(t, 2, sumDaily(resp), "un commit massif reste un commit dans la série journalière")
	assert.Equal(t, 2, sumRepos(resp), "et dans les compteurs par repo")
	assert.Equal(t, 2, resp.TotalCommits, "et dans le total")
	assert.Equal(t, 100, resp.TotalAdded, "mais ses LIGNES restent exclues des agrégats LOC")
	assert.Equal(t, 10, resp.TotalDeleted, "idem pour les suppressions")
}

// TestIncrementalMerge_ActiveReposSurvivesEmptyWindow — un repo sans commit dans la DERNIÈRE HEURE
// reste un repo actif sur 6 mois.
//
// ActiveRepos était repris tel quel du fetch incrémental (`newData.ActiveRepos`), c'est-à-dire compté
// sur la fenêtre étroite : après un refresh sans nouveau commit, le compteur « repos actifs » du CV
// tombait à 0 alors que rien n'avait changé. Comme le reste, il doit se dériver de la fenêtre 6 mois.
func TestIncrementalMerge_ActiveReposSurvivesEmptyWindow(t *testing.T) {
	svc, mr, fake := newConsistencyService(t)
	day := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

	fake.setCommits(commitJSON("sha1", day, 10, 1))
	full, err := svc.GetStats(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, 1, full.ActiveRepos, "prérequis : le repo est actif après le fetch complet")

	fake.setCommits() // plus rien dans la fenêtre incrémentale
	stamp := backdateCache(t, mr, time.Hour)
	_, err = svc.GetStats(context.Background(), false)
	require.NoError(t, err)
	waitForRefresh(t, mr, stamp)

	merged, err := svc.GetStats(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, 1, merged.ActiveRepos, "le repo reste actif sur 6 mois malgré une fenêtre vide")
	assert.Equal(t, 1, merged.TotalCommits, "et son commit n'est pas perdu")
}
