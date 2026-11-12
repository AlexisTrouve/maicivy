package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Panne réelle du 2026-10-09 15:35 UTC. Un clic sur « Refresh » de /gitstats (?force=true) a SUPPRIMÉ
// le cache puis lancé un fetch complet de 6 mois. Pendant ce fetch, chaque autre appelant de GetStats
// (métriques démo à chaque heartbeat, CV, page gitstats) trouvait le cache vide et lançait SON PROPRE
// fetch complet, sans verrou : 30 fetches complets en parallèle, chacun = 145 repos × git diff côté
// Gitea. Charge VPS142 à 32, Gitea en timeout, maiProFiles affamé → blog et portfolio en 502.
// Ces tests verrouillent les deux maillons : un seul fetch complet à la fois, et un force qui ne
// retire jamais le cache aux autres.

// newBlockingGitea monte un faux Gitea qui compte les fetches COMPLETS (un fetch = un appel à
// /repos/search, première étape de fetchStats) et bloque chacun jusqu'à `release`.
func newBlockingGitea(t *testing.T) (svc *GiteaStatsService, mr *miniredis.Miniredis, fetches *int32, started chan struct{}, release func()) {
	t.Helper()
	mr = miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	var n int32
	started = make(chan struct{}, 64)
	gate := make(chan struct{})
	var once sync.Once
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release) // jamais de handler pendu à la fermeture du serveur

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repos/search") {
			atomic.AddInt32(&n, 1)
			started <- struct{}{}
			<-gate
			_, _ = w.Write([]byte(`{"ok":true,"data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	svc = NewGiteaStatsService(rdb, server.URL, "tok", "StillHammer")
	require.NotNil(t, svc)
	return svc, mr, &n, started, release
}

// Cache vide + rafale d'appelants concurrents → UN SEUL fetch complet, partagé par tous.
func TestGetStats_NoCache_ConcurrentCallersShareOneFullFetch(t *testing.T) {
	svc, _, fetches, started, release := newBlockingGitea(t)

	const callers = 10
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.GetStats(context.Background(), false)
			errs <- err
		}()
	}

	// Attendre que le premier fetch soit en vol, laisser aux autres le temps d'arriver.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("aucun fetch n'a démarré")
	}
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(fetches),
		"cache vide sous rafale : un seul fetch complet doit partir, pas un par appelant")

	release()
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "chaque appelant reçoit le résultat du fetch partagé")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(fetches))
}

// Un force=true ne retire PAS le cache aux autres : pendant le refetch forcé, les appelants ordinaires
// continuent d'être servis depuis le cache existant, sans lancer le moindre fetch.
func TestGetStats_ForceKeepsCacheForOthers(t *testing.T) {
	svc, mr, fetches, started, release := newBlockingGitea(t)
	seedStaleStatsCache(t, mr) // sentinelle TotalCommits=42 (cache périmé)

	forceDone := make(chan error, 1)
	go func() {
		_, err := svc.GetStats(context.Background(), true)
		forceDone <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("le fetch forcé n'a pas démarré")
	}

	// Pendant le fetch forcé : le cache doit toujours exister et être servi.
	assert.True(t, mr.Exists(cacheKey), "force ne doit pas supprimer le cache pendant le refetch")
	for i := 0; i < 5; i++ {
		// Appel borné dans le temps : si l'appelant se met à attendre le fetch forcé (bloqué ici
		// jusqu'à release), c'est exactement la panne — on échoue au lieu de pendre.
		type res struct {
			resp *GitStatsResponse
			err  error
		}
		got := make(chan res, 1)
		go func() {
			r, err := svc.GetStats(context.Background(), false)
			got <- res{r, err}
		}()
		select {
		case r := <-got:
			require.NoError(t, r.err)
			assert.Equal(t, 42, r.resp.TotalCommits, "les autres appelants restent servis depuis le cache")
		case <-time.After(time.Second):
			release() // débloque les goroutines avant d'échouer
			t.Fatal("un appelant ordinaire attend le fetch forcé au lieu d'être servi depuis le cache")
		}
	}
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(fetches),
		"pendant un force, aucun autre fetch ne doit partir (ni complet, ni de fond)")

	release()
	require.NoError(t, <-forceDone)
}

// newSinceRecordingGitea : faux Gitea avec UN repo, qui enregistre le paramètre `since` de chaque
// appel /commits — c'est lui qui dit jusqu'où le refresh remonte.
func newSinceRecordingGitea(t *testing.T) (*GiteaStatsService, *miniredis.Miniredis, func() []time.Time) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	var mu sync.Mutex
	var sinces []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/repos/search"):
			_, _ = w.Write([]byte(`{"ok":true,"data":[{"name":"r1","default_branch":"main","updated_at":"2026-10-01T00:00:00Z"}]}`))
		case strings.Contains(r.URL.Path, "/commits"):
			if ts, err := time.Parse(time.RFC3339, r.URL.Query().Get("since")); err == nil {
				mu.Lock()
				sinces = append(sinces, ts)
				mu.Unlock()
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	t.Cleanup(server.Close)
	svc := NewGiteaStatsService(rdb, server.URL, "tok", "StillHammer")
	require.NotNil(t, svc)
	return svc, mr, func() []time.Time { mu.Lock(); defer mu.Unlock(); return append([]time.Time(nil), sinces...) }
}

func seedStatsCacheFetchedAt(t *testing.T, mr *miniredis.Miniredis, at time.Time) {
	t.Helper()
	data, err := json.Marshal(gitStatsCache{Response: GitStatsResponse{TotalCommits: 42, Period: "6months"}, FetchedAt: at})
	require.NoError(t, err)
	require.NoError(t, mr.Set(cacheKey, string(data)))
}

// Refresh (force) avec un cache existant = relire les DERNIERS JOURS, jamais les 6 mois.
// Cache récent (1 min) : on remonte quand même de forceRefreshWindow, pas seulement d'1h.
func TestGetStats_ForceWithCache_RefetchesRecentDaysOnly(t *testing.T) {
	svc, mr, sinces := newSinceRecordingGitea(t)
	seedStatsCacheFetchedAt(t, mr, time.Now().Add(-1*time.Minute))

	_, err := svc.GetStats(context.Background(), true)
	require.NoError(t, err)

	got := sinces()
	require.NotEmpty(t, got, "le refresh doit interroger les commits")
	for _, s := range got {
		age := time.Since(s)
		assert.Less(t, age, forceRefreshWindow+time.Hour, "force ne doit PAS remonter 6 mois (since=%s)", s)
		assert.Greater(t, age, forceRefreshWindow-time.Hour, "force doit couvrir les derniers jours, pas seulement 1h (since=%s)", s)
	}
}

// Cache plus vieux que la fenêtre : on remonte jusqu'au dernier fetch, sinon un trou apparaîtrait.
func TestGetStats_ForceWithOldCache_CoversSinceLastFetch(t *testing.T) {
	svc, mr, sinces := newSinceRecordingGitea(t)
	last := time.Now().Add(-10 * 24 * time.Hour)
	seedStatsCacheFetchedAt(t, mr, last)

	_, err := svc.GetStats(context.Background(), true)
	require.NoError(t, err)
	for _, s := range sinces() {
		assert.False(t, s.After(last), "since=%s doit être ≤ dernier fetch %s (pas de trou)", s, last)
		assert.Less(t, time.Since(s), 30*24*time.Hour, "pas de refetch 6 mois (since=%s)", s)
	}
}

// Même invariant pour les langages : cache vide sous rafale → un seul fetch.
func TestGetLanguageStats_NoCache_ConcurrentCallersShareOneFetch(t *testing.T) {
	svc, _, fetches, started, release := newBlockingGitea(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.GetLanguageStats(context.Background(), false)
		}()
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("aucun fetch n'a démarré")
	}
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(fetches), "langages : un seul fetch sous rafale")
	release()
	wg.Wait()
}
