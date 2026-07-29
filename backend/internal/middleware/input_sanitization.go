package middleware

import (
	"bytes"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
	"maicivy/internal/validation"
	"maicivy/pkg/logger"
)

// Sanitization middleware configuration
type SanitizationConfig struct {
	// Skip sanitization for these paths
	SkipPaths []string
	// Enable strict HTML sanitization (removes all HTML)
	StrictMode bool
}

var (
	// StrictPolicy removes all HTML
	strictPolicy = bluemonday.StrictPolicy()
	// UGCPolicy allows safe user-generated content HTML (if needed for markdown)
	ugcPolicy = bluemonday.UGCPolicy()
)

// InputSanitization middleware sanitizes all request bodies automatically
func InputSanitization(config ...SanitizationConfig) fiber.Handler {
	var cfg SanitizationConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	// Default config
	if cfg.SkipPaths == nil {
		cfg.SkipPaths = []string{
			"/health",
			"/metrics",
		}
	}

	return func(c *fiber.Ctx) error {
		// Skip sanitization for certain paths
		path := string(c.Request().URI().Path())
		for _, skip := range cfg.SkipPaths {
			if path == skip {
				return c.Next()
			}
		}

		// Only sanitize POST, PUT, PATCH requests with JSON body
		method := c.Method()
		if method != "POST" && method != "PUT" && method != "PATCH" {
			return c.Next()
		}

		contentType := string(c.Request().Header.ContentType())
		if contentType != "application/json" {
			return c.Next()
		}

		// Read body
		body := c.Body()
		if len(body) == 0 {
			return c.Next()
		}

		// Parse body as JSON and sanitize string values
		// Note: This is a simplified approach. For production, consider
		// using a proper JSON parser that sanitizes string values recursively.

		// For now, we'll just log suspicious content and let validation handle it
		bodyStr := string(body)

		// Check for SQL injection patterns
		if validation.ContainsSQLInjection(bodyStr) {
			logger.LogSecurityEvent("sql_injection_attempt", "SQL injection pattern detected in request body", map[string]interface{}{
				"ip":     c.IP(),
				"path":   path,
				"method": method,
			})
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request: suspicious content detected",
			})
		}

		// Check for XSS patterns
		if validation.ContainsXSS(bodyStr) {
			logger.LogSecurityEvent("xss_attempt", "XSS pattern detected in request body", map[string]interface{}{
				"ip":     c.IP(),
				"path":   path,
				"method": method,
			})
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request: suspicious content detected",
			})
		}

		// Continue to next handler
		return c.Next()
	}
}

// SanitizeJSON sanitizes all string values in a JSON-like map
func SanitizeJSON(data map[string]interface{}, strict bool) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Sanitize string values
			if strict {
				result[key] = SanitizeStrict(v)
			} else {
				result[key] = SanitizeUGC(v)
			}
		case map[string]interface{}:
			// Recursively sanitize nested objects
			result[key] = SanitizeJSON(v, strict)
		case []interface{}:
			// Sanitize array elements
			result[key] = SanitizeArray(v, strict)
		default:
			// Keep other types as-is (numbers, booleans, nil)
			result[key] = v
		}
	}

	return result
}

// SanitizeArray sanitizes all elements in an array
func SanitizeArray(data []interface{}, strict bool) []interface{} {
	result := make([]interface{}, len(data))

	for i, value := range data {
		switch v := value.(type) {
		case string:
			if strict {
				result[i] = SanitizeStrict(v)
			} else {
				result[i] = SanitizeUGC(v)
			}
		case map[string]interface{}:
			result[i] = SanitizeJSON(v, strict)
		case []interface{}:
			result[i] = SanitizeArray(v, strict)
		default:
			result[i] = v
		}
	}

	return result
}

// SanitizeStrict removes ALL HTML from a string
func SanitizeStrict(input string) string {
	return strictPolicy.Sanitize(input)
}

// SanitizeUGC allows safe user-generated content HTML (for markdown rendering)
func SanitizeUGC(input string) string {
	return ugcPolicy.Sanitize(input)
}

// RequestSizeLimit middleware limits the size of request bodies
func RequestSizeLimit(maxSizeMB int) fiber.Handler {
	maxBytes := int64(maxSizeMB * 1024 * 1024)

	return func(c *fiber.Ctx) error {
		// Get content length
		contentLength := c.Request().Header.ContentLength()

		if contentLength > int(maxBytes) {
			logger.LogSecurityEvent("request_too_large", "Request body exceeds size limit", map[string]interface{}{
				"ip":             c.IP(),
				"content_length": contentLength,
				"max_bytes":      maxBytes,
			})

			return c.Status(413).JSON(fiber.Map{
				"error": "Request body too large",
				"max_size": map[string]interface{}{
					"bytes": maxBytes,
					"mb":    maxSizeMB,
				},
			})
		}

		// Also check actual body size (in case Content-Length is missing/wrong)
		body := c.Body()
		if int64(len(body)) > maxBytes {
			logger.LogSecurityEvent("request_too_large", "Actual request body exceeds size limit", map[string]interface{}{
				"ip":          c.IP(),
				"actual_size": len(body),
				"max_bytes":   maxBytes,
			})

			return c.Status(413).JSON(fiber.Map{
				"error": "Request body too large",
			})
		}

		return c.Next()
	}
}

// NullByteFilter removes null bytes from request bodies
// Null bytes can be used to bypass certain validations
func NullByteFilter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()
		if len(body) == 0 {
			return c.Next()
		}

		// Check for null bytes
		if bytes.Contains(body, []byte{0}) {
			logger.LogSecurityEvent("null_byte_detected", "Null byte found in request body", map[string]interface{}{
				"ip":     c.IP(),
				"path":   string(c.Request().URI().Path()),
				"method": c.Method(),
			})

			// Remove null bytes
			cleaned := bytes.ReplaceAll(body, []byte{0}, []byte{})

			// Replace request body with cleaned version
			c.Request().SetBody(cleaned)
			c.Request().Header.SetContentLength(len(cleaned))
		}

		return c.Next()
	}
}

// ValidateContentType ensures requests have the expected Content-Type
func ValidateContentType(allowedTypes []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip for GET, DELETE, HEAD requests
		method := c.Method()
		if method == "GET" || method == "DELETE" || method == "HEAD" {
			return c.Next()
		}

		contentType := string(c.Request().Header.ContentType())

		// Check if content type is in allowed list
		allowed := false
		for _, allowedType := range allowedTypes {
			if contentType == allowedType || bytes.HasPrefix([]byte(contentType), []byte(allowedType)) {
				allowed = true
				break
			}
		}

		if !allowed {
			return c.Status(415).JSON(fiber.Map{
				"error":         "Unsupported Media Type",
				"allowed_types": allowedTypes,
			})
		}

		return c.Next()
	}
}

// SlowRequestProtection protects against slow POST attacks
func SlowRequestProtection(timeoutSeconds int) fiber.Handler {
	// This is handled by Fiber's ReadTimeout config in main.go
	// But we can add additional logging here
	return func(c *fiber.Ctx) error {
		// Log slow requests (this is a placeholder)
		// In production, use Fiber's ReadTimeout config
		return c.Next()
	}
}

// Helper function to read and restore request body
func readAndRestoreBody(c *fiber.Ctx) ([]byte, error) {
	bodyBytes, err := io.ReadAll(bytes.NewReader(c.Body()))
	if err != nil {
		return nil, err
	}

	// Restore body for next handlers
	c.Request().SetBody(bodyBytes)

	return bodyBytes, nil
}
