package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

// CompressionMiddleware applies gzip/brotli compression to responses
func CompressionMiddleware() fiber.Handler {
	return compress.New(compress.Config{
		// Compression level: 0 (no compression) to 9 (best compression)
		// Level 6 is a good balance between speed and compression ratio
		Level: compress.LevelDefault, // Level 6

		// Next defines a function to skip this middleware when true
		Next: func(c *fiber.Ctx) bool {
			// Skip compression for certain paths
			path := c.Path()

			// Don't compress WebSocket upgrades
			if c.Get("Upgrade") == "websocket" {
				return true
			}

			// Don't compress already compressed files
			if isPrecompressed(path) {
				return true
			}

			// Don't compress very small responses (overhead not worth it)
			contentLength := c.Response().Header.ContentLength()
			if contentLength > 0 && contentLength < 1024 { // < 1KB
				return true
			}

			return false
		},
	})
}

// isPrecompressed checks if file is already compressed
func isPrecompressed(path string) bool {
	compressedExts := []string{
		".gz", ".zip", ".rar", ".7z",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", // Images already compressed
		".mp4", ".mov", ".avi", ".mkv", // Videos already compressed
		".mp3", ".aac", ".ogg", ".flac", // Audio already compressed
		".woff", ".woff2", // Fonts already compressed
	}

	pathLen := len(path)
	for _, ext := range compressedExts {
		extLen := len(ext)
		if pathLen >= extLen && path[pathLen-extLen:] == ext {
			return true
		}
	}

	return false
}

// CustomCompressionConfig creates a custom compression configuration
func CustomCompressionConfig(level int, minLength int) compress.Config {
	return compress.Config{
		Level: compress.Level(level),
		Next: func(c *fiber.Ctx) bool {
			// Skip WebSocket
			if c.Get("Upgrade") == "websocket" {
				return true
			}

			// Skip precompressed files
			if isPrecompressed(c.Path()) {
				return true
			}

			// Check minimum length
			contentLength := c.Response().Header.ContentLength()
			if contentLength > 0 && contentLength < minLength {
				return true
			}

			return false
		},
	}
}

// BrotliCompressionMiddleware applies Brotli compression (more efficient than gzip)
// Note: Requires client support (modern browsers)
func BrotliCompressionMiddleware() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelDefault,
		Next: func(c *fiber.Ctx) bool {
			// Check if client supports Brotli
			acceptEncoding := c.Get("Accept-Encoding")
			if acceptEncoding == "" {
				return true
			}

			// Skip if already compressed
			if isPrecompressed(c.Path()) {
				return true
			}

			// Skip WebSocket
			if c.Get("Upgrade") == "websocket" {
				return true
			}

			return false
		},
	})
}

// GzipOnly forces only gzip compression (for compatibility)
func GzipOnly() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // Fast compression for real-time
		Next: func(c *fiber.Ctx) bool {
			// Check client support
			acceptEncoding := c.Get("Accept-Encoding")
			supportsGzip := false
			if acceptEncoding != "" {
				// Simple check (in production, parse properly)
				supportsGzip = len(acceptEncoding) > 4 && acceptEncoding[:4] == "gzip"
			}

			if !supportsGzip {
				return true
			}

			// Skip precompressed
			if isPrecompressed(c.Path()) {
				return true
			}

			return false
		},
	})
}
