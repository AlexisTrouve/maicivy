package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"maicivy/pkg/logger"
)

// CORSConfig configures strict CORS behavior
type CORSConfig struct {
	// Allowed origins (whitelist)
	AllowedOrigins []string
	// Environment (development, production)
	Environment string
	// Allow credentials (cookies)
	AllowCredentials bool
	// Max age for preflight cache
	MaxAge int
	// Allowed methods
	AllowedMethods []string
	// Allowed headers
	AllowedHeaders []string
	// Exposed headers
	ExposedHeaders []string
}

// StrictCORS creates a strict CORS middleware with whitelist
func StrictCORS(config CORSConfig) fiber.Handler {
	// Validate config
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = getDefaultAllowedOrigins(config.Environment)
	}

	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		}
	}

	if config.MaxAge == 0 {
		config.MaxAge = 3600 // 1 hour
	}

	// Log CORS configuration
	logger.LogSecurityEvent("cors_configured", "CORS middleware configured", map[string]interface{}{
		"allowed_origins": config.AllowedOrigins,
		"environment":     config.Environment,
		"credentials":     config.AllowCredentials,
	})

	return cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.AllowedOrigins, ","),
		AllowMethods:     strings.Join(config.AllowedMethods, ","),
		AllowHeaders:     strings.Join(config.AllowedHeaders, ","),
		AllowCredentials: config.AllowCredentials,
		ExposeHeaders:    strings.Join(config.ExposedHeaders, ","),
		MaxAge:           config.MaxAge,
	})
}

// getDefaultAllowedOrigins returns default allowed origins based on environment
func getDefaultAllowedOrigins(environment string) []string {
	if environment == "production" {
		// Production: only allow production domain
		return []string{
			"https://maicivy.example.com",
		}
	}

	// Development: allow localhost
	return []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:8080",
	}
}

// ValidateOrigin validates if an origin is in the whitelist
func ValidateOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}

		// Allow wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// CORSValidator middleware validates CORS requests and logs violations
func CORSValidator(allowedOrigins []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		// If no Origin header, it's not a CORS request (same-origin)
		if origin == "" {
			return c.Next()
		}

		// Validate origin
		if !ValidateOrigin(origin, allowedOrigins) {
			logger.LogSecurityEvent("cors_violation", "Request from unauthorized origin", map[string]interface{}{
				"origin":          origin,
				"ip":              c.IP(),
				"path":            string(c.Request().URI().Path()),
				"method":          c.Method(),
				"allowed_origins": allowedOrigins,
			})

			// Don't send CORS headers for unauthorized origins
			return c.Status(403).JSON(fiber.Map{
				"error": "Origin not allowed",
			})
		}

		return c.Next()
	}
}

// SecureCORSConfig returns a secure CORS configuration for production
func SecureCORSConfig(environment string, domains []string) CORSConfig {
	return CORSConfig{
		AllowedOrigins:   domains,
		Environment:      environment,
		AllowCredentials: true, // Needed for session cookies
		MaxAge:           3600,
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Retry-After",
		},
	}
}

// DevelopmentCORSConfig returns a permissive CORS configuration for development
func DevelopmentCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		},
		Environment:      "development",
		AllowCredentials: true,
		MaxAge:           3600,
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
			"PATCH",
		},
		AllowedHeaders: []string{
			"*", // Allow all headers in development
		},
		ExposedHeaders: []string{
			"*", // Expose all headers in development
		},
	}
}

// PreflightHandler handles OPTIONS requests for CORS preflight
func PreflightHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == "OPTIONS" {
			// Preflight requests should return 204 No Content
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

// IsValidOrigin checks if an origin uses HTTPS (in production)
func IsValidOrigin(origin string, environment string) bool {
	if environment == "production" {
		// Production: only allow HTTPS
		return strings.HasPrefix(origin, "https://")
	}

	// Development: allow HTTP
	return strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://")
}

// EnforceHTTPSOrigin middleware ensures origins use HTTPS in production
func EnforceHTTPSOrigin(environment string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		if origin != "" && environment == "production" {
			if !strings.HasPrefix(origin, "https://") {
				logger.LogSecurityEvent("insecure_origin", "HTTP origin in production", map[string]interface{}{
					"origin": origin,
					"ip":     c.IP(),
				})

				return c.Status(403).JSON(fiber.Map{
					"error": "HTTPS required",
				})
			}
		}

		return c.Next()
	}
}

// OriginLogger logs all CORS requests for monitoring
func OriginLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		if origin != "" {
			logger.LogSecurityEvent("cors_request", "CORS request received", map[string]interface{}{
				"origin": origin,
				"method": c.Method(),
				"path":   string(c.Request().URI().Path()),
				"ip":     c.IP(),
			})
		}

		return c.Next()
	}
}
