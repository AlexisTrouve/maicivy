package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"maicivy/pkg/logger"
)

// RateLimiterAdvanced provides sophisticated rate limiting with sliding windows
type RateLimiterAdvanced struct {
	redis *redis.Client
	ctx   context.Context
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	Limit      int           // Number of requests allowed
	Window     time.Duration // Time window
	KeyPrefix  string        // Redis key prefix
	Identifier string        // What to identify by (ip, session, user)
	Ban        BanConfig     // Ban configuration
}

// BanConfig configures temporary banning for abusive IPs
type BanConfig struct {
	Enabled       bool          // Enable temporary bans
	Threshold     int           // Number of violations before ban
	Duration      time.Duration // Ban duration
	ViolationTTL  time.Duration // How long to remember violations
}

func NewRateLimiterAdvanced(redisClient *redis.Client) *RateLimiterAdvanced {
	return &RateLimiterAdvanced{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// GlobalRateLimit: Default global rate limiting (100 requests/minute per IP)
func (rl *RateLimiterAdvanced) GlobalRateLimit() fiber.Handler {
	config := RateLimitConfig{
		Limit:      100,
		Window:     time.Minute,
		KeyPrefix:  "ratelimit:global",
		Identifier: "ip",
		Ban: BanConfig{
			Enabled:      true,
			Threshold:    5,  // 5 violations
			Duration:     time.Hour,
			ViolationTTL: 10 * time.Minute,
		},
	}
	return rl.RateLimitWithConfig(config)
}

// AIRateLimit: Rate limiting for AI endpoints (5 requests/day per session)
func (rl *RateLimiterAdvanced) AIRateLimit() fiber.Handler {
	config := RateLimitConfig{
		Limit:      5,
		Window:     24 * time.Hour,
		KeyPrefix:  "ratelimit:ai",
		Identifier: "session",
		Ban: BanConfig{
			Enabled:      true,
			Threshold:    10, // 10 violations (trying to abuse)
			Duration:     24 * time.Hour,
			ViolationTTL: time.Hour,
		},
	}
	return rl.RateLimitWithConfig(config)
}

// APIRateLimit: Rate limiting for API endpoints (1000 requests/hour per IP)
func (rl *RateLimiterAdvanced) APIRateLimit() fiber.Handler {
	config := RateLimitConfig{
		Limit:      1000,
		Window:     time.Hour,
		KeyPrefix:  "ratelimit:api",
		Identifier: "ip",
		Ban: BanConfig{
			Enabled:      false,
		},
	}
	return rl.RateLimitWithConfig(config)
}

// RateLimitWithConfig creates a rate limiter with custom configuration
func (rl *RateLimiterAdvanced) RateLimitWithConfig(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get identifier (IP or session)
		identifier := rl.getIdentifier(c, config.Identifier)
		if identifier == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request: missing identifier",
			})
		}

		// Check if banned
		if config.Ban.Enabled {
			banned, ttl := rl.isBanned(identifier, config.KeyPrefix)
			if banned {
				logger.LogSecurityEvent("banned_access_attempt", "Banned IP/session attempted access", map[string]interface{}{
					"identifier": identifier,
					"endpoint":   string(c.Request().URI().Path()),
					"ttl":        ttl.Seconds(),
				})

				return c.Status(403).JSON(fiber.Map{
					"error":       "Access denied: temporarily banned",
					"retry_after": int(ttl.Seconds()),
				})
			}
		}

		// Check rate limit using sliding window
		allowed, remaining, resetAt := rl.checkSlidingWindow(identifier, config)

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		if !allowed {
			// Record violation
			if config.Ban.Enabled {
				violations := rl.recordViolation(identifier, config)
				if violations >= config.Ban.Threshold {
					rl.banIdentifier(identifier, config)

					logger.LogSecurityEvent("auto_ban", "Identifier banned for excessive violations", map[string]interface{}{
						"identifier": identifier,
						"violations": violations,
						"duration":   config.Ban.Duration.String(),
					})
				}
			}

			// Log rate limit exceeded
			logger.LogSecurityEvent("rate_limit_exceeded", "Rate limit exceeded", map[string]interface{}{
				"identifier": identifier,
				"endpoint":   string(c.Request().URI().Path()),
				"limit":      config.Limit,
				"window":     config.Window.String(),
			})

			// Calculate retry-after
			retryAfter := resetAt - time.Now().Unix()
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

			return c.Status(429).JSON(fiber.Map{
				"error":         "Too many requests",
				"limit":         config.Limit,
				"window":        config.Window.String(),
				"retry_after":   retryAfter,
				"reset_at":      resetAt,
			})
		}

		return c.Next()
	}
}

// getIdentifier extracts the identifier from the request (IP or session)
func (rl *RateLimiterAdvanced) getIdentifier(c *fiber.Ctx, identifierType string) string {
	switch identifierType {
	case "ip":
		return c.IP()
	case "session":
		return c.Cookies("maicivy_session")
	case "user":
		// For future user authentication
		return c.Get("X-User-ID", "")
	default:
		return c.IP()
	}
}

// checkSlidingWindow implements sliding window rate limiting
// Returns (allowed, remaining, resetAt)
func (rl *RateLimiterAdvanced) checkSlidingWindow(identifier string, config RateLimitConfig) (bool, int, int64) {
	key := fmt.Sprintf("%s:%s", config.KeyPrefix, identifier)
	now := time.Now()
	windowStart := now.Add(-config.Window)

	// Use Redis sorted set for sliding window
	// Score = timestamp, Value = unique request ID
	pipe := rl.redis.Pipeline()

	// Remove old entries (outside window)
	pipe.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests in window
	countCmd := pipe.ZCard(rl.ctx, key)

	// Add current request
	requestID := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(rl.ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: requestID,
	})

	// Set expiration on key
	pipe.Expire(rl.ctx, key, config.Window+time.Minute)

	// Execute pipeline
	_, err := pipe.Exec(rl.ctx)
	if err != nil {
		// Fail open (allow request) but log error
		logger.LogSecurityEvent("rate_limit_error", "Rate limit check failed", map[string]interface{}{
			"identifier": identifier,
			"error":      err.Error(),
		})
		return true, config.Limit, now.Add(config.Window).Unix()
	}

	// Get count (before adding current request)
	count := int(countCmd.Val())

	// Calculate remaining and reset time
	remaining := config.Limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	// Reset time is when the oldest request in window expires
	resetAt := now.Add(config.Window).Unix()

	// Check if limit exceeded
	allowed := count < config.Limit

	return allowed, remaining, resetAt
}

// recordViolation records a rate limit violation
// Returns total violations in the violation window
func (rl *RateLimiterAdvanced) recordViolation(identifier string, config RateLimitConfig) int {
	key := fmt.Sprintf("%s:violations:%s", config.KeyPrefix, identifier)

	// Increment violations counter
	count, err := rl.redis.Incr(rl.ctx, key).Result()
	if err != nil {
		logger.LogSecurityEvent("violation_record_error", "Failed to record violation", map[string]interface{}{
			"identifier": identifier,
			"error":      err.Error(),
		})
		return 0
	}

	// Set TTL if first violation
	if count == 1 {
		rl.redis.Expire(rl.ctx, key, config.Ban.ViolationTTL)
	}

	return int(count)
}

// banIdentifier temporarily bans an identifier
func (rl *RateLimiterAdvanced) banIdentifier(identifier string, config RateLimitConfig) {
	key := fmt.Sprintf("%s:ban:%s", config.KeyPrefix, identifier)

	// Set ban flag with expiration
	err := rl.redis.Set(rl.ctx, key, "banned", config.Ban.Duration).Err()
	if err != nil {
		logger.LogSecurityEvent("ban_error", "Failed to ban identifier", map[string]interface{}{
			"identifier": identifier,
			"error":      err.Error(),
		})
	}
}

// isBanned checks if an identifier is currently banned
// Returns (banned, ttl)
func (rl *RateLimiterAdvanced) isBanned(identifier string, keyPrefix string) (bool, time.Duration) {
	key := fmt.Sprintf("%s:ban:%s", keyPrefix, identifier)

	// Check if ban key exists
	ttl := rl.redis.TTL(rl.ctx, key).Val()
	if ttl < 0 {
		return false, 0
	}

	return true, ttl
}

// Whitelist allows bypassing rate limits for trusted IPs/sessions
func (rl *RateLimiterAdvanced) Whitelist(whitelist []string) fiber.Handler {
	whitelistMap := make(map[string]bool)
	for _, item := range whitelist {
		whitelistMap[item] = true
	}

	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if whitelistMap[ip] {
			// Skip rate limiting
			c.Locals("whitelisted", true)
		}
		return c.Next()
	}
}

// GetRateLimitInfo returns current rate limit status for an identifier
func (rl *RateLimiterAdvanced) GetRateLimitInfo(identifier string, config RateLimitConfig) map[string]interface{} {
	key := fmt.Sprintf("%s:%s", config.KeyPrefix, identifier)
	now := time.Now()
	windowStart := now.Add(-config.Window)

	// Count requests in window
	count, err := rl.redis.ZCount(rl.ctx, key, fmt.Sprintf("%d", windowStart.UnixNano()), fmt.Sprintf("%d", now.UnixNano())).Result()
	if err != nil {
		count = 0
	}

	remaining := config.Limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Check if banned
	banned, banTTL := rl.isBanned(identifier, config.KeyPrefix)

	return map[string]interface{}{
		"limit":       config.Limit,
		"used":        int(count),
		"remaining":   remaining,
		"window":      config.Window.String(),
		"reset_at":    now.Add(config.Window).Unix(),
		"banned":      banned,
		"ban_expires": banTTL.Seconds(),
	}
}

// ResetRateLimit manually resets rate limit for an identifier (admin use)
func (rl *RateLimiterAdvanced) ResetRateLimit(identifier string, keyPrefix string) error {
	keys := []string{
		fmt.Sprintf("%s:%s", keyPrefix, identifier),
		fmt.Sprintf("%s:violations:%s", keyPrefix, identifier),
		fmt.Sprintf("%s:ban:%s", keyPrefix, identifier),
	}

	for _, key := range keys {
		err := rl.redis.Del(rl.ctx, key).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
