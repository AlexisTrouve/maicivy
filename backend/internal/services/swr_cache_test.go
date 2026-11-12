package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Plafond des lectures en fond : 20 clés périmées lues d'un coup ne doivent jamais produire plus de
// `cap(limite)` appels SIMULTANÉS à l'amont. Panne du 2026-09-27 : 76 rafraîchissements parallèles
// ont saturé le worker unique de maiProFiles pendant 8 minutes.
func TestSwrStore_RafraichissementsEnFondPlafonnes(t *testing.T) {
	mr := miniredis.RunT(t)
	store := &swrStore{
		redis:     redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		nom:       "test",
		fraicheur: time.Minute,
		limite:    make(chan struct{}, 2),
	}
	ctx := context.Background()

	var enCours, maxVu, total atomic.Int32
	lent := func(context.Context) (interface{}, error) {
		n := enCours.Add(1)
		for {
			m := maxVu.Load()
			if n <= m || maxVu.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		enCours.Add(-1)
		total.Add(1)
		return map[string]string{"v": "frais"}, nil
	}
	rapide := func(context.Context) (interface{}, error) { return map[string]string{"v": "ancien"}, nil }

	const n = 20
	for i := 0; i < n; i++ {
		var out map[string]string
		require.NoError(t, store.lire(ctx, fmt.Sprintf("k%d", i), &out, rapide))
		vieillir(t, mr, fmt.Sprintf("k%d", i), 2*time.Minute)
	}

	// Rafale : 20 lectures de clés périmées → 20 rafraîchissements demandés d'un coup.
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var out map[string]string
			require.NoError(t, store.lire(ctx, fmt.Sprintf("k%d", i), &out, lent))
			require.Equal(t, "ancien", out["v"], "le périmé est servi sans attendre la file")
		}(i)
	}
	wg.Wait()

	require.Eventually(t, func() bool { return total.Load() == n }, 5*time.Second, 10*time.Millisecond,
		"tous les rafraîchissements doivent finir par passer")
	require.LessOrEqual(t, maxVu.Load(), int32(2), "jamais plus de 2 lectures simultanées vers l'amont")
}

// Warm ne relit PAS les pages déjà en cache, même périmées : seuls les livres (3 éditions) peuvent
// être rafraîchis. Avant, chaque démarrage relançait toutes les pages d'un coup.
func TestPortfolioCache_WarmNeRafraichitPasLesPagesDejaEnCache(t *testing.T) {
	c, src, mr := cacheDeTest(t)
	ctx := context.Background()
	c.Warm(ctx)

	for lang := range langsLivre {
		vieillir(t, mr, cleLivre(lang), 2*portfolioFraicheur)
		vieillir(t, mr, clePage(lang, "theomen"), 2*portfolioFraicheur)
	}
	avant := src.appels.Load()

	c.Warm(ctx)
	time.Sleep(200 * time.Millisecond) // laisse finir d'éventuels rafraîchissements en fond

	require.LessOrEqual(t, src.appels.Load()-avant, int32(len(langsLivre)),
		"au plus un rafraîchissement par livre ; aucune page déjà en cache ne doit être relue")
}
