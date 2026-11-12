package services

import (
	"context"

	"github.com/rs/zerolog/log"
)

// PagesPortfolio — les projets qui ont une page publique sur le site (/portfolio/<id>).
//
// QUOI : une fonction qui rend l'ensemble des identifiants du livre portfolio (les ~25 projets
// curés), ou nil si le livre est indisponible.
// POURQUOI : le Composer doit lier la PAGE du projet qu'il cite, pas l'accueil du site (constat
// Malt 2026, cf. checkProjectLink). Pour ça il faut savoir quels projets ont une page : l'agent
// (marque [page] dans l'inventaire) et le contrôle mécanique (lien mort) lisent la même liste.
// COMMENT : lu dans le PortfolioCache (Redis SWR) — la même source que les pages elles-mêmes, donc
// une page marquée existe. Les identifiants du livre sont ceux de l'inventaire (vérifié le
// 2026-10-12 : 25 sur 25). Échec → nil, que les appelants traitent comme « inconnu », jamais comme
// « aucune page ».
type PagesPortfolio func(ctx context.Context) map[string]bool

// NewPagesPortfolio branche la liste sur le cache du livre. cache nil → toujours nil.
func NewPagesPortfolio(cache *PortfolioCache) PagesPortfolio {
	return func(ctx context.Context) map[string]bool {
		if cache == nil {
			return nil
		}
		livre, err := cache.Book(ctx, "fr")
		if err != nil || livre == nil {
			log.Debug().Err(err).Msg("pages portfolio indisponibles")
			return nil
		}
		ids := make(map[string]bool, len(livre.Projects))
		for _, p := range livre.Projects {
			if p.ID != "" {
				ids[p.ID] = true
			}
		}
		return ids
	}
}
