// Binaire front-door : checkpoint anti-abus (sus-rate-limit + signature de path) devant tout le
// trafic, puis proxy /api→backend, reste→frontend. Voir internal/frontdoor pour la logique.
package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"maicivy/internal/frontdoor"
	"maicivy/internal/middleware"
)

// splitCSV découpe une liste CSV d'env (allowlist) en entrées ; "" → nil.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// env lit une variable d'env avec défaut.
func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// envF lit une variable d'env float avec défaut.
func envF(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, e := strconv.ParseFloat(v, 64); e == nil {
			return f
		}
	}
	return def
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_HOST", "redis") + ":" + env("REDIS_PORT", "6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	app := frontdoor.New(frontdoor.Config{
		Sus: middleware.SusConfig{
			Redis:         rdb,
			HalflifeHours: envF("SUS_HALFLIFE_HOURS", 48),
			AlertScore:    envF("SUS_ALERT_SCORE", 20),
			WebhookURL:    os.Getenv("SUS_WEBHOOK_URL"),
			// Signal signature de path ACTIVÉ ici (le checkpoint voit tout le trafic).
			ScannerPath: middleware.ScannerPathMatcher(middleware.AllScannerPatterns()...),
			// Exemptions : owner via header (IP-indépendant) + IPs/CIDR stables (allowlist).
			OwnerKey:  os.Getenv("MAICIVY_OWNER_API_KEY"),
			Allowlist: splitCSV(os.Getenv("SUS_ALLOWLIST")),
			// Types d'alerte mutés (loggés mais pas notifiés Discord). Défaut "php" : les scans
			// webshell PHP sont du bruit (aucun PHP servi). CSV, ex "php,wordpress".
			AlertMuteTypes: splitCSV(env("SUS_ALERT_MUTE_TYPES", "php")),
		},
		BackendURL:  env("BACKEND_URL", "http://maicivy-backend:8080"),
		FrontendURL: env("FRONTEND_URL", "http://maicivy-frontend:3000"),
		BodyLimit:   25 * 1024 * 1024, // CV / offres d'emploi peuvent être volumineux
	})

	port := env("FRONTDOOR_PORT", "8090")
	log.Info().Str("port", port).Msg("frontdoor up (sus checkpoint + proxy)")
	if err := app.Listen(":" + port); err != nil {
		log.Fatal().Err(err).Msg("frontdoor listen failed")
	}
}
