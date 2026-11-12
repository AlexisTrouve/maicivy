package services

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// sourceSimulee — un faux maiProFiles dont on pilote la santé : réponses, panne, projet retiré.
type sourceSimulee struct {
	mu       sync.Mutex
	enPanne  bool
	absents  map[string]bool
	appels   atomic.Int32
	nomLivre string
	nomPage  string
}

var errPanne = errors.New("maiprofiles fetch: context deadline exceeded")

func (s *sourceSimulee) GetPortfolioBook(_ context.Context, lang string) (*MPFBook, error) {
	s.appels.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enPanne {
		return nil, errPanne
	}
	return &MPFBook{Lang: lang, Title: s.nomLivre, Projects: []MPFBookProject{{ID: "theomen", Name: "Theomen"}}}, nil
}

func (s *sourceSimulee) GetPortfolioPage(_ context.Context, id, lang string) (*MPFPortfolioPage, error) {
	s.appels.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enPanne {
		return nil, errPanne
	}
	if s.absents[id] {
		return nil, ErrPortfolioAbsent
	}
	p := &MPFPortfolioPage{DescriptionPortfolioHTML: "<h1>" + s.nomPage + "</h1>"}
	p.ID, p.Name = id, s.nomPage
	return p, nil
}

func (s *sourceSimulee) regler(f func(*sourceSimulee)) { s.mu.Lock(); f(s); s.mu.Unlock() }

// cacheDeTest monte le cache sur un miniredis et une source simulée.
func cacheDeTest(t *testing.T) (*PortfolioCache, *sourceSimulee, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	src := &sourceSimulee{nomLivre: "v1", nomPage: "v1", absents: map[string]bool{}}
	return newPortfolioCache(src, redis.NewClient(&redis.Options{Addr: mr.Addr()})), src, mr
}

// vieillir antidate le fetchedAt d'une clé pour simuler un cache périmé.
func vieillir(t *testing.T, mr *miniredis.Miniredis, cle string, age time.Duration) {
	t.Helper()
	brut, err := mr.Get(cle)
	require.NoError(t, err)
	var blob swrBlob
	require.NoError(t, json.Unmarshal([]byte(brut), &blob))
	blob.FetchedAt = time.Now().Add(-age)
	b, _ := json.Marshal(blob)
	require.NoError(t, mr.Set(cle, string(b)))
}

// LE cas de la panne du 2026-09-27 : maiProFiles tombe, le portfolio doit rester servi.
func TestPortfolioCache_PanneAmontServiLePerime(t *testing.T) {
	c, src, mr := cacheDeTest(t)
	ctx := context.Background()

	_, err := c.Page(ctx, "theomen", "fr")
	require.NoError(t, err)
	vieillir(t, mr, clePage("fr", "theomen"), 2*portfolioFraicheur)
	src.regler(func(s *sourceSimulee) { s.enPanne = true })

	page, err := c.Page(ctx, "theomen", "fr")
	require.NoError(t, err, "maiProFiles en panne : la version en cache doit être servie, pas une erreur")
	require.Equal(t, "theomen", page.ID)
	require.Contains(t, page.DescriptionPortfolioHTML, "<h1>", "le HTML rendu doit survivre au cache")

	livre, err := c.Book(ctx, "fr")
	require.Error(t, err, "livre jamais mis en cache + panne : l'erreur remonte, rien à servir")
	require.Nil(t, livre)
}

// Un cache périmé est servi IMMÉDIATEMENT, et la nouvelle version arrive en fond.
func TestPortfolioCache_PerimeRafraichiEnFond(t *testing.T) {
	c, src, mr := cacheDeTest(t)
	ctx := context.Background()

	_, err := c.Book(ctx, "fr")
	require.NoError(t, err)
	vieillir(t, mr, cleLivre("fr"), 2*portfolioFraicheur)
	src.regler(func(s *sourceSimulee) { s.nomLivre = "v2" })

	livre, err := c.Book(ctx, "fr")
	require.NoError(t, err)
	require.Equal(t, "v1", livre.Title, "le périmé est rendu sans attendre la source")

	require.Eventually(t, func() bool {
		l, err := c.Book(ctx, "fr")
		return err == nil && l.Title == "v2"
	}, 2*time.Second, 20*time.Millisecond, "le rafraîchissement en fond doit publier la nouvelle version")
}

// Frais = aucun appel à maiProFiles.
func TestPortfolioCache_FraisNAppellePasLaSource(t *testing.T) {
	c, src, _ := cacheDeTest(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := c.Page(ctx, "theomen", "fr")
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, src.appels.Load(), "une seule lecture amont pour cinq visites")
}

// Un projet retiré de la curation ne doit pas rester servi indéfiniment depuis le cache.
func TestPortfolioCache_ProjetRetireSortDuCache(t *testing.T) {
	c, src, mr := cacheDeTest(t)
	ctx := context.Background()

	_, err := c.Page(ctx, "theomen", "fr")
	require.NoError(t, err)
	vieillir(t, mr, clePage("fr", "theomen"), 2*portfolioFraicheur)
	src.regler(func(s *sourceSimulee) { s.absents["theomen"] = true })

	_, err = c.Page(ctx, "theomen", "fr") // sert le périmé, déclenche le rafraîchissement
	require.NoError(t, err)

	require.Eventually(t, func() bool { return !mr.Exists(clePage("fr", "theomen")) },
		2*time.Second, 20*time.Millisecond, "le projet retiré doit être supprimé du cache")
	_, err = c.Page(ctx, "theomen", "fr")
	require.ErrorIs(t, err, ErrPortfolioAbsent, "une fois retiré, c'est un 404 légitime")
}

// de/it partagent l'édition anglaise : une seule clé, pas trois copies.
func TestPortfolioCache_LangueNormaliseeAvantLaCle(t *testing.T) {
	c, src, _ := cacheDeTest(t)
	ctx := context.Background()

	for _, l := range []string{"en", "de", "it"} {
		_, err := c.Book(ctx, l)
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, src.appels.Load())
}

// Le préchauffage remplit le livre ET les pages qu'il liste.
func TestPortfolioCache_WarmRemplitLivresEtPages(t *testing.T) {
	c, src, mr := cacheDeTest(t)
	c.Warm(context.Background())

	for lang := range langsLivre {
		require.True(t, mr.Exists(cleLivre(lang)), "livre %s absent après préchauffage", lang)
		require.True(t, mr.Exists(clePage(lang, "theomen")), "page %s absente après préchauffage", lang)
	}
	avant := src.appels.Load()
	_, err := c.Page(context.Background(), "theomen", "zh")
	require.NoError(t, err)
	require.Equal(t, avant, src.appels.Load(), "après préchauffage, la première visite ne touche pas maiProFiles")
}
