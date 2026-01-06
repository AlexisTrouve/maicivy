package middleware

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/models"
	"gorm.io/gorm"
)

// AccessGateConfig configuration pour le middleware d'accès IA
type AccessGateConfig struct {
	Redis           *redis.Client
	DB              *gorm.DB
	MinVisits       int  // Nombre minimum de visites (défaut: 3)
	BypassOnProfile bool // true = profils détectés bypass le compteur de visites
}

// AccessGate vérifie si le visiteur a accès aux fonctionnalités IA
// Règles: >= 3 visites OU profil cible détecté (recruiter, tech_lead, cto, ceo)
func AccessGate(config AccessGateConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// Récupérer le session ID depuis le cookie (même nom que tracking middleware)
		sessionID := c.Cookies("maicivy_session")
		if sessionID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session non trouvée",
				"code":  "SESSION_REQUIRED",
				"message": "Une session est requise pour accéder aux fonctionnalités IA. " +
					"Veuillez activer les cookies.",
			})
		}

		// Vérifier si profil détecté (bypass)
		if config.BypassOnProfile {
			profileKey := fmt.Sprintf("visitor:%s:profile", sessionID)
			profile, err := config.Redis.Get(ctx, profileKey).Result()

			if err == nil && profile != "" {
				// Vérifier que c'est un profil cible
				targetProfiles := []string{"recruiter", "tech_lead", "cto", "ceo"}
				for _, tp := range targetProfiles {
					if profile == tp {
						// Profil détecté - accès immédiat
						c.Locals("access_granted_reason", "profile_detected")
						c.Locals("profile_type", profile)
						return c.Next()
					}
				}
			}
		}

		// Récupérer le visiteur depuis la DB
		var visitor models.Visitor
		result := config.DB.Where("session_id = ?", sessionID).First(&visitor)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// Visiteur pas encore créé (premier passage)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Accès refusé",
					"code":  "INSUFFICIENT_VISITS",
					"details": fiber.Map{
						"current_visits":   0,
						"required_visits":  config.MinVisits,
						"visits_remaining": config.MinVisits,
					},
					"message": fmt.Sprintf(
						"Les fonctionnalités IA sont disponibles à partir de la %dème visite. "+
							"Encore %d visite(s) nécessaire(s).",
						config.MinVisits,
						config.MinVisits,
					),
					"teaser": "Découvrez la génération automatique de lettres de motivation " +
						"et anti-motivation personnalisées par IA ! Continuez à explorer le CV " +
						"pour débloquer cette fonctionnalité unique.",
				})
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erreur lors de la vérification d'accès",
				"code":  "INTERNAL_ERROR",
			})
		}

		// Vérifier le nombre de visites
		if visitor.VisitCount < config.MinVisits {
			visitsRemaining := config.MinVisits - visitor.VisitCount

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Accès refusé",
				"code":  "INSUFFICIENT_VISITS",
				"details": fiber.Map{
					"current_visits":   visitor.VisitCount,
					"required_visits":  config.MinVisits,
					"visits_remaining": visitsRemaining,
				},
				"message": fmt.Sprintf(
					"Les fonctionnalités IA sont disponibles à partir de la %dème visite. "+
						"Encore %d visite(s) nécessaire(s).",
					config.MinVisits,
					visitsRemaining,
				),
				"teaser": "Découvrez la génération automatique de lettres de motivation " +
					"et anti-motivation personnalisées par IA ! Continuez à explorer le CV " +
					"pour débloquer cette fonctionnalité unique.",
			})
		}

		// Accès accordé
		c.Locals("access_granted_reason", "visits_threshold")
		c.Locals("visit_count", visitor.VisitCount)
		c.Locals("session_id", sessionID)

		return c.Next()
	}
}
