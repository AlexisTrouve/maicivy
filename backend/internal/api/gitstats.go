package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// GitStatsHandler expose les stats Git (Gitea + GitLab mergés)
type GitStatsHandler struct {
	gitea  *services.GiteaStatsService
	gitlab *services.GitLabStatsService // nil si non configuré (GITLAB_STATS_* absents)
	// partner : Gitea tiers (dépôt partagé d'un jeu partenaire) — nil si PARTNER_GITEA_* absents.
	partner *services.PartnerGiteaStatsService
}

// NewGitStatsHandler crée le handler — chaque source accepte nil (service désactivé)
func NewGitStatsHandler(gitea *services.GiteaStatsService, gitlab *services.GitLabStatsService, partner *services.PartnerGiteaStatsService) *GitStatsHandler {
	return &GitStatsHandler{gitea: gitea, gitlab: gitlab, partner: partner}
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

	// Merge des commits GitLab (projet partagé, filtrés sur l'auteur) dans les agrégats. Non bloquant :
	// si GitLab est down ou non configuré, on rend les stats Gitea seules (pas de trou).
	if h.gitlab != nil {
		if gitlabDaily, glErr := h.gitlab.GetDaily(c.Context(), force); glErr == nil {
			services.MergeGitLabDaily(stats, gitlabDaily)
			// Exposer AUSSI la série GitLab isolée : le front la trace en 2e courbe (« GitLab » vs
			// « Total »). Assigné après le merge, hors cache Redis → pas de pollution du blob gitstats.
			stats.GitlabDaily = gitlabDaily
		} else {
			log.Warn().Err(glErr).Msg("gitstats: GitLab merge skipped (Gitea-only)")
		}
	}

	// Merge du Gitea tiers (dépôt partagé, filtré sur l'email d'Alexis) — même contrat que GitLab :
	// non bloquant, et il n'alimente QUE les agrégats (le dépôt n'est pas listé, pas de courbe dédiée).
	// MergeGitLabDaily est générique sur des DayStat malgré son nom.
	if h.partner != nil {
		if partnerDaily, pErr := h.partner.GetDaily(c.Context(), force); pErr == nil {
			services.MergeGitLabDaily(stats, partnerDaily)
		} else {
			log.Warn().Err(pErr).Msg("gitstats: partner Gitea merge skipped")
		}
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
