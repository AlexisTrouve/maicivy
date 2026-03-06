package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// AIRateLimitConfig configuration pour le rate limiting IA
type AIRateLimitConfig struct {
	Redis            *redis.Client
	MaxPerDay        int           // Nombre maximum de générations par jour par session (défaut: 5)
	MaxPerDayPerIP   int           // Nombre maximum de générations par jour par IP (défaut: 3)
	CooldownDuration time.Duration // Temps d'attente entre générations (défaut: 2 minutes)
	OwnerAPIKey      string        // Si set : les requêtes avec X-Owner-Key matchant bypass le rate limit
}

// AIRateLimit middleware pour limiter les générations IA par session + par IP
// Trois limites :
// 1. Owner bypass : X-Owner-Key valide → pas de rate limit
// 2. Cooldown : CooldownDuration entre chaque génération (par session)
// 3. Limite journalière session : MaxPerDay/jour
// 4. Limite journalière IP : MaxPerDayPerIP/jour (protège contre incognito/nouvelles sessions)
func AIRateLimit(config AIRateLimitConfig) fiber.Handler {
	// Défaut IP limit si non configuré
	maxPerDayPerIP := config.MaxPerDayPerIP
	if maxPerDayPerIP <= 0 {
		maxPerDayPerIP = 3
	}

	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// --- Owner bypass : X-Owner-Key valide → skip tout le rate limiting ---
		// Permet à l'owner de générer sans limite via son API key
		if config.OwnerAPIKey != "" && c.Get("X-Owner-Key") == config.OwnerAPIKey {
			// Marquer comme owner dans les locals pour le handler (sélection de modèle)
			c.Locals("is_owner", true)
			return c.Next()
		}

		sessionID := c.Cookies("maicivy_session")
		if sessionID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session non trouvée",
				"code":  "SESSION_REQUIRED",
			})
		}

		// --- Vérification Cooldown (2 minutes) ---
		cooldownKey := fmt.Sprintf("ratelimit:ai:%s:cooldown", sessionID)
		cooldownExists, err := config.Redis.Exists(ctx, cooldownKey).Result()

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erreur lors de la vérification du rate limiting",
				"code":  "INTERNAL_ERROR",
			})
		}

		if cooldownExists > 0 {
			ttl, _ := config.Redis.TTL(ctx, cooldownKey).Result()

			c.Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.Set("X-RateLimit-Type", "cooldown")

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Cooldown actif",
				"code":        "COOLDOWN_ACTIVE",
				"retry_after": int(ttl.Seconds()),
				"message": fmt.Sprintf(
					"Merci de patienter %d seconde(s) avant la prochaine génération. "+
						"Cette limitation permet de maintenir la qualité du service.",
					int(ttl.Seconds()),
				),
			})
		}

		// --- Vérification Limite Journalière (5/jour) ---
		dailyKey := fmt.Sprintf("ratelimit:ai:%s:daily", sessionID)
		countStr, err := config.Redis.Get(ctx, dailyKey).Result()

		var count int
		if err == redis.Nil {
			count = 0
		} else if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erreur lors de la vérification du rate limiting",
				"code":  "INTERNAL_ERROR",
			})
		} else {
			fmt.Sscanf(countStr, "%d", &count)
		}

		if count >= config.MaxPerDay {
			ttl, _ := config.Redis.TTL(ctx, dailyKey).Result()

			c.Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxPerDay))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))
			c.Set("X-RateLimit-Type", "daily")

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Limite journalière atteinte",
				"code":  "DAILY_LIMIT_REACHED",
				"details": fiber.Map{
					"max_per_day": config.MaxPerDay,
					"used":        count,
					"reset_in":    int(ttl.Seconds()),
					"reset_at":    time.Now().Add(ttl).Format("2006-01-02 15:04:05"),
				},
				"message": fmt.Sprintf(
					"Vous avez atteint la limite de %d générations par jour. "+
						"Réinitialisation dans %s. Cette limitation permet de contrôler "+
						"les coûts d'API IA.",
					config.MaxPerDay,
					formatDuration(ttl),
				),
			})
		}

		// --- Vérification Limite Journalière par IP ---
		// Protège contre le bypass via incognito ou renouvellement de session
		ip := c.IP()
		ipDailyKey := fmt.Sprintf("ratelimit:ai:ip:%s:daily", ip)
		ipCountStr, err := config.Redis.Get(ctx, ipDailyKey).Result()

		var ipCount int
		if err == redis.Nil {
			ipCount = 0
		} else if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erreur lors de la vérification du rate limiting",
				"code":  "INTERNAL_ERROR",
			})
		} else {
			fmt.Sscanf(ipCountStr, "%d", &ipCount)
		}

		if ipCount >= maxPerDayPerIP {
			ttl, _ := config.Redis.TTL(ctx, ipDailyKey).Result()

			c.Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.Set("X-RateLimit-Type", "ip-daily")

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Limite journalière atteinte",
				"code":  "DAILY_LIMIT_REACHED",
				"details": fiber.Map{
					"max_per_day": maxPerDayPerIP,
					"used":        ipCount,
					"reset_in":    int(ttl.Seconds()),
					"reset_at":    time.Now().Add(ttl).Format("2006-01-02 15:04:05"),
				},
				"message": fmt.Sprintf(
					"Vous avez atteint la limite de %d générations par jour. "+
						"Réinitialisation dans %s.",
					maxPerDayPerIP,
					formatDuration(ttl),
				),
			})
		}

		// Passer les infos au handler pour incrémentation après succès
		c.Locals("rate_limit_session_id", sessionID)
		c.Locals("rate_limit_daily_key", dailyKey)
		c.Locals("rate_limit_cooldown_key", cooldownKey)
		c.Locals("rate_limit_ip_daily_key", ipDailyKey)
		c.Locals("rate_limit_remaining", config.MaxPerDay-count-1)

		// Headers informatifs
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxPerDay))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", config.MaxPerDay-count-1))

		return c.Next()
	}
}

// IncrementAIRateLimit à appeler APRÈS génération réussie
// Incrémente les compteurs de rate limiting (journalier session + journalier IP + cooldown)
// Si is_owner=true dans les locals (owner bypass actif), n'incrémente rien.
func IncrementAIRateLimit(c *fiber.Ctx, redis *redis.Client, cooldownDuration time.Duration) error {
	// Owner bypass : pas de compteur à incrémenter
	if isOwner, _ := c.Locals("is_owner").(bool); isOwner {
		return nil
	}

	ctx := context.Background()

	dailyKey, ok := c.Locals("rate_limit_daily_key").(string)
	if !ok {
		return fmt.Errorf("rate_limit_daily_key not found in context")
	}

	cooldownKey, ok := c.Locals("rate_limit_cooldown_key").(string)
	if !ok {
		return fmt.Errorf("rate_limit_cooldown_key not found in context")
	}

	// Calculer TTL jusqu'à minuit (reset journalier)
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	dailyTTL := midnight.Sub(now)

	// Incrémenter compteur journalier session
	count, err := redis.Incr(ctx, dailyKey).Result()
	if err != nil {
		return fmt.Errorf("failed to increment daily counter: %w", err)
	}
	if count == 1 {
		redis.Expire(ctx, dailyKey, dailyTTL)
	}

	// Incrémenter compteur journalier IP (si la clé est dans les locals)
	if ipDailyKey, ok := c.Locals("rate_limit_ip_daily_key").(string); ok && ipDailyKey != "" {
		ipCount, err := redis.Incr(ctx, ipDailyKey).Result()
		if err != nil {
			return fmt.Errorf("failed to increment IP daily counter: %w", err)
		}
		if ipCount == 1 {
			redis.Expire(ctx, ipDailyKey, dailyTTL)
		}
	}

	// Activer cooldown
	err = redis.Set(ctx, cooldownKey, "1", cooldownDuration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cooldown: %w", err)
	}

	return nil
}

// formatDuration formate une durée en format lisible (ex: "2h30m", "45m")
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}
