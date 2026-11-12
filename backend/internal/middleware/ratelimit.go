package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

// estSondeSante dit si le chemin est une sonde de santé (/health, /health/deep).
// POURQUOI exemptées : c'est le healthcheck docker et les moniteurs qui les appellent, à
// intervalle fixe. Un 429 dessus marque le backend `unhealthy` alors qu'il va bien, et leur trafic
// régulier mangeait le quota partagé des requêtes internes (cf. panne du 2026-10-09).
func estSondeSante(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/health/")
}

// Global rate limiting par IP
func (rlm *RateLimitMiddleware) Global() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if estSondeSante(c.Path()) {
			return c.Next()
		}

		ctx := context.Background()
		ip := c.IP()

		// Clé Redis pour rate limiting global
		key := fmt.Sprintf("ratelimit:global:%s", ip)

		// Incrémenter le compteur ET lire son TTL en un seul aller-retour (pipeline).
		// POURQUOI lire le TTL : l'ancien code ne posait l'EXPIRE que si count == 1. Si le
		// processus mourait entre l'INCR et l'EXPIRE (conteneur recréé pendant un déploiement), la
		// clé restait SANS échéance et le compteur ne redescendait plus jamais — 429 permanent.
		// Panne réelle du 2026-10-09 : `ratelimit:global:` bloquée à 162, TTL -1, blog entier en
		// erreur. Désormais, toute clé trouvée sans TTL (-1) en reçoit un : l'état se répare de
		// lui-même à la requête suivante au lieu de dépendre d'une purge à la main.
		pipe := rlm.redis.Pipeline()
		incr := pipe.Incr(ctx, key)
		ttl := pipe.TTL(ctx, key)
		if _, err := pipe.Exec(ctx); err != nil {
			log.Error().Err(err).Msg("Redis incr failed for rate limit")
			return c.Next() // Fail open (ne pas bloquer si Redis down)
		}
		count := incr.Val()

		// Poser la fenêtre : à la première requête, ou sur une clé orpheline (TTL -1 = sans échéance).
		if count == 1 || ttl.Val() < 0 {
			if err := rlm.redis.Expire(ctx, key, GlobalRateWindow).Err(); err != nil {
				log.Error().Err(err).Str("key", key).Msg("Redis expire failed for rate limit")
			}
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
