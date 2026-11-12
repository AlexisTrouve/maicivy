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

	"maicivy/internal/models"
)

// blogSimule — un faux maiProFiles côté blog : on pilote la panne, les articles absents, le titre.
type blogSimule struct {
	mu      sync.Mutex
	enPanne bool
	absents map[string]bool
	titre   string
	appels  atomic.Int32
}

func (s *blogSimule) GetBlogPosts(_ context.Context, page, perPage int, _ string) (*models.BlogPostListResponse, error) {
	s.appels.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enPanne {
		return nil, errPanne
	}
	return &models.BlogPostListResponse{
		Posts: []models.BlogPost{{Slug: "premier", Title: s.titre, ContentHTML: "<p>x</p>"}},
		Total: 1, Page: page, PerPage: perPage, TotalPages: 1,
	}, nil
}

func (s *blogSimule) GetBlogPost(_ context.Context, slug, _ string) (*models.BlogPost, error) {
	s.appels.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enPanne {
		return nil, errPanne
	}
	if s.absents[slug] {
		return nil, ErrBlogPostAbsent
	}
	return &models.BlogPost{Slug: slug, Title: s.titre, ContentHTML: "<h2>corps</h2>"}, nil
}

func (s *blogSimule) regler(f func(*blogSimule)) { s.mu.Lock(); f(s); s.mu.Unlock() }

func blogDeTest(t *testing.T) (*BlogCache, *blogSimule, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	src := &blogSimule{titre: "v1", absents: map[string]bool{}}
	return newBlogCache(src, redis.NewClient(&redis.Options{Addr: mr.Addr()})), src, mr
}

// LE cas de la panne : maiProFiles tombe, l'article déjà lu reste servi.
func TestBlogCache_PanneAmontServiLePerime(t *testing.T) {
	c, src, mr := blogDeTest(t)
	ctx := context.Background()

	_, err := c.Post(ctx, "premier", "fr")
	require.NoError(t, err)
	vieillir(t, mr, cleArticle("fr", "premier"), 2*blogFraicheur)
	src.regler(func(s *blogSimule) { s.enPanne = true })

	article, err := c.Post(ctx, "premier", "fr")
	require.NoError(t, err, "maiProFiles en panne : l'article en cache doit être servi")
	require.Equal(t, "premier", article.Slug)
	require.Contains(t, article.ContentHTML, "<h2>", "le HTML rendu doit survivre au cache")
}

// Les clés portent un TTL : un robot qui balaye des slugs ne doit pas remplir Redis pour toujours.
func TestBlogCache_ClesAvecTTL(t *testing.T) {
	c, _, mr := blogDeTest(t)
	_, err := c.Posts(context.Background(), 1, 10, "fr")
	require.NoError(t, err)
	ttl := mr.TTL(cleCatalogue("fr"))
	require.Greater(t, ttl, time.Duration(0), "une clé de blog sans TTL ne disparaîtrait jamais")
}

// Un slug inexistant n'est JAMAIS mis en cache, et remonte ErrBlogPostAbsent (le handler en fait un 404).
func TestBlogCache_SlugInexistantNonCache(t *testing.T) {
	c, src, mr := blogDeTest(t)
	src.regler(func(s *blogSimule) { s.absents["fantome"] = true })

	_, err := c.Post(context.Background(), "fantome", "fr")
	require.ErrorIs(t, err, ErrBlogPostAbsent)
	require.False(t, mr.Exists(cleArticle("fr", "fantome")), "un échec ne doit pas être stocké")
}

// Un article supprimé ou dépublié sort du cache au rafraîchissement.
func TestBlogCache_ArticleSupprimeSortDuCache(t *testing.T) {
	c, src, mr := blogDeTest(t)
	ctx := context.Background()

	_, err := c.Post(ctx, "premier", "fr")
	require.NoError(t, err)
	vieillir(t, mr, cleArticle("fr", "premier"), 2*blogFraicheur)
	src.regler(func(s *blogSimule) { s.absents["premier"] = true })

	_, err = c.Post(ctx, "premier", "fr") // sert le périmé, déclenche le rafraîchissement
	require.NoError(t, err)
	require.Eventually(t, func() bool { return !mr.Exists(cleArticle("fr", "premier")) },
		2*time.Second, 20*time.Millisecond, "l'article supprimé doit quitter le cache")
}

// Après une écriture de l'owner, la version suivante est lue en amont tout de suite.
func TestBlogCache_InvalidateVideTout(t *testing.T) {
	c, src, mr := blogDeTest(t)
	ctx := context.Background()

	_, _ = c.Posts(ctx, 1, 10, "fr")
	_, _ = c.Posts(ctx, 1, 50, "en")
	_, _ = c.Post(ctx, "premier", "fr")
	require.NoError(t, mr.Set("autre:cle", "intacte"))
	src.regler(func(s *blogSimule) { s.titre = "v2" })

	c.Invalidate(ctx)

	require.False(t, mr.Exists(cleCatalogue("fr")))
	require.False(t, mr.Exists(cleCatalogue("en")))
	require.False(t, mr.Exists(cleArticle("fr", "premier")))
	require.True(t, mr.Exists("autre:cle"), "l'invalidation ne touche que les clés du blog")

	liste, err := c.Posts(ctx, 1, 10, "fr")
	require.NoError(t, err)
	require.Equal(t, "v2", liste.Posts[0].Title, "après invalidation, la nouvelle version est servie immédiatement")
}

// Frais = aucun appel à maiProFiles.
func TestBlogCache_FraisNAppellePasLaSource(t *testing.T) {
	c, src, _ := blogDeTest(t)
	for i := 0; i < 5; i++ {
		_, err := c.Posts(context.Background(), 1, 10, "fr")
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, src.appels.Load())
}

// catalogueSimule : un faux maiProFiles qui pagine un vrai catalogue (n articles, thèmes alternés).
type catalogueSimule struct {
	blogSimule
	n int
}

func (s *catalogueSimule) GetBlogPosts(_ context.Context, page, perPage int, _ string) (*models.BlogPostListResponse, error) {
	s.appels.Add(1)
	var tous []models.BlogPost
	for i := 0; i < s.n; i++ {
		theme := "tech"
		if i%3 == 0 {
			theme = "games"
		}
		tous = append(tous, models.BlogPost{Slug: fmt.Sprintf("a%03d", i), Tags: []string{"auto", theme}})
	}
	r := paginer(tous, page, perPage)
	return &r, nil
}

// Régression du 2026-10-11 : le blog n'affichait que la 1re page de 50 (88 articles sur 138
// invisibles). La liste se calcule désormais sur le catalogue ENTIER, lu page par page en amont.
func TestBlogCache_ListeCouvreToutLeCatalogue(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &catalogueSimule{n: 138}
	c := newBlogCache(src, redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ctx := context.Background()

	p1, err := c.Liste(ctx, "fr", "", 1, 12)
	require.NoError(t, err)
	require.EqualValues(t, 138, p1.Total)
	require.Equal(t, 12, p1.TotalPages)
	require.Len(t, p1.Posts, 12)
	require.Equal(t, map[string]int{"games": 46, "tech": 92}, p1.Themes, "compteurs sur TOUT le catalogue")

	der, err := c.Liste(ctx, "fr", "", 12, 12)
	require.NoError(t, err)
	require.Len(t, der.Posts, 6, "la dernière page porte les plus anciens articles")
	require.Equal(t, "a137", der.Posts[5].Slug)

	games, err := c.Liste(ctx, "fr", "games", 4, 12)
	require.NoError(t, err)
	require.EqualValues(t, 46, games.Total)
	require.Equal(t, 138, games.TotalTous, "le compteur « Tous » ignore le filtre")
	require.Len(t, games.Posts, 10)
	for _, p := range games.Posts {
		require.Contains(t, []string(p.Tags), "games")
	}

	hors, err := c.Liste(ctx, "fr", "", 99, 12)
	require.NoError(t, err)
	require.Empty(t, hors.Posts, "au-delà de la dernière page : vide, jamais une erreur")

	require.EqualValues(t, 2, src.appels.Load(), "2 pages de 100 en amont, puis tout depuis le cache")
}
