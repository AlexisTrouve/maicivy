package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// PortfolioCache — le portfolio reste servi quand maiProFiles est lent ou tombé.
// ----------------------------------------------------------------------------
// QUOI : cache Redis stale-while-revalidate (swrStore) devant les deux lectures du portfolio : le
// livre d'une langue, et la page d'un projet.
//
// POURQUOI : panne réelle du 2026-09-27. Le VPS était saturé (load 30-40 pour 8 CPU), maiProFiles
// répondait en 30 s ou plus, le client abandonnait à 8 s et le portfolio renvoyait 502 sur TOUTES
// les pages — affichées en « 404 » par le front. Le seul cache existant était en mémoire, 5 min :
// il expirait, et chaque visite suivante retombait sur maiProFiles en direct.
//
// COMMENT : clé par (langue du livre, id), aucun TTL (clés en nombre fini : 3 éditions × ~25
// projets). Un projet retiré de la curation (ErrPortfolioAbsent au rafraîchissement) sort du cache.
// Warm() préchauffe tout au démarrage, pour que même le premier visiteur ne paie rien.
// ============================================================================

const (
	// portfolioCachePrefix — versionné : un changement de forme du blob se fait par bump de clé.
	portfolioCachePrefix = "portfolio:v1"
	// portfolioFraicheur — au-delà, on rafraîchit en fond. Une modification de la curation
	// apparaît en ~10 min sans jamais faire attendre un visiteur.
	portfolioFraicheur = 10 * time.Minute
)

// portfolioSource — ce que le cache sait lire en amont. Interface pour tester le cache contre une
// source simulée (lente, en panne, qui retire un projet) sans monter de serveur HTTP.
type portfolioSource interface {
	GetPortfolioBook(ctx context.Context, lang string) (*MPFBook, error)
	GetPortfolioPage(ctx context.Context, id, lang string) (*MPFPortfolioPage, error)
}

// PortfolioCache — voir l'en-tête du fichier.
type PortfolioCache struct {
	source portfolioSource
	store  *swrStore
}

// NewPortfolioCache — rdb peut être nil (tests, Redis absent) : le cache devient transparent.
func NewPortfolioCache(source *MaiProFilesClient, rdb *redis.Client) *PortfolioCache {
	return newPortfolioCache(source, rdb)
}

func newPortfolioCache(source portfolioSource, rdb *redis.Client) *PortfolioCache {
	return &PortfolioCache{
		source: source,
		store: &swrStore{
			redis:     rdb,
			nom:       "portfolio",
			fraicheur: portfolioFraicheur,
			ttl:       0, // clés en nombre fini : la donnée se rafraîchit, elle n'expire pas
			estAbsent: func(err error) bool { return errors.Is(err, ErrPortfolioAbsent) },
			limite:    limiteFondMaiProFiles,
		},
	}
}

func cleLivre(lang string) string { return fmt.Sprintf("%s:book:%s", portfolioCachePrefix, lang) }
func clePage(lang, id string) string {
	return fmt.Sprintf("%s:page:%s:%s", portfolioCachePrefix, lang, id)
}

// Book — le livre (index du portfolio) dans la langue demandée.
func (p *PortfolioCache) Book(ctx context.Context, lang string) (*MPFBook, error) {
	// Normalisation AVANT la clé : de/it partagent l'édition anglaise, inutile de la stocker trois fois.
	lang = langueDuLivre(lang)
	var livre MPFBook
	err := p.store.lire(ctx, cleLivre(lang), &livre, func(c context.Context) (interface{}, error) {
		return p.source.GetPortfolioBook(c, lang)
	})
	if err != nil {
		return nil, err
	}
	return &livre, nil
}

// Page — la page complète d'un projet curé.
func (p *PortfolioCache) Page(ctx context.Context, id, lang string) (*MPFPortfolioPage, error) {
	lang = langueDuLivre(lang)
	var page MPFPortfolioPage
	err := p.store.lire(ctx, clePage(lang, id), &page, func(c context.Context) (interface{}, error) {
		return p.source.GetPortfolioPage(c, id, lang)
	})
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Warm — remplit les pages ABSENTES du cache : les livres des éditions curées, puis chaque page
// qu'ils listent et qui n'est pas encore en Redis.
//
// POURQUOI : sans lui, le premier visiteur de chaque page paie maiProFiles en direct — c'est
// précisément ce qui échoue quand la machine est chargée.
// ⚠️ Une page DÉJÀ en cache est sautée, même périmée. Avant, Warm la lisait via Page(), ce qui
// déclenchait son rafraîchissement : à chaque démarrage, ~77 rafraîchissements d'un coup sur le
// worker unique de maiProFiles (mesuré le 2026-09-27, il a mis 8 min à s'en remettre). Une page
// périmée se rafraîchit à la visite suivante, une à la fois, sous le plafond limiteFondMaiProFiles.
func (p *PortfolioCache) Warm(ctx context.Context) {
	if p.store.redis == nil {
		return
	}
	pages, deja, echecs := 0, 0, 0
	for lang := range langsLivre {
		livre, err := p.Book(ctx, lang)
		if err != nil {
			echecs++
			log.Warn().Err(err).Str("lang", lang).Msg("portfolio: préchauffage du livre échoué")
			continue
		}
		for _, projet := range livre.Projects {
			if projet.ID == "" {
				continue
			}
			if n, err := p.store.redis.Exists(ctx, clePage(lang, projet.ID)).Result(); err == nil && n > 0 {
				deja++
				continue
			}
			if _, err := p.Page(ctx, projet.ID, lang); err != nil {
				echecs++
				continue
			}
			pages++
		}
	}
	log.Info().Int("pages", pages).Int("deja_en_cache", deja).Int("echecs", echecs).Msg("portfolio: préchauffage terminé")
}
