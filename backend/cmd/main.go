package main

import (
	"context"
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
	"maicivy/internal/jobs"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
	"maicivy/internal/websocket"
	"maicivy/internal/workers"
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

	// 3.5. Run database migrations
	if err := database.RunAutoMigrations(db); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
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

	// 7. Initialiser services (needed for analytics middleware)
	cvService := services.NewCVService(db, redisClient)
	analyticsService := services.NewAnalyticsService(db, redisClient)

	// 8. Analytics middleware (après tracking pour avoir visitor_id)
	analyticsMW := middleware.NewAnalytics(analyticsService)
	app.Use(analyticsMW.Handler())

	// 9. Rate limiting global
	rateLimitMW := middleware.NewRateLimit(redisClient)
	app.Use(rateLimitMW.Global())

	// AI Config et services
	aiConfig := config.LoadAIConfig()
	aiService, err := services.NewAIService(aiConfig, &services.DefaultMetricsRecorder{})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize AI service - letters generation will be unavailable")
	}

	// Scraper services
	scraperConfig := config.LoadScraperConfig()
	scraper := services.NewCompanyScraper(scraperConfig, redisClient)

	// Letter queue service
	letterQueueService := services.NewLetterQueueService(redisClient)

	// Profile builder service
	profileBuilder := services.NewProfileBuilder(db)

	// Build user profile from database
	userProfile := profileBuilder.BuildProfile(context.Background())

	// PDF letter service
	pdfLetterService, err := services.NewPDFLetterService("templates/letters")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize PDF letter service - PDF generation will be unavailable")
		pdfLetterService = nil
	}

	// Letter generator service (combines AI, scraper, PDF)
	var letterGenerator *services.LetterGenerator
	if aiService != nil && scraper != nil {
		letterGenerator = services.NewLetterGenerator(aiService, scraper, pdfLetterService, userProfile)
		log.Info().Msg("Letter generator service initialized")
	} else {
		log.Warn().Msg("Letter generator service not initialized - AI or scraper missing")
	}

	// GitHub services
	githubOAuthService := services.NewGitHubOAuthService(db, redisClient)
	githubSyncService := services.NewGitHubSyncService(db, redisClient)

	// Timeline service (currently not used in routes but initialized for future use)
	_ = services.NewTimelineService(db, redisClient)

	// Profile detection services
	clearbitClient := services.NewClearbitClient(redisClient)
	uaParser := services.NewUserAgentParser()
	profileDetector := services.NewProfileDetectorService(db, redisClient, clearbitClient, uaParser)

	// PDF service for CV export (separate from letter PDF service)
	_ = services.NewPDFService()

	// 8. Initialiser handlers
	healthHandler := api.NewHealthHandler(db, redisClient)
	cvHandler := api.NewCVHandler(cvService)
	analyticsHandler := api.NewAnalyticsHandler(analyticsService)
	lettersHandler := api.NewLettersHandler(db, redisClient, letterQueueService)
	githubHandler := api.NewGitHubHandler(githubOAuthService, githubSyncService)
	timelineHandler := api.NewTimelineHandler(db)
	profileHandler := api.NewProfileHandler(db, redisClient, profileDetector)
	swaggerHandler := api.NewSwaggerHandler()
	visitorHandler := api.NewVisitorHandler(db, redisClient)

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

	// Routes Letters avec rate limiting AI (Phase 3 - IMPLEMENTED)
	lettersGroup := apiV1.Group("/letters")
	// Rate limit AI uniquement sur /generate (utilise le bon middleware avec incrémentation après succès)
	aiRateLimitMW := middleware.AIRateLimit(middleware.AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,
		CooldownDuration: 2 * time.Minute,
	})
	lettersGroup.Post("/generate", aiRateLimitMW, lettersHandler.GenerateLetter)
	lettersGroup.Get("/job/:jobId", lettersHandler.GetJobStatus)
	lettersGroup.Get("/pair", lettersHandler.GetLetterPair) // ?company=Google
	lettersGroup.Get("/history", lettersHandler.GetHistory)
	lettersGroup.Get("/access/status", lettersHandler.GetAccessStatus)
	lettersGroup.Get("/ratelimit/status", lettersHandler.GetRateLimitStatus)
	lettersGroup.Get("/:id/pdf", lettersHandler.DownloadPDF)
	lettersGroup.Get("/:id", lettersHandler.GetLetter) // Must be last (catch-all)

	// Routes Analytics (Phase 4 - IMPLEMENTED)
	analyticsHandler.RegisterRoutes(app)

	// WebSocket for analytics real-time (Phase 4 - IMPLEMENTED)
	wsHandler := websocket.NewAnalyticsWSHandler(analyticsService, redisClient)
	wsHandler.RegisterRoutes(app)

	// Routes GitHub (Phase 5 - IMPLEMENTED)
	githubHandler.RegisterRoutes(apiV1)

	// Routes Timeline (Phase 5 - IMPLEMENTED)
	apiV1.Get("/timeline", timelineHandler.GetTimeline)
	apiV1.Get("/timeline/categories", timelineHandler.GetCategories)
	apiV1.Get("/timeline/milestones", timelineHandler.GetMilestones)

	// Routes Profile (Phase 5 - IMPLEMENTED)
	apiV1.Get("/profile/detect", profileHandler.GetDetect)
	apiV1.Get("/profile/current", profileHandler.GetCurrentProfile)

	// Routes Visitor (Tracking & Access Gate)
	apiV1.Get("/visitors/check", visitorHandler.CheckVisitorStatus)
	apiV1.Get("/visitor/status", visitorHandler.GetVisitorStatus)

	// Routes Swagger (Documentation API)
	swaggerHandler.RegisterRoutes(app)

	// TODO: WebSocket pour analytics temps réel (à implémenter)
	// app.Get("/ws/analytics", websocket.New(analyticsHandler.HandleWebSocket))

	// 10. Démarrer les background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Job 1: Analytics cleanup (daily at 2am)
	analyticsCleanupJob := jobs.NewAnalyticsCleanupJob(analyticsService, 90) // 90 jours de rétention
	go analyticsCleanupJob.Start(ctx)
	log.Info().Msg("Analytics cleanup job started")

	// Job 2: GitHub auto-sync (every 6 hours)
	githubAutoSyncJob := jobs.NewGitHubAutoSyncJob(db, githubSyncService)
	if err := githubAutoSyncJob.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start GitHub auto-sync job")
	} else {
		log.Info().Msg("GitHub auto-sync job started")
	}

	// Job 3: Letter generation worker (processes letter queue)
	if letterGenerator != nil {
		letterWorker := workers.NewLetterWorker(db, letterQueueService, aiService, scraper, letterGenerator, profileBuilder)
		go letterWorker.Start()
		log.Info().Msg("Letter generation worker started")
	} else {
		log.Warn().Msg("Letter generation worker not started - dependencies missing")
	}

	// 11. Graceful shutdown
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

	// 12. Attendre signal de shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Arrêter les background jobs
	log.Info().Msg("Stopping background jobs...")
	cancel() // Arrêter analytics cleanup job
	githubAutoSyncJob.Stop() // Arrêter GitHub auto-sync job

	// Arrêter le serveur HTTP
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped gracefully")
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
