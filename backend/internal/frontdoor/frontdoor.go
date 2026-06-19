// Package frontdoor construit le reverse-proxy "front-door" : le checkpoint anti-abus placé
// DEVANT tout le trafic.
//
// QUOI    : une app Fiber qui (1) applique middleware.SusRateLimit (signature de path incluse)
//
//	sur CHAQUE requête, puis (2) proxy /api/* → backend, tout le reste → frontend.
//
// POURQUOI: le sus-rate-limit vivant dans le backend ne voyait que /api (5% du scan). Ici il est
//
//	au point de convergence → il voit TOUT (le scan du frontend aussi). C'est le
//	"choke-point" identifié par le tuning ③ comme le levier #1.
//
// COMMENT : New(cfg) assemble l'app (testable en isolation avec de faux upstreams). Le proxy
//
//	réémet la requête telle quelle (proxy.Do) et recopie la réponse — le middleware
//	observe le vrai status renvoyé par backend/frontend pour scorer. Fail-open hérité du
//	middleware (Redis down → on ne bloque pas). /ws/ n'arrive jamais ici (bypass nginx).
package frontdoor

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"

	"maicivy/internal/middleware"
)

// Config paramètre le front-door.
type Config struct {
	Sus         middleware.SusConfig // checkpoint anti-abus (avec ScannerPath activé en prod)
	BackendURL  string               // ex: http://maicivy-backend:8080
	FrontendURL string               // ex: http://maicivy-frontend:3000
	BodyLimit   int                  // octets max du corps (0 = défaut Fiber) ; CV/offres volumineux
	Timeout     time.Duration        // timeout du proxy (0 = défaut 120s) ; génération CV synchrone longue
}

// New assemble l'app front-door (sans la démarrer — testable).
func New(cfg Config) *fiber.App {
	fcfg := fiber.Config{
		ProxyHeader:           "X-Real-IP", // l'IP réelle vient de nginx (cf. cloudflare-realip)
		DisableStartupMessage: true,
	}
	if cfg.BodyLimit > 0 {
		fcfg.BodyLimit = cfg.BodyLimit
	}
	app := fiber.New(fcfg)

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second // couvre la génération CV/IA synchrone
	}

	// Healthcheck local du front-door (non proxifié) — enregistré avant le checkpoint.
	app.Get("/__frontdoor/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// 1. Checkpoint anti-abus sur TOUT le trafic.
	app.Use(middleware.SusRateLimit(cfg.Sus))

	// 2. Routage : /api/* → backend, tout le reste → frontend. proxy.Do réémet la requête
	//    (méthode, headers dont X-Real-IP, corps) et recopie la réponse + son status.
	app.All("/*", func(c *fiber.Ctx) error {
		target := cfg.FrontendURL
		if strings.HasPrefix(c.Path(), "/api") {
			target = cfg.BackendURL
		}
		return proxy.DoTimeout(c, target+c.OriginalURL(), timeout)
	})

	return app
}
