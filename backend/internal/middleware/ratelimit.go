package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// Rate limits globaux
	GlobalRateLimit  = 100             // requêtes par IP
	GlobalRateWindow = 1 * time.Minute // fenêtre de temps

	// Rate limits IA (depuis PROJECT_SPEC.md)
	AIGenerationsLimit  = 5                // max générations par session
	AIGenerationsWindow = 24 * time.Hour   // par jour
	AICooldown          = 2 * time.Minute  // cooldown entre générations
)

type RateLimitMiddleware struct {
	redis *redis.Client
}

func NewRateLimit(redisClient *redis.Client) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redis: redisClient,
	}
}

// Global rate limiting par IP
func (rlm *RateLimitMiddleware) Global() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		ip := c.IP()

		// Clé Redis pour rate limiting global
		key := fmt.Sprintf("ratelimit:global:%s", ip)

		// Incrémenter compteur
		count, err := rlm.redis.Incr(ctx, key).Result()
		if err != nil {
			log.Error().Err(err).Msg("Redis incr failed for rate limit")
			return c.Next() // Fail open (ne pas bloquer si Redis down)
		}

		// Set TTL si première requête dans fenêtre
		if count == 1 {
			rlm.redis.Expire(ctx, key, GlobalRateWindow)
		}

		// Vérifier limite
		if count > GlobalRateLimit {
			// Headers de rate limiting
			c.Set("X-RateLimit-Limit", strconv.Itoa(GlobalRateLimit))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", strconv.Itoa(int(GlobalRateWindow.Seconds())))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many requests",
				"message":     fmt.Sprintf("Rate limit exceeded. Max %d requests per minute.", GlobalRateLimit),
				"retry_after": GlobalRateWindow.Seconds(),
			})
		}

		// Headers de rate limiting
		c.Set("X-RateLimit-Limit", strconv.Itoa(GlobalRateLimit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(GlobalRateLimit-int(count)))

		return c.Next()
	}
}

// AI rate limiting par session (règles strictes)
func (rlm *RateLimitMiddleware) AI() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// Récupérer session ID depuis tracking middleware
		sessionID := c.Locals("session_id")
		if sessionID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No session found",
			})
		}

		sessionIDStr := sessionID.(string)

		// 1. Vérifier cooldown (2 minutes entre générations)
		cooldownKey := fmt.Sprintf("ratelimit:ai:cooldown:%s", sessionIDStr)
		exists, err := rlm.redis.Exists(ctx, cooldownKey).Result()
		if err != nil {
			log.Error().Err(err).Msg("Redis exists failed for cooldown check")
		} else if exists > 0 {
			ttl, _ := rlm.redis.TTL(ctx, cooldownKey).Result()

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Cooldown active",
				"message":     "Please wait before generating another letter",
				"retry_after": int(ttl.Seconds()),
			})
		}

		// 2. Vérifier limite journalière (5 générations/jour)
		dailyKey := fmt.Sprintf("ratelimit:ai:daily:%s", sessionIDStr)
		count, err := rlm.redis.Get(ctx, dailyKey).Int64()
		if err != nil && err != redis.Nil {
			log.Error().Err(err).Msg("Redis get failed for daily limit")
		}

		if count >= AIGenerationsLimit {
			ttl, _ := rlm.redis.TTL(ctx, dailyKey).Result()

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Daily limit reached",
				"message":     fmt.Sprintf("Maximum %d letter generations per day reached", AIGenerationsLimit),
				"retry_after": int(ttl.Seconds()),
			})
		}

		// NOTE: On n'incrémente PAS ici - c'est fait dans le handler APRÈS succès
		// via IncrementAIRateLimit()

		// Stocker remaining pour le handler
		c.Locals("rate_limit_remaining", int(AIGenerationsLimit-count))

		// Headers de rate limiting
		c.Set("X-RateLimit-AI-Limit", strconv.Itoa(AIGenerationsLimit))
		c.Set("X-RateLimit-AI-Remaining", strconv.Itoa(int(AIGenerationsLimit-count)))

		return c.Next()
	}
}
