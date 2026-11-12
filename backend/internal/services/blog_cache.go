package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"maicivy/internal/models"
)

// ============================================================================
// BlogCache — le blog reste servi quand maiProFiles est lent ou tombé.
// ----------------------------------------------------------------------------
// QUOI : cache Redis stale-while-revalidate (swrStore) devant les deux lectures publiques du blog :
// la liste paginée des articles et un article par slug. Les écritures (création, mise à jour,
// publication, suppression) invalident le cache.
//
// POURQUOI : le blog lit maiProFiles comme le portfolio, et tombait avec lui lors de la panne du
// 2026-09-27. Seul un cache mémoire de 5 min le protégeait : expiré, chaque visite retombait sur
// maiProFiles. Et toute erreur de lecture d'un article répondait « 404 », panne comprise.
//
// COMMENT :
//   - clés blog:v1:catalogue:<lang> (tous les articles, sans contenu) et blog:v1:post:<lang>:<slug> ;
//   - TTL de 7 jours : contrairement au portfolio, ces clés dérivent d'entrées libres (slug, page,
//     taille de page choisis par le visiteur) — sans TTL, un robot qui balaye des slugs inventés
//     remplirait Redis. Une clé visitée est réécrite à chaque rafraîchissement et ne meurt pas ;
//   - seuls les succès sont stockés : un slug inexistant n'est jamais mis en cache ;
//   - un article supprimé ou dépublié (ErrBlogPostAbsent au rafraîchissement) sort du cache ;
//   - Invalidate() après toute écriture : l'owner doit voir sa publication tout de suite, pas
//     après le délai de fraîcheur.
// ============================================================================

const (
	// blogCachePrefix — versionné : un changement de forme du blob se fait par bump de clé.
	blogCachePrefix = "blog:v1"
	// blogFraicheur — un article publié par le pipeline apparaît en ~5 min dans la liste même sans
	// invalidation (le pipeline WanMira écrit dans maiProFiles sans passer par maicivy).
	blogFraicheur = 5 * time.Minute
	// blogTTL — borne la durée de vie des clés jamais revisitées (cf. en-tête).
	blogTTL = 7 * 24 * time.Hour
)

// blogSource — ce que le cache sait lire en amont (interface pour les tests).
type blogSource interface {
	GetBlogPosts(ctx context.Context, page, perPage int, lang string) (*models.BlogPostListResponse, error)
	GetBlogPost(ctx context.Context, postSlug string, lang string) (*models.BlogPost, error)
}

// BlogCache — voir l'en-tête du fichier.
type BlogCache struct {
	source blogSource
	store  *swrStore
}

// NewBlogCache — rdb peut être nil : le cache devient un passe-plat.
func NewBlogCache(source *MaiProFilesClient, rdb *redis.Client) *BlogCache {
	return newBlogCache(source, rdb)
}

func newBlogCache(source blogSource, rdb *redis.Client) *BlogCache {
	return &BlogCache{
		source: source,
		store: &swrStore{
			redis:     rdb,
			nom:       "blog",
			fraicheur: blogFraicheur,
			ttl:       blogTTL,
			estAbsent: func(err error) bool { return errors.Is(err, ErrBlogPostAbsent) },
			limite:    limiteFondMaiProFiles,
		},
	}
}

func cleCatalogue(lang string) string {
	return fmt.Sprintf("%s:catalogue:%s", blogCachePrefix, lang)
}
func cleArticle(lang, slug string) string {
	return fmt.Sprintf("%s:post:%s:%s", blogCachePrefix, lang, slug)
}

// ThemesBlog — les thèmes curated du filtre du blog. Liste FIXE : on ne filtre pas sur les ~50
// tags auto-générés (bruyants) mais sur ces catégories posées par le pipeline WanMira.
// COUPLAGE : chaque clé correspond au `theme` d'un profil WanMira (devblog.yaml → "tech",
// drifterra.yaml et futurs jeux → "games") ; le publisher pousse ce thème dans les tags.
var ThemesBlog = []string{"tech", "games"}

// blogCataloguePage — taille de page demandée à maiProFiles pour lire le catalogue (il borne à 100).
const blogCataloguePage = 100

// BlogListe — une page de la liste, et le nombre d'articles par thème sur TOUT le catalogue.
type BlogListe struct {
	models.BlogPostListResponse
	Themes map[string]int `json:"themes"`
	// TotalTous : nombre d'articles de TOUT le blog (compteur « Tous »), quel que soit le filtre.
	TotalTous int `json:"total_all"`
}

// Catalogue — tous les articles publiés (sans leur contenu), du plus récent au plus ancien.
//
// POURQUOI un catalogue entier plutôt qu'une clé par page : le filtre par thème et ses compteurs
// portent sur l'ENSEMBLE des articles, que maiProFiles ne sait pas filtrer. Mesuré le 2026-10-11 :
// le front chargeait une seule page de 50 et filtrait en mémoire — 88 articles sur 138 étaient
// invisibles, compteurs de thèmes compris. Le catalogue sans contenu pèse quelques dizaines de kB
// (maiProFiles exclut content/content_html de la liste) : une clé par langue suffit, et un visiteur
// qui invente des numéros de page ou des thèmes ne crée plus aucune clé Redis.
func (b *BlogCache) Catalogue(ctx context.Context, lang string) ([]models.BlogPost, error) {
	var posts []models.BlogPost
	err := b.store.lire(ctx, cleCatalogue(lang), &posts, func(c context.Context) (interface{}, error) {
		var tous []models.BlogPost
		for page := 1; ; page++ {
			r, err := b.source.GetBlogPosts(c, page, blogCataloguePage, lang)
			if err != nil {
				return nil, err
			}
			tous = append(tous, r.Posts...)
			// Arrêt sur le nombre de pages annoncé, et garde-fou sur une page vide (un total
			// incohérent ne doit pas faire boucler).
			if page >= r.TotalPages || len(r.Posts) == 0 {
				break
			}
		}
		return tous, nil
	})
	if err != nil {
		return nil, err
	}
	return posts, nil
}

// Liste — une page du catalogue, filtrée par thème ("" = tous), et les compteurs de thèmes.
// Une page au-delà de la dernière rend une liste vide (le front en fait un 404).
func (b *BlogCache) Liste(ctx context.Context, lang, theme string, page, perPage int) (*BlogListe, error) {
	tous, err := b.Catalogue(ctx, lang)
	if err != nil {
		return nil, err
	}
	themes := make(map[string]int, len(ThemesBlog))
	for _, t := range ThemesBlog {
		themes[t] = 0
	}
	filtres := make([]models.BlogPost, 0, len(tous))
	for _, p := range tous {
		for _, tag := range p.Tags {
			if _, connu := themes[tag]; connu {
				themes[tag]++
			}
		}
		if theme == "" || contientTag(p.Tags, theme) {
			filtres = append(filtres, p)
		}
	}
	return &BlogListe{BlogPostListResponse: paginer(filtres, page, perPage), Themes: themes, TotalTous: len(tous)}, nil
}

// Posts — une page du catalogue complet, sans filtre (flux RSS, newsletter, liste publique).
func (b *BlogCache) Posts(ctx context.Context, page, perPage int, lang string) (*models.BlogPostListResponse, error) {
	l, err := b.Liste(ctx, lang, "", page, perPage)
	if err != nil {
		return nil, err
	}
	return &l.BlogPostListResponse, nil
}

func contientTag(tags []string, cible string) bool {
	for _, t := range tags {
		if t == cible {
			return true
		}
	}
	return false
}

// paginer découpe une liste déjà triée. page et perPage sont supposés ≥ 1 (validés par le handler).
func paginer(posts []models.BlogPost, page, perPage int) models.BlogPostListResponse {
	total := len(posts)
	pages := (total + perPage - 1) / perPage
	debut := (page - 1) * perPage
	if debut > total {
		debut = total
	}
	fin := debut + perPage
	if fin > total {
		fin = total
	}
	return models.BlogPostListResponse{
		Posts: append([]models.BlogPost{}, posts[debut:fin]...), Total: int64(total),
		Page: page, PerPage: perPage, TotalPages: pages,
	}
}

// Post — un article par son slug. ErrBlogPostAbsent si l'article n'existe pas.
func (b *BlogCache) Post(ctx context.Context, slug, lang string) (*models.BlogPost, error) {
	var article models.BlogPost
	err := b.store.lire(ctx, cleArticle(lang, slug), &article, func(c context.Context) (interface{}, error) {
		return b.source.GetBlogPost(c, slug, lang)
	})
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// Invalidate — efface tout le cache du blog. Appelé après chaque écriture de l'owner.
// COMMENT : tout, et non la seule clé touchée — une publication change la liste, les pages de
// pagination et les compteurs, et l'id reçu par les routes admin ne donne pas le slug. Le volume
// (quelques dizaines de clés) rend le calcul fin inutile ; le prochain visiteur repeuple.
func (b *BlogCache) Invalidate(ctx context.Context) {
	b.store.supprimerPrefixe(ctx, blogCachePrefix+":")
}
