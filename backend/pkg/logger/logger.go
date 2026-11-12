package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(environment string) {
	// Configuration selon environnement
	if environment == "development" {
		// Pretty logging en dev
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		// JSON logging en production
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().
		Str("environment", environment).
		Msg("Logger initialized")
}

// Helper functions pour logging standardisé
func Error(err error) *zerolog.Event {
	return log.Error().Err(err)
}

func Info() *zerolog.Event {
	return log.Info()
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

// LogSecurityEvent logs a security-related event
func LogSecurityEvent(eventType, message string, details map[string]interface{}) {
	event := log.Warn().
		Str("event_type", eventType).
		Str("message", message)

	for key, value := range details {
		event = event.Interface(key, value)
	}

	event.Msg("Security event detected")
}
