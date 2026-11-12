package dossiervivant

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QUOI : le rappel SOUPLE du tempo (tempo.go) — présent dans le message des appels faits après le
// seuil, absent avant, et il n'arrête rien.
//
// COMMENT : l'horloge du run est injectée (Options.Now) ; le test la place avant ou après le seuil
// sans attendre. La fausse passerelle de run_test.go garde le message de chaque appel.

// manualClock — une horloge que le test avance à la main.
type manualClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *manualClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *manualClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

const (
	approachMark = "the team aims to hand over the dossier within 30 s"
	lateMark     = "This run is past its 30 s target"
)

// TestTempo_RappelApresLeSeuilPasAvant — trois appels du même agent, à 5 s, 21 s et 42 s du début :
// le premier part sans rappel, le deuxième avec « on approche », le troisième avec « conclure ».
func TestTempo_RappelApresLeSeuilPasAvant(t *testing.T) {
	g := newFakeGateway(t)
	t0 := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{t: t0}
	r := &run{ai: newTestService(t, g).ai, model: testModel, lang: "fr", ph: phrases("fr"), brief: tessera,
		emit: &recorder{}, meter: newMeter(), tempo: tempo{start: t0, now: clock.now}}

	for _, at := range []time.Duration{5 * time.Second, 21 * time.Second, 42 * time.Second} {
		clock.set(t0.Add(at))
		var out readerOut
		require.NoError(t, r.ask(context.Background(), Reader, taskReader, needBlock(tessera), &out))
	}

	require.Len(t, g.prompts, 3)
	before, approaching, late := g.prompts[0], g.prompts[1], g.prompts[2]
	assert.NotContains(t, before, "<pace>", "avant le seuil : aucun rappel")
	assert.Contains(t, approaching, approachMark, "après 20 s : le rappel « on approche »")
	assert.Contains(t, approaching, "going for 21 s")
	assert.NotContains(t, approaching, lateMark)
	assert.Contains(t, late, lateMark, "après 30 s : le rappel « conclure »")
	assert.Contains(t, late, "42 s elapsed")
	// Le rappel suit le message : la consigne système, elle, ne change pas (cache de prompt).
	for i := 1; i < 3; i++ {
		assert.Equal(t, g.bodies[0]["system"], g.bodies[i]["system"])
	}
}

// TestTempo_RunEnRetardResserreSansSArreter — un run dont l'horloge dépasse la cible dès son
// premier appel : il va quand même jusqu'à la remise ; tout appel parti après le seuil porte le
// rappel, et le code ouvre moins de fiches (tightReads, tightRefs).
func TestTempo_RunEnRetardResserreSansSArreter(t *testing.T) {
	g := newFakeGateway(t)
	t0 := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{t: t0}
	// La première lecture (le début du run) est à l'heure ; toutes les suivantes, 40 s plus tard.
	first := true
	var mu sync.Mutex
	now := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		if first {
			first = false
			return clock.now()
		}
		return clock.now().Add(40 * time.Second)
	}
	s := NewService(newTestService(t, g).ai, memSources{}, Options{Model: testModel, SchemaModel: testSchemaModel, Beat: -1, Now: now})

	rec := &recorder{}
	require.NoError(t, s.Run(context.Background(), tessera, "fr", rec))
	assertDelivered(t, rec)

	g.mu.Lock()
	prompts, tasks := append([]string(nil), g.prompts...), append([]string(nil), g.tasks...)
	g.mu.Unlock()
	for i, p := range prompts {
		assert.Contains(t, p, lateMark, "appel %d (%s) parti après la cible sans rappel", i, tasks[i])
	}
	verdicts, lessons := 0, 0
	for _, task := range tasks {
		switch task {
		case "verdict":
			verdicts++
		case "lesson":
			lessons++
		}
	}
	assert.Equal(t, tightReads, verdicts, "en retard, le Chercheur ouvre moins de fiches")
	assert.LessOrEqual(t, lessons, tightRefs, "en retard, l'Architecte ouvre moins de fiches")
}

// TestTempo_RunDansLesTempsSansRappel — dans les temps, aucun message ne porte de rappel.
func TestTempo_RunDansLesTempsSansRappel(t *testing.T) {
	g := newFakeGateway(t)
	_, err := runOnce(t, newTestService(t, g))
	require.NoError(t, err)
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, p := range g.prompts {
		assert.False(t, strings.Contains(p, "<pace>"), "appel %d (%s) : rappel alors que le run est dans les temps", i, g.tasks[i])
	}
}
