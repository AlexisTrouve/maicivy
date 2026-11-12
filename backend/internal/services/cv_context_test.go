package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// fakeCVGen : compte les générations et peut les retenir (gate) pour observer l'état « pending ».
type fakeCVGen struct {
	calls  atomic.Int32
	gate   chan struct{} // nil = répond tout de suite
	err    error
	offers []string
	langs  []string
	mu     sync.Mutex
}

func (f *fakeCVGen) GenerateDynamicCV(_ context.Context, offer, lang string) (*AdaptiveCVResponse, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.offers = append(f.offers, offer)
	f.langs = append(f.langs, lang)
	f.mu.Unlock()
	if f.gate != nil {
		<-f.gate
	}
	if f.err != nil {
		return nil, f.err
	}
	return &AdaptiveCVResponse{Theme: config.CVTheme{ID: "dynamic", Name: "Lead Go"}, Summary: "adapté"}, nil
}

type fakeBudget struct {
	allow    bool
	consumed atomic.Int32
}

func (b *fakeBudget) Allow(context.Context, string, string) bool { return b.allow }
func (b *fakeBudget) Consume(context.Context, string, string) error {
	b.consumed.Add(1)
	return nil
}

// newCtxSvc : service sur miniredis ; `wg` est décrémenté à la fin de chaque génération en fond.
func newCtxSvc(t *testing.T, gen CVContextGenerator, budget *fakeBudget) (*CVContextService, *miniredis.Miniredis, *sync.WaitGroup) {
	t.Helper()
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s := NewCVContextService(r, gen, budget)
	wg := &sync.WaitGroup{}
	s.done = wg.Done
	return s, mr, wg
}

var visitor = CVVisitor{SessionID: "sess-1.sig", IP: "1.2.3.4"}

const need = "Je cherche un développeur Go senior pour une plateforme d'agents IA en production."

func TestCVContext_SansBesoin_None(t *testing.T) {
	gen := &fakeCVGen{}
	s, _, _ := newCtxSvc(t, gen, &fakeBudget{allow: true})

	view, err := s.Ensure(context.Background(), visitor, "fr")
	require.NoError(t, err)
	assert.Equal(t, CVContextNone, view.Status)
	assert.Zero(t, gen.calls.Load(), "aucun besoin → aucune génération")
}

func TestCVContext_BesoinLanceUneSeuleGeneration(t *testing.T) {
	gen := &fakeCVGen{gate: make(chan struct{})}
	budget := &fakeBudget{allow: true}
	s, _, wg := newCtxSvc(t, gen, budget)
	ctx := context.Background()

	wg.Add(1)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))

	// Pendant la génération : relectures, même besoin re-noté, lectures concurrentes → toujours une.
	var readers sync.WaitGroup
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			v, err := s.Ensure(ctx, visitor, "fr")
			assert.NoError(t, err)
			assert.Equal(t, CVContextPending, v.Status)
		}()
	}
	readers.Wait()
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, "  "+need+"  ", "fr"))

	close(gen.gate)
	wg.Wait()

	assert.EqualValues(t, 1, gen.calls.Load(), "single-flight : une génération par session et par besoin")
	assert.EqualValues(t, 1, budget.consumed.Load(), "la génération réussie est comptée dans le budget")
	assert.True(t, strings.HasPrefix(gen.offers[0], "Besoin exprimé"), "le besoin est présenté comme tel, jamais comme une URL")

	view, err := s.Ensure(ctx, visitor, "fr")
	require.NoError(t, err)
	assert.Equal(t, CVContextReady, view.Status)
	assert.Equal(t, NeedSourceChat, view.Source)
	assert.Equal(t, need, view.Need)
	require.NotNil(t, view.CV)
	assert.Equal(t, "Lead Go", view.CV.Theme.Name)
	assert.EqualValues(t, 1, gen.calls.Load(), "lire un CV prêt ne relance rien")
}

func TestCVContext_NouveauBesoinRelance(t *testing.T) {
	gen := &fakeCVGen{}
	s, _, wg := newCtxSvc(t, gen, &fakeBudget{allow: true})
	ctx := context.Background()

	wg.Add(2)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceDossier, "Refonte d'un moteur C++ temps réel pour un studio de jeu.", "fr"))
	wg.Wait()

	assert.EqualValues(t, 2, gen.calls.Load(), "un besoin DIFFÉRENT est une nouvelle génération")
	view, _ := s.Ensure(ctx, visitor, "fr")
	assert.Equal(t, NeedSourceDossier, view.Source, "le dernier besoin remplace le précédent")
}

func TestCVContext_AutreLangue_UneGenerationParLangue(t *testing.T) {
	gen := &fakeCVGen{}
	s, _, wg := newCtxSvc(t, gen, &fakeBudget{allow: true})
	ctx := context.Background()

	wg.Add(1)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))
	wg.Wait()

	// /de/cv : l'allemand n'est pas une langue de CV adapté → anglais (TailoredLanguage).
	wg.Add(1)
	v, err := s.Ensure(ctx, visitor, "de")
	require.NoError(t, err)
	assert.Equal(t, "en", v.Lang)
	wg.Wait()
	_, _ = s.Ensure(ctx, visitor, "it") // it → en aussi : déjà généré
	assert.EqualValues(t, 2, gen.calls.Load())
	assert.Equal(t, []string{"fr", "en"}, gen.langs)
}

func TestCVContext_TTL24h(t *testing.T) {
	gen := &fakeCVGen{}
	s, mr, wg := newCtxSvc(t, gen, &fakeBudget{allow: true})
	ctx := context.Background()

	wg.Add(1)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))
	wg.Wait()

	assert.Equal(t, 24*time.Hour, mr.TTL(needKey(visitor.SessionID)), "le besoin vit 24 h")
	for _, k := range mr.Keys() {
		assert.Greater(t, mr.TTL(k), time.Duration(0), "aucune clé sans expiration : %s", k)
	}

	mr.FastForward(24*time.Hour + time.Second)
	view, err := s.Ensure(ctx, visitor, "fr")
	require.NoError(t, err)
	assert.Equal(t, CVContextNone, view.Status, "après 24 h le contexte a disparu")
	assert.Empty(t, mr.Keys())
}

func TestCVContext_BudgetEpuise_Limited_SansGeneration(t *testing.T) {
	gen := &fakeCVGen{}
	s, mr, _ := newCtxSvc(t, gen, &fakeBudget{allow: false})

	require.NoError(t, s.RecordNeed(context.Background(), visitor, NeedSourceChat, need, "fr"))
	view, err := s.Ensure(context.Background(), visitor, "fr")
	require.NoError(t, err)
	assert.Equal(t, CVContextLimited, view.Status)
	assert.Zero(t, gen.calls.Load(), "plafonds IA visiteurs atteints → aucune génération")
	assert.Len(t, mr.Keys(), 1, "seul le besoin est gardé : un refus de budget n'écrit pas d'état")
}

func TestCVContext_EchecGeneration_FailedBorne(t *testing.T) {
	gen := &fakeCVGen{err: errors.New("upstream 529")}
	budget := &fakeBudget{allow: true}
	s, mr, wg := newCtxSvc(t, gen, budget)
	ctx := context.Background()

	wg.Add(1)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))
	wg.Wait()

	view, _ := s.Ensure(ctx, visitor, "fr")
	assert.Equal(t, CVContextFailed, view.Status)
	assert.Zero(t, budget.consumed.Load(), "un échec n'est pas compté")

	// Après le délai d'échec, une lecture relance (panne passagère de l'IA).
	mr.FastForward(cvContextFailedTTL + time.Second)
	wg.Add(1)
	view, _ = s.Ensure(ctx, visitor, "fr")
	assert.Equal(t, CVContextPending, view.Status)
	wg.Wait()
	assert.EqualValues(t, 2, gen.calls.Load())
}

func TestCVContext_BesoinTropCourt_Refuse(t *testing.T) {
	gen := &fakeCVGen{}
	s, mr, _ := newCtxSvc(t, gen, &fakeBudget{allow: true})
	err := s.RecordNeed(context.Background(), visitor, NeedSourceChat, "   go  ", "fr")
	assert.ErrorIs(t, err, ErrNeedLength)
	assert.Empty(t, mr.Keys())
}

func TestCVContext_BesoinBorneEnRunes(t *testing.T) {
	long := strings.Repeat("é", cvContextMaxNeedRune+50)
	assert.Equal(t, cvContextMaxNeedRune, len([]rune(normalizeNeed(long))))
}

func TestCVContext_AdaptedCV(t *testing.T) {
	gen := &fakeCVGen{gate: make(chan struct{})}
	s, _, wg := newCtxSvc(t, gen, &fakeBudget{allow: true})
	ctx := context.Background()

	cv, _, err := s.AdaptedCV(ctx, visitor.SessionID, "fr")
	require.NoError(t, err)
	assert.Nil(t, cv, "pas de besoin → pas de CV adapté")

	wg.Add(1)
	require.NoError(t, s.RecordNeed(ctx, visitor, NeedSourceChat, need, "fr"))
	cv, _, _ = s.AdaptedCV(ctx, visitor.SessionID, "fr")
	assert.Nil(t, cv, "en préparation → pas encore de CV")
	close(gen.gate)
	wg.Wait()

	cv, lang, err := s.AdaptedCV(ctx, visitor.SessionID, "fr")
	require.NoError(t, err)
	require.NotNil(t, cv)
	assert.Equal(t, "fr", lang)
}
