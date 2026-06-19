package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// GitStatsHandler expose les stats Git Gitea
type GitStatsHandler struct {
	gitea *services.GiteaStatsService
}

// NewGitStatsHandler crée le handler — accepte nil (service désactivé)
func NewGitStatsHandler(gitea *services.GiteaStatsService) *GitStatsHandler {
	return &GitStatsHandler{gitea: gitea}
}

// RegisterRoutes enregistre les routes git stats
func (h *GitStatsHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Get("/cv/gitstats", h.GetGitStats)
	api.Get("/cv/loc", h.GetLanguageStats) // LOC par langage → fiche détail skill
}

// GetGitStats retourne les stats Git agrégées (commits/jour, lignes/jour, repos)
func (h *GitStatsHandler) GetGitStats(c *fiber.Ctx) error {
	if h.gitea == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Git stats not available — GITEA_STATS_TOKEN not configured",
		})
	}

	force := c.Query("force") == "true"
	stats, err := h.gitea.GetStats(c.Context(), force)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetLanguageStats retourne les LOC agrégées par langage (octets Gitea → lignes approx).
// Alimente la fiche détail d'un skill côté frontend (pastille cliquable). Cache Redis à refresh
// lent : ?force=true le contourne.
func (h *GitStatsHandler) GetLanguageStats(c *fiber.Ctx) error {
	if h.gitea == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Git stats not available — GITEA_STATS_TOKEN not configured",
		})
	}

	force := c.Query("force") == "true"
	stats, err := h.gitea.GetLanguageStats(c.Context(), force)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
