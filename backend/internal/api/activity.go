package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// ActivityHandler gère les endpoints API d'activité
type ActivityHandler struct {
	scanner *services.RepoScanner
}

// NewActivityHandler crée une nouvelle instance
func NewActivityHandler(scanner *services.RepoScanner) *ActivityHandler {
	return &ActivityHandler{
		scanner: scanner,
	}
}

// RegisterRoutes enregistre les routes d'activité
func (h *ActivityHandler) RegisterRoutes(router fiber.Router) {
	activity := router.Group("/activity")

	activity.Get("/feed", h.GetFeed)
	activity.Get("/projects", h.GetProjects)
	activity.Get("/stats", h.GetStats)
	activity.Post("/refresh", h.TriggerRefresh)
}

// GetFeed retourne le feed d'activité (scan temps réel)
// GET /api/v1/activity/feed?showcase=true
func (h *ActivityHandler) GetFeed(c *fiber.Ctx) error {
	showcaseOnly := c.QueryBool("showcase", false)

	result, err := h.scanner.Scan(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_scan",
			"message": err.Error(),
		})
	}

	// Filter showcase if requested
	if showcaseOnly {
		var filtered []services.ScannedProject
		for _, p := range result.Projects {
			if p.Showcase {
				filtered = append(filtered, p)
			}
		}
		result.Projects = filtered
	}

	return c.JSON(result)
}

// GetProjects retourne la liste des projets
// GET /api/v1/activity/projects?showcase=true
func (h *ActivityHandler) GetProjects(c *fiber.Ctx) error {
	showcaseOnly := c.QueryBool("showcase", false)

	result, err := h.scanner.Scan(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_scan",
			"message": err.Error(),
		})
	}

	projects := result.Projects
	if showcaseOnly {
		var filtered []services.ScannedProject
		for _, p := range projects {
			if p.Showcase {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	return c.JSON(fiber.Map{
		"projects": projects,
	})
}

// GetStats retourne les statistiques d'activité
// GET /api/v1/activity/stats
func (h *ActivityHandler) GetStats(c *fiber.Ctx) error {
	result, err := h.scanner.Scan(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_scan",
			"message": err.Error(),
		})
	}

	return c.JSON(result.Stats)
}

// TriggerRefresh invalide le cache pour forcer un re-scan
// POST /api/v1/activity/refresh
func (h *ActivityHandler) TriggerRefresh(c *fiber.Ctx) error {
	h.scanner.InvalidateCache(c.Context())

	return c.JSON(fiber.Map{
		"status":  "cache_invalidated",
		"message": "Next request will trigger a fresh scan",
	})
}
