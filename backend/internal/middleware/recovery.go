package middleware

import (
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Recovery récupère les panics et renvoie une erreur 500
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Log panic avec stack trace
				log.Error().
					Str("request_id", c.Locals("requestid").(string)).
					Str("path", c.Path()).
					Str("method", c.Method()).
					Interface("panic", r).
					Bytes("stack", debug.Stack()).
					Msg("Panic recovered")

				// Réponse d'erreur
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":      "Internal server error",
					"request_id": c.Locals("requestid"),
				})
				if err != nil {
					log.Error().Err(err).Msg("Failed to send panic response")
				}
			}
		}()

		return c.Next()
	}
}
