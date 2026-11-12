package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// PortfolioHandler sert le portfolio curé : l'index des projets choisis et leur page complète.
//
// La donnée vient de maiProFiles (`data/book.yaml`, la couche de curation) — la MÊME source qui
// produit le PDF du portfolio book. maicivy en rend les pages web, bookgen en rend le PDF ; aucun
// des deux ne recopie le contenu.
//
// Les lectures passent par PortfolioCache (Redis, stale-while-revalidate) et non plus par le client
// en direct : quand maiProFiles est lent ou tombé, on sert la dernière version connue au lieu d'un
// 502 (panne du 2026-09-27, cf. portfolio_cache.go).
type PortfolioHandler struct {
	cache *services.PortfolioCache
}

func NewPortfolioHandler(cache *services.PortfolioCache) *PortfolioHandler {
	return &PortfolioHandler{cache: cache}
}

func (h *PortfolioHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/portfolio", h.GetIndex)
	api.Get("/portfolio/:id", h.GetPage)
}

// GET /api/v1/portfolio?lang=fr — l'index des projets curés.
//
// La langue vient de la locale du visiteur ; les éditions curées sont en/fr/zh et toute autre
// langue retombe sur l'anglais (jamais le français, cf. langueDuLivre côté service).
func (h *PortfolioHandler) GetIndex(c *fiber.Ctx) error {
	livre, err := h.cache.Book(c.Context(), c.Query("lang", "en"))
	if err != nil {
		// 502 et non 500 : la panne est en amont, chez maiProFiles. Le distinguer évite de
		// chercher le défaut dans ce service quand il n'y est pas.
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "portfolio indisponible", "detail": err.Error(),
		})
	}
	return c.JSON(livre)
}

// GET /api/v1/portfolio/:id?lang=fr — la page complète d'un projet curé.
func (h *PortfolioHandler) GetPage(c *fiber.Ctx) error {
	id := c.Params("id")
	page, err := h.cache.Page(c.Context(), id, c.Query("lang", "en"))
	if err != nil {
		// Le portfolio est CURÉ : un projet du catalogue n'y est pas forcément. maiProFiles
		// répond 404 dans ce cas — on le rend tel quel, pour que le front affiche une page
		// « inconnue » plutôt qu'une erreur serveur trompeuse.
		if errors.Is(err, services.ErrPortfolioAbsent) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "projet absent de la curation du portfolio", "id": id,
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "portfolio indisponible", "detail": err.Error(),
		})
	}
	return c.JSON(page)
}
