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
