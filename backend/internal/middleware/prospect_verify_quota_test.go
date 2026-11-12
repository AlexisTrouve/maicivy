package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : verrouille le plafond quotidien des vérifications Composer d'un prospecteur.
// POURQUOI : /admin/compose/verify appelle le modèle (traduction + checklist) à chaque tour. Avant ce
// plafond, un prospecteur pouvait le déclencher sans limite aux frais du propriétaire.

func newQuotaRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return mr, redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestProspectVerifyQuota_RefuseAuDelaDuPlafond(t *testing.T) {
	_, rdb := newQuotaRedis(t)
	q := NewProspectVerifyQuota(rdb, 2)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		ok, err := q.Consume(ctx, "p1")
		require.NoError(t, err)
		assert.True(t, ok, "appel %d sous le plafond", i)
	}
	ok, err := q.Consume(ctx, "p1")
	require.NoError(t, err)
	assert.False(t, ok, "le 3e appel dépasse le plafond de 2")

	// Le compteur est PAR prospecteur : un autre n'est pas touché.
	ok, err = q.Consume(ctx, "p2")
	require.NoError(t, err)
	assert.True(t, ok)
}

// Le compteur expire à minuit : sans TTL il serait immortel et bloquerait le prospecteur à vie.
func TestProspectVerifyQuota_CompteurAvecTTL(t *testing.T) {
	mr, rdb := newQuotaRedis(t)
	q := NewProspectVerifyQuota(rdb, 5)
	_, err := q.Consume(context.Background(), "p1")
	require.NoError(t, err)

	ttl := mr.TTL(prospectVerifyKey("p1"))
	assert.Greater(t, ttl, time.Duration(0), "la clé doit porter un TTL")
	assert.LessOrEqual(t, ttl, 24*time.Hour)
}

// Plafond nul ou Redis absent : on REFUSE (échec fermé), jamais un accès illimité.
func TestProspectVerifyQuota_EchecFerme(t *testing.T) {
	_, rdb := newQuotaRedis(t)
	ok, _ := NewProspectVerifyQuota(rdb, 0).Consume(context.Background(), "p1")
	assert.False(t, ok, "plafond 0 = aucune vérification")

	ok, err := NewProspectVerifyQuota(nil, 10).Consume(context.Background(), "p1")
	assert.False(t, ok)
	assert.Error(t, err, "sans Redis, l'erreur doit être nommée")

	var nul *ProspectVerifyQuota
	ok, err = nul.Consume(context.Background(), "p1")
	assert.False(t, ok)
	assert.Error(t, err)
}

// Plafond NÉGATIF = plafond levé (décision d'Alexi du 2026-10-15) : tout passe, sans rien compter,
// même sans Redis. Distinct de 0, qui reste « aucune vérification » (échec fermé).
func TestProspectVerifyQuota_PlafondLeve(t *testing.T) {
	mr, rdb := newQuotaRedis(t)
	q := NewProspectVerifyQuota(rdb, -1)
	for i := 0; i < 500; i++ {
		ok, err := q.Consume(context.Background(), "p1")
		require.NoError(t, err)
		require.True(t, ok, "appel %d refusé alors que le plafond est levé", i)
	}
	assert.False(t, mr.Exists(prospectVerifyKey("p1")), "un plafond levé ne compte rien")

	sansRedis := NewProspectVerifyQuota(nil, -1)
	ok, err := sansRedis.Consume(context.Background(), "p1")
	require.NoError(t, err)
	assert.True(t, ok)
}
