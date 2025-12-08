package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// SecurityHeadersConfig configures security headers behavior
type SecurityHeadersConfig struct {
	// Content Security Policy
	CSP string
	// Enable HSTS
	EnableHSTS bool
	// HSTS max age in seconds
	HSTSMaxAge int
	// Include subdomains in HSTS
	HSTSIncludeSubdomains bool
	// HSTS preload
	HSTSPreload bool
	// Environment (development, production)
	Environment string
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders(config ...SecurityHeadersConfig) fiber.Handler {
	var cfg SecurityHeadersConfig

	if len(config) > 0 {
		cfg = config[0]
	} else {
		// Default configuration
		cfg = SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            63072000, // 2 years
			HSTSIncludeSubdomains: true,
			HSTSPreload:           true,
			Environment:           "production",
		}
	}

	// Default CSP if not provided
	if cfg.CSP == "" {
		cfg.CSP = getDefaultCSP(cfg.Environment)
	}

	return func(c *fiber.Ctx) error {
		// X-Frame-Options: Prevents clickjacking attacks
		c.Set("X-Frame-Options", "DENY")

		// X-Content-Type-Options: Prevents MIME sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection: Legacy XSS protection (modern browsers use CSP)
		c.Set("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: Controls referrer information
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy: Controls resource loading
		c.Set("Content-Security-Policy", cfg.CSP)

		// Permissions-Policy: Controls browser features
		c.Set("Permissions-Policy", getPermissionsPolicy())

		// Strict-Transport-Security (HSTS): Forces HTTPS
		if cfg.EnableHSTS && cfg.Environment == "production" {
			hstsValue := getHSTSValue(cfg)
			c.Set("Strict-Transport-Security", hstsValue)
		}

		// X-DNS-Prefetch-Control: Controls DNS prefetching
		c.Set("X-DNS-Prefetch-Control", "on")

		// X-Download-Options: Prevents IE from executing downloads
		c.Set("X-Download-Options", "noopen")

		// X-Permitted-Cross-Domain-Policies: Controls Adobe Flash/PDF cross-domain
		c.Set("X-Permitted-Cross-Domain-Policies", "none")

		// Cache-Control for sensitive endpoints
		path := string(c.Request().URI().Path())
		if isSensitivePath(path) {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
		}

		return c.Next()
	}
}

// getDefaultCSP returns a default Content Security Policy
func getDefaultCSP(environment string) string {
	if environment == "development" {
		// More permissive for development (allows localhost, unsafe-eval for hot reload)
		return "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com data:; " +
			"img-src 'self' data: https: http://localhost:*; " +
			"connect-src 'self' http://localhost:* ws://localhost:* wss://localhost:* https://api.anthropic.com https://api.openai.com; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self';"
	}

	// Production: strict CSP
	return "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
		"font-src 'self' https://fonts.gstatic.com data:; " +
		"img-src 'self' data: https:; " +
		"connect-src 'self' https://api.anthropic.com https://api.openai.com wss:; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"upgrade-insecure-requests;"
}

// getPermissionsPolicy returns Permissions-Policy header value
func getPermissionsPolicy() string {
	// Disable unnecessary browser features
	return "geolocation=(), " +
		"microphone=(), " +
		"camera=(), " +
		"payment=(), " +
		"usb=(), " +
		"magnetometer=(), " +
		"gyroscope=(), " +
		"speaker=(), " +
		"vibrate=(), " +
		"fullscreen=(self), " +
		"sync-xhr=()"
}

// getHSTSValue constructs HSTS header value
func getHSTSValue(cfg SecurityHeadersConfig) string {
	hsts := ""

	// Max age
	hsts += "max-age=" + string(rune(cfg.HSTSMaxAge))

	// Include subdomains
	if cfg.HSTSIncludeSubdomains {
		hsts += "; includeSubDomains"
	}

	// Preload
	if cfg.HSTSPreload {
		hsts += "; preload"
	}

	return hsts
}

// isSensitivePath checks if a path should have strict cache control
func isSensitivePath(path string) bool {
	sensitivePaths := []string{
		"/api/letters/generate",
		"/api/analytics",
		"/api/cv",
	}

	for _, sensitive := range sensitivePaths {
		if path == sensitive {
			return true
		}
	}

	return false
}

// NoCache middleware explicitly disables caching for specific routes
func NoCache() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	}
}

// CacheControl middleware sets appropriate cache headers for static assets
func CacheControl(maxAge int) fiber.Handler {
	cacheValue := ""
	if maxAge > 0 {
		cacheValue = "public, max-age=" + string(rune(maxAge))
	} else {
		cacheValue = "no-cache"
	}

	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", cacheValue)
		return c.Next()
	}
}

// SecureConfig returns a production-ready security headers configuration
func SecureConfig(environment string) SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableHSTS:            environment == "production",
		HSTSMaxAge:            63072000, // 2 years
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		Environment:           environment,
		CSP:                   getDefaultCSP(environment),
	}
}

// CustomCSP creates a custom CSP builder
type CSPBuilder struct {
	directives map[string][]string
}

// NewCSPBuilder creates a new CSP builder
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[string][]string),
	}
}

// DefaultSrc sets default-src directive
func (b *CSPBuilder) DefaultSrc(sources ...string) *CSPBuilder {
	b.directives["default-src"] = sources
	return b
}

// ScriptSrc sets script-src directive
func (b *CSPBuilder) ScriptSrc(sources ...string) *CSPBuilder {
	b.directives["script-src"] = sources
	return b
}

// StyleSrc sets style-src directive
func (b *CSPBuilder) StyleSrc(sources ...string) *CSPBuilder {
	b.directives["style-src"] = sources
	return b
}

// ImgSrc sets img-src directive
func (b *CSPBuilder) ImgSrc(sources ...string) *CSPBuilder {
	b.directives["img-src"] = sources
	return b
}

// ConnectSrc sets connect-src directive
func (b *CSPBuilder) ConnectSrc(sources ...string) *CSPBuilder {
	b.directives["connect-src"] = sources
	return b
}

// FrameAncestors sets frame-ancestors directive
func (b *CSPBuilder) FrameAncestors(sources ...string) *CSPBuilder {
	b.directives["frame-ancestors"] = sources
	return b
}

// Build constructs the CSP string
func (b *CSPBuilder) Build() string {
	csp := ""
	for directive, sources := range b.directives {
		if len(sources) > 0 {
			csp += directive + " "
			for i, source := range sources {
				csp += source
				if i < len(sources)-1 {
					csp += " "
				}
			}
			csp += "; "
		}
	}
	return csp
}
