package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// MontagneHandler sert la montagne des projets : strates, liens « sert à construire »,
// positions 3D précalculées et visuels (cf. services/montagne_service.go).
//
// La page /[locale]/montagne la lit côté serveur pour rendre la liste accessible des projets,
// puis confie la même donnée au canvas : une seule lecture par rendu, déjà cachée en Redis.
type MontagneHandler struct {
	service *services.MontagneService
}

func NewMontagneHandler(service *services.MontagneService) *MontagneHandler {
	return &MontagneHandler{service: service}
}

func (h *MontagneHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/montagne", h.Get)
}

// GET /api/v1/montagne?lang=fr — la montagne dans la langue du visiteur (fr/en/zh natifs, toute
// autre langue servie en anglais, jamais en français).
func (h *MontagneHandler) Get(c *fiber.Ctx) error {
	m, err := h.service.Get(c.Context(), c.Query("lang", "en"))
	if err != nil {
		// 502 et non 500 : la seule cause possible est l'amont (maiProFiles) quand rien n'est en
		// cache — la distinguer évite de chercher le défaut ici.
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "montagne indisponible", "detail": err.Error(),
		})
	}
	return c.JSON(m)
}
