package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID ajoute un ID unique à chaque requête
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Vérifier si header X-Request-ID existe déjà (ex: proxy)
		requestID := c.Get("X-Request-ID")

		// Sinon, générer nouveau UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Stocker dans context local
		c.Locals("requestid", requestID)

		// Ajouter au header de réponse
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}
