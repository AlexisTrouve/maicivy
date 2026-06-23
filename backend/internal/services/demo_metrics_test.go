package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestBump(t *testing.T) {
	assert.InDelta(t, 0, bump(0), 1e-9)
	assert.InDelta(t, 1, bump(1), 1e-9) // pic au point de convergence
	assert.Less(t, bump(0.2), bump(1))  // monte avant
	assert.Less(t, bump(4), bump(1))    // redescend après
	assert.Less(t, bump(10), bump(4))   // continue de s'effacer
}

// Cœur du modèle : le scaling demandé par Alexi (bas si peu de vrais users même avec gros commits ;
// pic au point de convergence ; s'efface quand le trafic réel devient gros).
func TestBlendFlow_RealGatedByUsers(t *testing.T) {
	tFixed := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC) // 15h → courbe horaire ~max
	const floor, potential = 12.0, 300.0                    // potentiel élevé (commits élevés)

	synthAt := func(R float64) float64 { return blendFlow(0, R, floor, potential, tFixed) } // real=0 → isole le synthétique

	low := synthAt(3)                    // peu de vrais users
	conv := synthAt(demoConvergence)     // au point commun
	high := synthAt(demoConvergence * 5) // gros trafic

	// 1) Peu de users → bas MÊME avec potentiel 300 (les commits ne blow-up pas).
	assert.Less(t, low, 0.25*potential, "peu de vrais users → synthétique bas malgré potentiel élevé")
	// 2) Au point commun → le potentiel s'exprime.
	assert.Greater(t, conv, 0.6*potential, "à convergence → le potentiel commits s'exprime")
	assert.Greater(t, conv, low)
	// 3) Gros trafic → s'efface (laisse la place au réel).
	assert.Less(t, high, conv, "gros trafic → le synthétique s'efface")
	assert.Less(t, high, 0.5*conv)
}

// Les commits pilotent le potentiel : plus de commits → plafond plus haut (mais borné).
func TestCommitSeed_Potentials(t *testing.T) {
	quiet := commitSeed{total: 50, recent7: 1}
	active := commitSeed{total: 4886, recent7: 60}

	assert.Greater(t, active.lettersPotential(), quiet.lettersPotential())
	assert.Greater(t, active.visitorsPotential(), quiet.visitorsPotential())
	assert.Greater(t, active.onlinePotential(), quiet.onlinePotential())
	assert.LessOrEqual(t, active.onlinePotential(), 11.0)               // borné
	assert.LessOrEqual(t, active.visitorsPotential(), demoFloorVisitors+291) // borné
}

// hourCurve : creux la nuit, pic l'après-midi.
func TestHourCurve(t *testing.T) {
	night := hourCurve(time.Date(2026, 6, 23, 3, 0, 0, 0, time.UTC))
	day := hourCurve(time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC))
	assert.Less(t, night, day)
	assert.GreaterOrEqual(t, night, 0.3)
	assert.LessOrEqual(t, day, 1.0)
}

// Toggle OFF → passthrough total (aucune valeur synthétique).
func TestDemoMetrics_DisabledPassthrough(t *testing.T) {
	d := NewDemoMetrics(nil, nil, false)
	ctx := context.Background()
	assert.Equal(t, int64(2), d.Online(ctx, 2, 50))
	assert.Equal(t, int64(7), d.VisitorsToday(ctx, 7, 50))
	assert.Equal(t, int64(100), d.Letters(ctx, 100, 50))
	assert.Equal(t, int64(5), d.BlogReadsTotal(ctx, 5, 50))
}

// Ratchet : l'offset des totaux ne descend JAMAIS (un total qui recule = grillé).
func TestRatchetTotal_Monotonic(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	d := NewDemoMetrics(nil, rdb, true)
	ctx := context.Background()

	assert.InDelta(t, 100, d.ratchetTotal(ctx, "demo:test", 100), 1e-9)
	assert.InDelta(t, 250, d.ratchetTotal(ctx, "demo:test", 250), 1e-9) // monte
	assert.InDelta(t, 250, d.ratchetTotal(ctx, "demo:test", 80), 1e-9)  // cible plus basse → ne descend pas
}

// Total lectures blog = réel + offset ratcheté ; jamais sous le réel, jamais en recul.
func TestBlogReadsTotal_NeverBelowRealNorDecreasing(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	d := NewDemoMetrics(nil, rdb, true) // gitea nil → seed neutre
	ctx := context.Background()

	a := d.BlogReadsTotal(ctx, 10, demoConvergence)
	b := d.BlogReadsTotal(ctx, 10, 1) // gate plus bas, mais le ratchet ne recule pas
	assert.GreaterOrEqual(t, b, a, "le total ne recule jamais")
	assert.GreaterOrEqual(t, a, int64(10), "jamais sous le réel")
}

// Letters (flux) : jamais sous le réel.
func TestLetters_FlowNeverBelowReal(t *testing.T) {
	d := NewDemoMetrics(nil, nil, true)
	d.now = func() time.Time { return time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	assert.GreaterOrEqual(t, d.Letters(ctx, 5, demoConvergence), int64(5))
}

// Heatmap : zones LABELLISÉES (jamais vide → fin du "Unknown"), count ≥ 1 ; vide si désactivé.
func TestHeatmapPoints_LabeledGatedFloor(t *testing.T) {
	d := NewDemoMetrics(nil, nil, true)
	d.now = func() time.Time { return time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	pts := d.HeatmapPoints(ctx, demoConvergence)
	assert.NotEmpty(t, pts, "des zones synthétiques même à faible trafic (floor)")
	for _, p := range pts {
		assert.NotEmpty(t, p["element"], "chaque point porte un libellé (jamais vide)")
		assert.GreaterOrEqual(t, p["count"].(int), 1, "count >= 1")
	}

	assert.Nil(t, NewDemoMetrics(nil, nil, false).HeatmapPoints(ctx, 50), "désactivé → nil")
}

// Online : vivant mais borné, et jamais sous le réel.
func TestOnline_LiveAndBounded(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	d := NewDemoMetrics(nil, rdb, true)
	d.now = func() time.Time { return time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	v := d.Online(ctx, 1, 50)
	assert.GreaterOrEqual(t, v, int64(1), "jamais sous le réel")
	assert.LessOrEqual(t, v, int64(15), "borné (crédible)")
	// Réel 0 → jamais 0 affiché (plancher "toujours quelqu'un").
	assert.GreaterOrEqual(t, d.Online(ctx, 0, 5), int64(1), "jamais 0 quand activé")
}
