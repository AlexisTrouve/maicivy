// Command dossier-live — banc LOCAL du dossier vivant : un backend minimal qui sert ce dont la page
// /[locale]/dossier-vivant a besoin pour un run RÉEL, sans Postgres ni le reste du site.
//
// QUOI : sur LISTEN (défaut :8080) —
//   - POST /api/v1/admin/login, GET /api/v1/admin/me  : le vrai AdminHandler (mot de passe local) ;
//   - GET  /api/v1/montagne                           : le vrai MontagneService (maiProFiles, sans Redis) ;
//   - POST /api/v1/dossier-vivant/run (PUBLIQUE, sans cookie) et /api/v1/admin/dossier-vivant/run
//     (owner) : le VRAI handler et le vrai service d'agents, sur le vrai client IA (une seule
//     passerelle, ANTHROPIC_BASE_URL) ;
//   - GET  /api/v1/project-icons[/:id]                : le vrai ProjectIconService (Gitea, Redis en mémoire) ;
//   - POST /api/v1/dossier-vivant/pdf, /dossier-vivant/run/pdf, /admin/dossier-vivant/pdf : le PDF
//     du dossier (rejeu public, run réel public, run réel owner), sur le Chromium local ;
//   - tout autre GET /api/* : relayé en LECTURE SEULE vers UPSTREAM (le site public), pour les
//     données que la page lit en plus (fiches /api/v1/projects…). Tout autre verbe : 405.
//
// POURQUOI : le backend complet exige Postgres et Redis ; une preuve de run réel n'a besoin que de
// la route du run et de ses sources. Le banc réutilise les handlers de production, il n'en
// réécrit aucun.
//
// COMMENT (secrets) : la clé de passerelle arrive par l'environnement (coffre → run-secure), jamais
// écrite. ADMIN_PASSWORD et SESSION_SECRET sont des valeurs LOCALES du banc.
//
// DOSSIER_LIVE_FAKE_AGENTS=1 : les agents répondent par une FAUSSE passerelle locale
// (internal/dossiervivant/fakeagents) — tout le reste du chemin est celui de la production. C'est le mode de l'E2E
// du visiteur (aucune clé, aucun coût, sortie déterministe).
package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"maicivy/internal/api"
	"maicivy/internal/config"
	"maicivy/internal/dossiervivant"
	"maicivy/internal/dossiervivant/fakeagents"
	"maicivy/internal/services"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = hex.EncodeToString(b)
	}
	upstream := strings.TrimRight(env("UPSTREAM", "https://maicivy.etheryale.com"), "/")

	aiConfig := config.LoadAIConfig()
	if aiConfig.AnthropicBaseURL == "" {
		aiConfig.AnthropicBaseURL = "https://ai.etheryale.com"
	}
	if os.Getenv("DOSSIER_LIVE_FAKE_AGENTS") == "1" {
		aiConfig.AnthropicBaseURL = fakeagents.NewGateway(fakeagents.Hold).URL
		aiConfig.AnthropicAPIKey = "eai_fake_agents"
		log.Warn().Str("gateway", aiConfig.AnthropicBaseURL).Msg("dossier-live: agents SIMULÉS (fausse passerelle locale)")
	}
	mpf := services.NewMaiProFilesClient()
	montagne := services.NewMontagneService(mpf, nil)

	// Sans WriteTimeout : le banc mesure la durée réelle d'un run, sans la borne de transport de la
	// production (120 s, cf. api/dossier_vivant.go › connectionLimit, qui s'applique ici aussi).
	app := fiber.New(fiber.Config{AppName: "dossier-live"})
	app.Use(cors.New(cors.Config{AllowOrigins: env("CORS_ORIGIN", "http://localhost:3107"), AllowCredentials: true}))
	apiV1 := app.Group("/api/v1")

	api.NewAdminHandler(os.Getenv("ADMIN_PASSWORD"), secret).RegisterRoutes(apiV1)
	api.NewMontagneHandler(montagne).RegisterRoutes(apiV1)
	// PDF du dossier : les routes de production (rejeu public, run réel public et owner), même moteur.
	// Icônes des projets : le vrai service, sur le vrai Gitea (jeton GITEA_STATS_TOKEN par
	// l'environnement, lecture seule), avec un Redis EN MÉMOIRE — le banc n'a pas de Redis, et sans
	// cache chaque visite relirait tous les dépôts.
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("dossier-live: redis en mémoire")
	}
	memRedis := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	icons := services.NewProjectIconService(mpf, memRedis,
		env("GITEA_STATS_URL", "https://git.etheryale.com"), os.Getenv("GITEA_STATS_TOKEN"), env("GITEA_STATS_USER", "StillHammer"), os.Getenv("GITEA_STATS_EXCLUDE"))
	api.NewProjectIconsHandler(icons).RegisterRoutes(apiV1)
	// Dossiers remis : la copie serveur que les PDF publics impriment, dans le même Redis en mémoire.
	dossiers := services.NewDossierStore(memRedis)
	dossierPDF := api.NewDossierPDFHandler(services.NewDossierPDFService(services.NewDossierFactsSource(montagne, mpf)).WithIcons(icons), dossiers, secret, aiConfig.OwnerAPIKey)
	dossierPDF.RegisterPublicRoutes(apiV1)
	dossierPDF.RegisterOwnerRoutes(apiV1)
	if ai, err := services.NewAIService(aiConfig, nil); err == nil {
		svc := dossiervivant.NewService(ai, dossiervivant.NewLiveSources(montagne, mpf), dossiervivant.Options{Model: aiConfig.DossierModel, SchemaModel: aiConfig.DossierSchemaModel})
		run := api.NewDossierVivantHandler(svc, dossiers, secret, aiConfig.OwnerAPIKey)
		run.RegisterPublicRoutes(apiV1)
		run.RegisterOwnerRoutes(apiV1)
		log.Info().Str("model", aiConfig.DossierModel).Str("schema_model", aiConfig.DossierSchemaModel).Str("gateway", aiConfig.AnthropicBaseURL).Msg("dossier-live: route du run active")
	} else {
		log.Warn().Err(err).Msg("dossier-live: pas de clé de passerelle — route du run absente")
	}

	app.All("/api/*", func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
		return proxy.Forward(upstream + c.OriginalURL())(c)
	})

	listen := env("LISTEN", ":8080")
	log.Info().Str("listen", listen).Str("upstream", upstream).Msg("dossier-live: prêt")
	if err := app.Listen(listen); err != nil {
		log.Fatal().Err(err).Msg("dossier-live")
	}
}
