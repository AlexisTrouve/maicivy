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

// newSWRTestService monte un GiteaStatsService branché sur miniredis + un faux serveur Gitea LENT.
// Le serveur compte ses hits et dort `delay` à chaque appel : si la requête bloquait sur le refetch,
// le temps écoulé le trahirait. Retourne le service, le handle miniredis et un compteur de hits atomique.
func newSWRTestService(t *testing.T, delay time.Duration) (*GiteaStatsService, *miniredis.Miniredis, *int32) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		time.Sleep(delay) // refetch volontairement lent → révèle tout blocage en avant-plan
		// repos/search → 0 repo (suffit : fetchStats renvoie une réponse vide sans appeler /commits).
		if strings.Contains(r.URL.Path, "/repos/search") {
			_, _ = w.Write([]byte(`{"ok":true,"data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	svc := NewGiteaStatsService(rdb, server.URL, "tok", "StillHammer")
	require.NotNil(t, svc, "service ne doit pas être nil avec une config valide")
	return svc, mr, &hits
}

// seedStaleStatsCache écrit un cache gitstats PÉRIMÉ (FetchedAt = il y a 1h > minFetchInterval de 30min)
// avec une valeur sentinelle reconnaissable (TotalCommits=42) pour prouver qu'on sert bien CE cache-là.
func seedStaleStatsCache(t *testing.T, mr *miniredis.Miniredis) {
	t.Helper()
	stale := gitStatsCache{
		Response:  GitStatsResponse{TotalCommits: 42, Period: "6months"},
		FetchedAt: time.Now().Add(-1 * time.Hour),
	}
	data, err := json.Marshal(stale)
	require.NoError(t, err)
	require.NoError(t, mr.Set(cacheKey, string(data)))
}

// TestGetStats_StaleServedImmediately — cœur du fix stale-while-revalidate.
// Cache périmé → GetStats doit rendre le STALE TOUT DE SUITE (sans attendre le refetch Gitea lent) et
// déclencher le rafraîchissement en FOND. Avec l'ancien code (refetch synchrone), ce test échoue :
// l'appel bloquait `delay` ET retournait la réponse fraîche (TotalCommits=0), pas le stale (42).
func TestGetStats_StaleServedImmediately(t *testing.T) {
	const delay = 400 * time.Millisecond
	svc, mr, hits := newSWRTestService(t, delay)
	seedStaleStatsCache(t, mr)

	start := time.Now()
	resp, err := svc.GetStats(context.Background(), false)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	// 1. On sert le STALE, pas la réponse fraîche.
	assert.Equal(t, 42, resp.TotalCommits, "doit servir le cache stale (sentinelle 42)")
	// 2. On n'a PAS attendu le refetch lent.
	assert.Less(t, elapsed, delay/2, "GetStats ne doit pas bloquer sur le refetch Gitea (%s)", delay)

	// 3. Le refresh de fond doit finir par taper Gitea (≥1 hit) dans un délai raisonnable.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(hits) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.GreaterOrEqual(t, atomic.LoadInt32(hits), int32(1), "le refresh de fond doit avoir tapé Gitea")
}

// TestGetStats_FreshCacheNoFetch — garde-fou : un cache FRAIS (< 30min) est rendu tel quel, zéro appel
// réseau (le chemin nominal ne doit jamais toucher Gitea).
func TestGetStats_FreshCacheNoFetch(t *testing.T) {
	svc, mr, hits := newSWRTestService(t, 10*time.Millisecond)
	fresh := gitStatsCache{
		Response:  GitStatsResponse{TotalCommits: 7, Period: "6months"},
		FetchedAt: time.Now().Add(-1 * time.Minute), // frais
	}
	data, _ := json.Marshal(fresh)
	require.NoError(t, mr.Set(cacheKey, string(data)))

	resp, err := svc.GetStats(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, 7, resp.TotalCommits)

	time.Sleep(100 * time.Millisecond) // laisse une éventuelle goroutine fautive s'exécuter
	assert.Equal(t, int32(0), atomic.LoadInt32(hits), "cache frais → aucun appel Gitea")
}

// TestGetStats_SingleFlight — prouve que le CÂBLAGE de GetStats utilise bien le single-flight : pendant
// qu'un refresh de fond est en vol, une rafale de requêtes sur le cache stale ne déclenche AUCUN refetch
// Gitea supplémentaire (un seul hit). Sans le verrou, chaque requête lancerait sa propre goroutine.
func TestGetStats_SingleFlight(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	var hits int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var once sync.Once
	doRelease := func() { once.Do(func() { close(release) }) }
	defer doRelease() // débloque le handler en toute circonstance (sinon server.Close pend)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release // bloque le refetch de fond → fenêtre garantie où le refresh est « en vol »
		if strings.Contains(r.URL.Path, "/repos/search") {
			_, _ = w.Write([]byte(`{"ok":true,"data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	svc := NewGiteaStatsService(rdb, server.URL, "tok", "StillHammer")
	require.NotNil(t, svc)
	seedStaleStatsCache(t, mr)

	// 1er appel → déclenche le refresh de fond (qui va se bloquer sur release).
	_, err := svc.GetStats(context.Background(), false)
	require.NoError(t, err)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("le refresh de fond n'a pas démarré")
	}

	// Rafale pendant que ce refresh est bloqué : aucun nouveau refetch ne doit partir.
	for i := 0; i < 5; i++ {
		_, err := svc.GetStats(context.Background(), false)
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits),
		"single-flight: un seul refetch Gitea malgré la rafale de requêtes")

	doRelease()
}
