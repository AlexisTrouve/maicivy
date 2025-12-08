package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/rs/zerolog/log"

	"maicivy/internal/api"
	"maicivy/internal/config"
	"maicivy/internal/database"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
	"maicivy/pkg/logger"
)

func main() {
	// 1. Charger la configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// 2. Initialiser le logger
	logger.Init(cfg.Environment)

	// 3. Connexion PostgreSQL
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}

	// 4. Connexion Redis
	redisClient, err := database.ConnectRedis(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	// 5. Créer l'application Fiber
	app := fiber.New(fiber.Config{
		AppName:      "maicivy API",
		ServerHeader: "Fiber",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    4 * 1024 * 1024, // 4MB max body size
		ErrorHandler: customErrorHandler,
	})

	// 6. Middlewares globaux (ORDRE IMPORTANT)

	// 1. CORS (sécurité en premier)
	app.Use(middleware.CORS(cfg.AllowedOrigins))

	// 2. Recovery (capture panics)
	app.Use(middleware.Recovery())

	// 3. Request ID (tracing)
	app.Use(middleware.RequestID())

	// 4. Logger (avec request ID)
	app.Use(middleware.Logger())

	// 5. Compression
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// 6. Tracking visiteurs
	trackingMW := middleware.NewTracking(db, redisClient)
	app.Use(trackingMW.Handler())

	// 7. Rate limiting global
	rateLimitMW := middleware.NewRateLimit(redisClient)
	app.Use(rateLimitMW.Global())

	// 7. Initialiser services
	cvService := services.NewCVService(db, redisClient)

	// 8. Initialiser handlers
	healthHandler := api.NewHealthHandler(db, redisClient)
	cvHandler := api.NewCVHandler(cvService)

	// 9. Routes
	app.Get("/health", healthHandler.Health)
	app.Get("/health/deep", healthHandler.HealthDeep)

	// Groupes API (prêts pour Phase 2+)
	apiV1 := app.Group("/api/v1")
	apiV1.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "maicivy API v1",
			"version": "1.0.0",
		})
	})

	// Routes CV (Phase 2 - IMPLEMENTED)
	cvHandler.RegisterRoutes(app)

	// Routes Letters avec rate limiting AI (Phase 3)
	// lettersGroup := apiV1.Group("/letters")
	// lettersGroup.Use(rateLimitMW.AI()) // Rate limit AI appliqué ici
	// lettersGroup.Post("/generate", generateLetterHandler)
	// lettersGroup.Get("/:id", getLetterHandler)

	// 10. Graceful shutdown
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
		log.Info().
			Str("addr", addr).
			Str("environment", cfg.Environment).
			Msg("Starting server")

		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// 11. Attendre signal de shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

// customErrorHandler gère les erreurs Fiber
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	log.Error().
		Err(err).
		Int("status", code).
		Str("path", c.Path()).
		Msg("Request error")

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"code":  code,
	})
}
