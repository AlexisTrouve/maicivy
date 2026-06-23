package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBgRefresher_SingleFlight verrouille le cœur anti-troupeau : pendant qu'un refresh tourne, tout
// nouveau trigger est ignoré ; une fois le flag retombé, un trigger repart.
func TestBgRefresher_SingleFlight(t *testing.T) {
	var r bgRefresher
	var runs int32
	started := make(chan struct{}, 1)
	block := make(chan struct{})
	firstDone := make(chan struct{})

	// 1er refresh : signale son démarrage, puis se bloque (fenêtre garantie où il est « en vol »).
	r.trigger(func(context.Context) {
		atomic.AddInt32(&runs, 1)
		started <- struct{}{}
		<-block
		close(firstDone)
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("le 1er refresh n'a pas démarré")
	}

	// Rafale pendant que le 1er est bloqué → tous ignorés (single-flight).
	for i := 0; i < 10; i++ {
		r.trigger(func(context.Context) { atomic.AddInt32(&runs, 1) })
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&runs), "un seul refresh à la fois malgré la rafale")

	// Libère le 1er et attends que le flag soit remis à false (le defer s'exécute après le retour de fn).
	close(block)
	<-firstDone
	require.Eventually(t, func() bool { return !r.running.Load() }, time.Second, 5*time.Millisecond,
		"le flag running doit retomber après la fin du refresh")

	// Un nouveau trigger doit maintenant repartir.
	secondRan := make(chan struct{})
	r.trigger(func(context.Context) { atomic.AddInt32(&runs, 1); close(secondRan) })
	select {
	case <-secondRan:
	case <-time.After(2 * time.Second):
		t.Fatal("le refresh n'a pas redémarré après libération du flag")
	}
	assert.Equal(t, int32(2), atomic.LoadInt32(&runs))
}

// TestBgRefresher_DetachedContext : fn reçoit un contexte non-nil et NON annulé (détaché de la requête).
func TestBgRefresher_DetachedContext(t *testing.T) {
	var r bgRefresher
	got := make(chan context.Context, 1)
	r.trigger(func(ctx context.Context) { got <- ctx })
	select {
	case ctx := <-got:
		require.NotNil(t, ctx)
		assert.NoError(t, ctx.Err(), "le ctx de fond ne doit pas être déjà annulé")
	case <-time.After(2 * time.Second):
		t.Fatal("fn jamais exécutée")
	}
}
