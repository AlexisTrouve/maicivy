package dossiervivant

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// QUOI : ce qu'un run n'a PLUS le droit de faire — s'arrêter sur un budget de tokens ou sur une
// durée — et ce que chaque appel n'a toujours pas le droit de faire : partir avec une entrée de
// 100 000 tokens ou plus (au-delà, Haiku 5.5 change de tarif).
//
// POURQUOI au niveau du run entier : les deux arrêts durs vivaient dans Service.Run (budget réservé
// dans mind.go, context.WithTimeout dans orchestrator.go) ; c'est le run complet qui doit finir sur
// run.completed, quelle que soit sa consommation.
//
// COMMENT : la fausse passerelle de run_test.go ; un Completer intercalé observe le contexte que
// chaque appel reçoit ; une source dont les fiches sont démesurées force le code à faire tenir
// chaque appel sous le plafond SANS arrêter le run.

// deadlineProbe — intercalé entre le run et le vrai client IA : relève si un appel a reçu un
// contexte borné dans le temps PAR LE RUN (le client IA pose ensuite son propre délai d'appel).
type deadlineProbe struct {
	inner Completer
	mu    sync.Mutex
	bound int // appels partis sous un contexte à échéance
}

func (p *deadlineProbe) Complete(ctx context.Context, req services.CompletionRequest) (*services.Completion, error) {
	if _, ok := ctx.Deadline(); ok {
		p.mu.Lock()
		p.bound++
		p.mu.Unlock()
	}
	return p.inner.Complete(ctx, req)
}

// assertDelivered — le run est allé jusqu'à la remise : run.completed en dernier, aucun run.failed,
// les cinq sections écrites.
func assertDelivered(t *testing.T, rec *recorder) {
	t.Helper()
	events := rec.all()
	require.NotEmpty(t, events)
	assert.Equal(t, RunCompleted{}, events[len(events)-1], "le run doit finir sur run.completed")
	done := map[string]bool{}
	for _, e := range events {
		_, failed := e.(RunFailed)
		assert.False(t, failed, "aucun run.failed : %v", e)
		if s, ok := e.(DossierSection); ok && s.Status == "done" {
			done[s.Section] = true
		}
	}
	for _, id := range []string{SecNeed, SecArchitecture, SecProofs, SecContribution, SecNext} {
		assert.True(t, done[id], "section %s jamais terminée", id)
	}
}

// TestRun_LongNestPasArrete — un run qui consomme plusieurs fois l'ancien plafond de 90 000 tokens
// (20 000 d'entrée déclarés par appel, une vingtaine d'appels) n'est pas arrêté, et aucun appel ne
// part sous une échéance posée par le run : rien ne coupe un run en cours de route.
func TestRun_LongNestPasArrete(t *testing.T) {
	g := newFakeGateway(t)
	g.inputTokens = 20_000
	probe := &deadlineProbe{inner: newTestService(t, g).ai}
	s := NewService(probe, memSources{}, Options{Model: testModel, SchemaModel: testSchemaModel, Beat: -1})

	rec := &recorder{}
	require.NoError(t, s.Run(context.Background(), tessera, "fr", rec))
	assertDelivered(t, rec)

	in, _, _ := meterOf(rec).totals()
	assert.Greater(t, in, 300_000, "le run a bien consommé plus de trois fois l'ancien plafond")
	assert.Zero(t, probe.bound, "aucun appel ne doit recevoir d'échéance posée par le run")
}

// meterOf — le total d'entrée d'un run, relu sur ses événements `usage` (cumuls par agent).
func meterOf(rec *recorder) *meter {
	m := newMeter()
	last := map[AgentID]Usage{}
	for _, e := range rec.all() {
		if u, ok := e.(Usage); ok {
			last[u.Agent] = u
		}
	}
	for a, u := range last {
		m.add(a, u.InputTokens, u.OutputTokens, u.CostUsd)
	}
	return m
}

// hugeSources — des fiches démesurées : `melodicode` pèse à elle seule plus de 100 000 tokens
// estimés (aucun appel ne peut la porter), chacune des autres environ 40 000 (une tient dans un
// verdict, quatre ne tiennent pas ensemble dans l'appel du Rédacteur qui les relit toutes).
type hugeSources struct{ memSources }

const (
	hugeAlone = 330_000 // octets ≈ 110 000 tokens estimés
	hugeEach  = 120_000 // octets ≈ 40 000 tokens estimés
)

func (h hugeSources) Project(ctx context.Context, id, lang string) (*services.MPFProject, error) {
	p, _ := h.memSources.Project(ctx, id, lang)
	size := hugeEach
	if id == "melodicode" {
		size = hugeAlone
	}
	p.Description.Portfolio = memStory(id) + strings.Repeat("x", size)
	return p, nil
}

// TestRun_AucunAppelAuDelaDe100k — avec des fiches démesurées, le run va jusqu'à la remise ET
// aucun appel n'est émis avec une entrée estimée ≥ 100 000 tokens. Le code fait tenir chaque appel
// (fiche trop longue non lue, preuves relues par le Rédacteur limitées à ce qui tient) au lieu
// d'arrêter le run.
func TestRun_AucunAppelAuDelaDe100k(t *testing.T) {
	g := newFakeGateway(t)
	s := NewService(newTestService(t, g).ai, hugeSources{}, Options{Model: testModel, SchemaModel: testSchemaModel, Beat: -1})

	rec := &recorder{}
	require.NoError(t, s.Run(context.Background(), tessera, "fr", rec))
	assertDelivered(t, rec)

	g.mu.Lock()
	defer g.mu.Unlock()
	require.NotEmpty(t, g.bodies)
	for i, b := range g.bodies {
		system := b["system"].([]any)[0].(map[string]any)["text"].(string)
		in := estimateTokens(system, g.prompts[i])
		assert.Less(t, in, 100_000, "appel %d (%s) parti avec %d tokens d'entrée estimés", i, g.tasks[i], in)
	}
	// Le cas limite est bien exercé : la fiche démesurée n'est jamais partie dans un appel.
	for i, p := range g.prompts {
		assert.NotContains(t, p, "fin-du-recit-melodicode", "appel %d (%s) porte la fiche de 110 000 tokens", i, g.tasks[i])
	}
}
