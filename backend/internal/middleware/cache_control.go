package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CacheControlMiddleware adds appropriate Cache-Control headers
func CacheControlMiddleware(c *fiber.Ctx) error {
	path := c.Path()

	// Static assets: long-term caching (1 year)
	// Next.js hashes assets, so safe with long expires
	if isStaticAssetPath(path) {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	}

	// HTML pages: no cache (always fresh)
	if isHTMLPage(path) {
		c.Set("Cache-Control", "no-cache, must-revalidate")
		c.Set("Pragma", "no-cache")
		return c.Next()
	}

	// API responses (depends on endpoint)
	if isPublicAPI(path) {
		// /api/cv/themes - rarely changes
		if path == "/api/cv/themes" {
			c.Set("Cache-Control", "public, max-age=3600") // 1 hour
			return c.Next()
		}

		// /api/analytics/realtime - real-time data
		if isRealtimeAPI(path) {
			c.Set("Cache-Control", "no-cache, must-revalidate")
			return c.Next()
		}

		// Static API endpoints (skills, projects)
		if isStaticAPI(path) {
			c.Set("Cache-Control", "public, max-age=1800") // 30 minutes
			return c.Next()
		}

		// Default API: 5 minutes
		c.Set("Cache-Control", "public, max-age=300")
		return c.Next()
	}

	return c.Next()
}

// isStaticAssetPath checks if path is a static asset
func isStaticAssetPath(path string) bool {
	staticExts := []string{
		".js", ".css", ".woff", ".woff2", ".ttf",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif",
		".svg", ".ico", ".json",
	}

	pathLower := strings.ToLower(path)
	for _, ext := range staticExts {
		if strings.HasSuffix(pathLower, ext) {
			return true
		}
	}

	// Check for Next.js specific paths
	if strings.Contains(path, "/_next/static/") || strings.Contains(path, "/_next/image/") {
		return true
	}

	return false
}

// isHTMLPage checks if path is an HTML page
func isHTMLPage(path string) bool {
	// Root and main pages
	htmlPages := []string{
		"/",
		"/cv",
		"/letters",
		"/analytics",
		"/about",
		"/contact",
	}

	for _, page := range htmlPages {
		if path == page || path == page+"/" {
			return true
		}
	}

	// Check if it ends with .html
	return strings.HasSuffix(strings.ToLower(path), ".html")
}

// isPublicAPI checks if path is an API route
func isPublicAPI(path string) bool {
	return strings.HasPrefix(path, "/api/")
}

// isRealtimeAPI checks if path is a real-time API endpoint
func isRealtimeAPI(path string) bool {
	realtimeRoutes := []string{
		"/api/analytics/realtime",
		"/api/letters/history",
		"/api/visitors/current",
		"/ws/",
	}

	for _, route := range realtimeRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}

// isStaticAPI checks if path is a static API endpoint (rarely changes)
func isStaticAPI(path string) bool {
	staticRoutes := []string{
		"/api/skills",
		"/api/projects",
		"/api/experiences",
	}

	for _, route := range staticRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}

// NoCacheMiddleware forces no caching (useful for specific routes)
func NoCacheMiddleware(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return c.Next()
}

// CacheMiddleware forces caching with custom max-age
func CacheMiddleware(maxAge int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age="+string(rune(maxAge)))
		return c.Next()
	}
}

// ETags support for conditional requests
func ETagMiddleware(c *fiber.Ctx) error {
	// Get ETag from request
	ifNoneMatch := c.Get("If-None-Match")

	// Execute handler
	err := c.Next()
	if err != nil {
		return err
	}

	// Generate ETag from response body
	body := c.Response().Body()
	if len(body) > 0 {
		// Simple hash-based ETag (in production, use proper hashing)
		etag := `"` + string(rune(len(body))) + `"`
		c.Set("ETag", etag)

		// Return 304 Not Modified if ETag matches
		if ifNoneMatch == etag {
			c.Status(fiber.StatusNotModified)
			return nil
		}
	}

	return nil
}
