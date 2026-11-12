package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Logger log chaque requête HTTP avec détails
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Exécuter handler
		err := c.Next()

		// Calculer durée
		duration := time.Since(start)

		// Préparer log structuré
		logEvent := log.Info()

		// Si erreur, log en error level
		if err != nil {
			logEvent = log.Error().Err(err)
		} else if c.Response().StatusCode() >= 400 {
			logEvent = log.Warn()
		}

		// Taille de la réponse. ⚠️ JAMAIS Body() sur une réponse en FLUX (SSE : chat, dossier
		// vivant) : fasthttp draine alors tout le flux en mémoire pour rendre ce corps, et le client
		// ne reçoit plus rien avant la fin (mesuré le 07/11/2026 : premier octet après 56 s, 504
		// Cloudflare). Un flux est journalisé avec size=-1. Verrouillé par logger_stream_test.go.
		size := -1
		if !c.Response().IsBodyStream() {
			size = len(c.Response().Body())
		}

		// Log complet
		logEvent.
			Str("request_id", c.Locals("requestid").(string)).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Int("status", c.Response().StatusCode()).
			Dur("duration_ms", duration).
			Str("user_agent", c.Get("User-Agent")).
			Int("size", size).
			Msg("HTTP request")

		return err
	}
}
