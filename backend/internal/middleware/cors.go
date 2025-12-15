package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS configure la politique CORS de l'application
func CORS(allowedOrigins []string) fiber.Handler {
	// Convert []string to comma-separated string
	originsStr := ""
	for i, origin := range allowedOrigins {
		if i > 0 {
			originsStr += ","
		}
		originsStr += origin
	}

	return cors.New(cors.Config{
		// Origins autorisées (depuis config)
		AllowOrigins: originsStr, // Ex: "http://localhost:3000,https://maicivy.com"

		// Méthodes autorisées
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",

		// Headers autorisés
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",

		// Headers exposés au client
		ExposeHeaders: "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",

		// Autoriser credentials (cookies)
		AllowCredentials: true,

		// Cache preflight requests (24h)
		MaxAge: 86400,
	})
}
