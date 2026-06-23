package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
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
		WriteTimeout: 120 * time.Second, // Sonnet 4.6 + CoT + ChromeDP PDF peuvent dépasser 10s
		IdleTimeout:  120 * time.Second,
		BodyLimit:    4 * 1024 * 1024, // 4MB max body size
		ErrorHandler: customErrorHandler,

		// QUOI : Fiber lit l'IP client réelle dans l'en-tête X-Real-IP au lieu de l'IP TCP directe.
		// POURQUOI : le backend tourne derrière nginx (bind 127.0.0.1:8081). Sans ça, c.IP() renvoie
		// toujours l'IP du proxy (172.18.0.1) → tous les rate-limits par IP et les logs/analytics
		// étaient écrasés sur une seule IP (faille : rate-limit par IP non-fonctionnel, forensics aveugle).
		// COMMENT : nginx pose `proxy_set_header X-Real-IP $remote_addr` qui ÉCRASE la valeur entrante
		// (donc non-spoofable par le client) ; on lit cet en-tête plutôt que X-Forwarded-For, que le
		// client peut préfixer d'une fausse IP que nginx se contente d'append (left-most spoofable).
		ProxyHeader: "X-Real-IP",
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

	// 5b. Anti-abus : le sus-rate-limit a MIGRÉ dans le front-door (cmd/frontdoor), placé devant
	// TOUT le trafic (frontend inclus, pas juste /api). Le garder ici aussi compterait /api 2× et
	// double-throttlerait → retiré. Le front-door throttle les scanners avant qu'ils n'atteignent
	// le backend, donc le tracking visiteurs reste protégé (un scanner 429 n'arrive jamais ici).

	// 6. Tracking visiteurs
	trackingMW := middleware.NewTracking(db, redisClient, cfg.SessionSecret)
	app.Use(trackingMW.Handler())

	// 7. Source de contenu CV : maiProFiles (source de vérité UNIQUE — plus de markdown local).
	// Gitea Stats Service — alimente le scoring vedette auto (commits par repo).
	// Créé AVANT le content provider qui le consomme.
	giteaStatsService := services.NewGiteaStatsService(redisClient, cfg.GiteaStatsURL, cfg.GiteaStatsToken, cfg.GiteaStatsUser)
	if giteaStatsService != nil {
		log.Info().Msg("Gitea stats service initialized")
	} else {
		log.Warn().Msg("Gitea stats service not available — GITEA_STATS_TOKEN not configured")
	}

	// GitLab Stats Service — commits d'un projet partagé (REMO) filtrés sur l'auteur, mergés dans les
	// gitstats. nil si GITLAB_STATS_TOKEN/PROJECT absents (feature désactivée proprement).
	gitlabStatsService := services.NewGitLabStatsService(redisClient, cfg.GitLabStatsURL, cfg.GitLabStatsToken, cfg.GitLabStatsProject, cfg.GitLabStatsAuthors)
	if gitlabStatsService != nil {
		log.Info().Msg("GitLab stats service initialized")
	} else {
		log.Warn().Msg("GitLab stats service not available — GITLAB_STATS_TOKEN/PROJECT not configured")
	}

	// Le provider fetch /experiences, /skills, /projects et cache 5 min.
	// Vedette = pins curés (maiprofiles) UNION top activité Gitea (via giteaStatsService).
	contentProvider := services.NewMaiProFilesContentProvider(giteaStatsService)

	// 8. Initialiser services (needed for analytics middleware)
	// LLM scoring via proxy Anthropic (optionnel — fallback tag-weight si non configuré)
	llmScoring := services.NewLLMScoringService(cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, redisClient)
	cvService := services.NewCVService(contentProvider, redisClient, llmScoring)
	analyticsService := services.NewAnalyticsService(db, redisClient)

	// DemoMetrics : générateur procédural d'analytics "vivantes" (seedé par les commits via
	// giteaStatsService, gaté par les vrais users). Activé par DEMO_METRICS=true|1. Injecté dans
	// l'analytics → le WS ET les endpoints héritent automatiquement du blend (point unique).
	demoEnabled := os.Getenv("DEMO_METRICS") == "true" || os.Getenv("DEMO_METRICS") == "1"
	analyticsService.SetDemoMetrics(services.NewDemoMetrics(giteaStatsService, redisClient, demoEnabled))
	if demoEnabled {
		log.Info().Msg("DemoMetrics ON — analytics synthétiques (seedées commits) activées")
	}

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
		letterGenerator = services.NewLetterGenerator(aiService, scraper, pdfLetterService, userProfile, contentProvider)
		log.Info().Msg("Letter generator service initialized")
	} else {
		log.Warn().Msg("Letter generator service not initialized - AI or scraper missing")
	}

	// GitHub services
	githubOAuthService := services.NewGitHubOAuthService(db, redisClient)
	githubSyncService := services.NewGitHubSyncService(db, redisClient)

	// Repo scanner service (replaces activity feed service)
	repoScanner := services.NewRepoScanner(redisClient, cfg.ReposDir)

	// Blog generator service
	blogGeneratorService := services.NewBlogGeneratorService(db, redisClient, aiService, repoScanner)

	// Profile detection services
	clearbitClient := services.NewClearbitClient(redisClient)
	uaParser := services.NewUserAgentParser()
	profileDetector := services.NewProfileDetectorService(db, redisClient, clearbitClient, uaParser)

	// PDF service for CV export (separate from letter PDF service)
	pdfService := services.NewPDFService()

	// CV tailoring : personnalisation par annonce avec réécriture LLM + stealth ATS
	var tailoringService *services.TailoringService
	if aiService != nil {
		tailoringService = services.NewTailoringService(cvService, aiService, pdfService)
		log.Info().Msg("CV tailoring service initialized")
	} else {
		log.Warn().Msg("CV tailoring service not available — AI not configured")
	}

	// CV generation : CV dynamique depuis offre d'emploi (texte ou URL)
	// Utilise le proxy Anthropic directement (même config que llm_scoring)
	cvGenerationService := services.NewCVGenerationService(contentProvider, cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, pdfService)
	if cvGenerationService != nil {
		log.Info().Msg("CV generation service initialized")
	} else {
		log.Warn().Msg("CV generation service not available — Anthropic credentials not configured")
	}

	// 7b. Gitea Stats Service : créé plus haut (consommé par le content provider pour la vedette).

	// 8. Initialiser handlers
	healthHandler := api.NewHealthHandler(db, redisClient)
	cvHandler := api.NewCVHandler(cvService, tailoringService, cvGenerationService, redisClient)
	analyticsHandler := api.NewAnalyticsHandler(analyticsService)
	lettersHandler := api.NewLettersHandler(db, redisClient, letterQueueService, letterGenerator, aiConfig.OwnerAPIKey)
	messagesHandler := api.NewMessagesHandler(db, redisClient, letterGenerator, aiConfig.OwnerAPIKey)
	githubHandler := api.NewGitHubHandler(githubOAuthService, githubSyncService)
	activityHandler := api.NewActivityHandler(repoScanner)
	// Client HTTP vers maiProFiles — utilisé par blogHandler pour le CRUD blog
	mpfClient := services.NewMaiProFilesClient()

	// blogHandler utilise mpfClient pour list/get/create/update/delete/publish
	// et blogGeneratorService uniquement pour la génération IA (GeneratePost)
	blogHandler := api.NewBlogHandler(mpfClient, blogGeneratorService, redisClient, aiConfig.OwnerAPIKey)
	timelineHandler := api.NewTimelineHandler(db, contentProvider)
	profileHandler := api.NewProfileHandler(db, redisClient, profileDetector)
	gitStatsHandler := api.NewGitStatsHandler(giteaStatsService, gitlabStatsService)
	swaggerHandler := api.NewSwaggerHandler()
	visitorHandler := api.NewVisitorHandler(db, redisClient, analyticsService)

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

	// Circuit-breaker coût : plafond quotidien GLOBAL de générations IA non-owner (toutes
	// sessions/IP confondues). Dernier garde-fou contre un abus distribué (rotation de
	// sessions/IP) qui brûlerait le budget token Claude. Configurable via AI_GLOBAL_DAILY_MAX,
	// défaut 200 — au-delà, toute génération non-owner renvoie 503 jusqu'à minuit UTC.
	aiGlobalDailyMax := 200
	if v := os.Getenv("AI_GLOBAL_DAILY_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			aiGlobalDailyMax = n
		}
	}

	// Middleware de rate-limiting IA — PARTAGÉ par tous les endpoints qui appellent Claude
	// (/letters/generate, /messages/generate, /cv/generate, /cv/tailor). Owner (X-Owner-Key)
	// bypass total. L'incrémentation des compteurs se fait dans chaque handler APRÈS succès.
	aiRateLimitMW := middleware.AIRateLimit(middleware.AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        5,                    // 5 générations/jour par session
		MaxPerDayPerIP:   3,                    // 3 générations/jour par IP réelle (anti-bypass incognito)
		CooldownDuration: 2 * time.Minute,      // cooldown entre 2 générations d'une même session
		GlobalDailyMax:   aiGlobalDailyMax,     // circuit-breaker coût (0 = désactivé)
		OwnerAPIKey:      aiConfig.OwnerAPIKey, // bypass total pour l'owner
		SessionSecret:    cfg.SessionSecret,    // valide aussi le cookie admin (maicivy_admin) = owner
	})

	// Routes CV (Phase 2) — /cv/generate & /cv/tailor passent derrière le rate-limit IA
	// (appellent Claude). indeed-outreach doit envoyer X-Owner-Key (bypass + tier Opus).
	cvHandler.RegisterRoutes(app, aiRateLimitMW)
	gitStatsHandler.RegisterRoutes(app)

	// Routes Letters avec rate limiting AI (Phase 3 - IMPLEMENTED)
	lettersGroup := apiV1.Group("/letters")
	lettersGroup.Post("/generate", aiRateLimitMW, lettersHandler.GenerateLetter)
	lettersGroup.Get("/job/:jobId", lettersHandler.GetJobStatus)
	lettersGroup.Get("/pair", lettersHandler.GetLetterPair) // ?company=Google
	lettersGroup.Get("/history", lettersHandler.GetHistory)
	lettersGroup.Get("/access/status", lettersHandler.GetAccessStatus)
	lettersGroup.Get("/ratelimit/status", lettersHandler.GetRateLimitStatus)
	lettersGroup.Get("/:id/pdf", lettersHandler.DownloadPDF)
	lettersGroup.Get("/:id", lettersHandler.GetLetter) // Must be last (catch-all)

	// Routes Messages plateforme (Malt, LinkedIn...) — génération sync
	messagesGroup := apiV1.Group("/messages")
	messagesGroup.Post("/generate", aiRateLimitMW, messagesHandler.GenerateMessage)

	// Routes Analytics (Phase 4 - IMPLEMENTED)
	analyticsHandler.RegisterRoutes(app)

	// WebSocket for analytics real-time (Phase 4 - IMPLEMENTED)
	wsHandler := websocket.NewAnalyticsWSHandler(analyticsService, redisClient)
	wsHandler.RegisterRoutes(app)

	// Routes GitHub (Phase 5 - IMPLEMENTED)
	githubHandler.RegisterRoutes(apiV1)

	// Routes Activity Feed (Auto-sync from ProjectTracker)
	activityHandler.RegisterRoutes(apiV1)

	// Routes Blog (Articles générés depuis commits)
	blogHandler.RegisterRoutes(apiV1)

	// Routes Admin — login owner (mot de passe → cookie maicivy_admin signé = privilèges owner).
	// Le cookie est reconnu par aiRateLimitMW (cf. VerifyAdminCookie). Désactivé si ADMIN_PASSWORD vide.
	adminHandler := api.NewAdminHandler(cfg.AdminPassword, cfg.SessionSecret)
	adminHandler.RegisterRoutes(apiV1)

	// Stats privées owner-only (GET /admin/stats) — coûts IA, sécurité/sus, analytics détaillées.
	adminStatsHandler := api.NewAdminStatsHandler(db, redisClient, cfg.SessionSecret)
	adminStatsHandler.RegisterRoutes(apiV1)

	// Chat agent : persistance des conversations owner-only (mémoire durable). Le streaming réutilise
	// /chat (owner → Opus + tools maiProFiles).
	adminChatHandler := api.NewAdminChatHandler(db, cfg.SessionSecret)
	adminChatHandler.RegisterRoutes(apiV1)

	// Routes Timeline (Phase 5 - IMPLEMENTED)
	apiV1.Get("/timeline", timelineHandler.GetTimeline)
	apiV1.Get("/timeline/categories", timelineHandler.GetCategories)
	apiV1.Get("/timeline/milestones", timelineHandler.GetMilestones)

	// Routes Profile (Phase 5 - IMPLEMENTED)
	apiV1.Get("/profile/detect", profileHandler.GetDetect)
	apiV1.Get("/profile/current", profileHandler.GetCurrentProfile)

	// Routes Chat portfolio (interface conversationnelle avec tool_use)
	portfolioService := services.NewPortfolioService()
	chatService := services.NewChatService(aiConfig, portfolioService, blogGeneratorService)
	chatHandler := api.NewChatHandler(chatService, aiConfig.OwnerAPIKey)
	chatGroup := apiV1.Group("/chat")
	chatGroup.Post("/stream", aiRateLimitMW, chatHandler.StreamChat)

	// Routes Visitor (Tracking & Access Gate)
	apiV1.Get("/visitors/check", visitorHandler.CheckVisitorStatus)
	apiV1.Get("/visitor/status", visitorHandler.GetVisitorStatus)
	apiV1.Post("/visitors/heartbeat", visitorHandler.Heartbeat)

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

	// Job 1.5: Visitor cleanup (every minute) - Nettoie les visiteurs inactifs > 5 minutes
	visitorCleanupJob := jobs.NewVisitorCleanupJob(analyticsService, 1*time.Minute)
	go visitorCleanupJob.Start(ctx)
	log.Info().Msg("Visitor cleanup job started")

	// Job 2: GitHub auto-sync (every 6 hours)
	githubAutoSyncJob := jobs.NewGitHubAutoSyncJob(db, githubSyncService)
	if err := githubAutoSyncJob.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start GitHub auto-sync job")
	} else {
		log.Info().Msg("GitHub auto-sync job started")
	}

	// Note: Activity sync job removed - now using on-demand repo scanning

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
	cancel()                 // Arrêter analytics cleanup job
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
