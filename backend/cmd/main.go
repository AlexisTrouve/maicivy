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
	"maicivy/internal/dossiervivant"
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
		giteaStatsService.SetExcludedRepos(cfg.GiteaStatsExclude)
		log.Info().Msg("Gitea stats service initialized")
	} else {
		log.Warn().Msg("Gitea stats service not available — GITEA_STATS_TOKEN not configured")
	}

	// GitLab Stats Service — commits d'un projet partagé (mission client) filtrés sur l'auteur, mergés dans les
	// gitstats. nil si GITLAB_STATS_TOKEN/PROJECT absents (feature désactivée proprement).
	gitlabStatsService := services.NewGitLabStatsService(redisClient, cfg.GitLabStatsURL, cfg.GitLabStatsToken, cfg.GitLabStatsProject, cfg.GitLabStatsAuthors, cfg.GitLabStatsRef)
	if gitlabStatsService != nil {
		log.Info().Msg("GitLab stats service initialized")
	} else {
		log.Warn().Msg("GitLab stats service not available — GITLAB_STATS_TOKEN/PROJECT not configured")
	}

	// Gitea tiers — commits d'Alexis sur un dépôt partagé hébergé hors de son Gitea (un jeu partenaire),
	// mergés comme GitLab. nil si PARTNER_GITEA_* absents.
	partnerGiteaService := services.NewPartnerGiteaStatsService(redisClient, cfg.PartnerGiteaURL, cfg.PartnerGiteaToken, cfg.PartnerGiteaRepo, cfg.PartnerGiteaRef, cfg.PartnerGiteaEmails)
	if partnerGiteaService != nil {
		log.Info().Msg("Partner Gitea stats service initialized")
	} else {
		log.Warn().Msg("Partner Gitea stats service not available — PARTNER_GITEA_* not configured")
	}

	// Le provider fetch /experiences, /skills, /projects et cache 5 min.
	// Vedette = pins curés (maiprofiles) UNION top activité Gitea (via giteaStatsService).
	contentProvider := services.NewMaiProFilesContentProvider(giteaStatsService)

	// 8. Initialiser services (needed for analytics middleware)
	// LLM scoring via proxy Anthropic (optionnel — fallback tag-weight si non configuré)
	// Modèle épinglé du CV (classement des projets + CV taillé), cf. config.AIConfig.CVModel.
	cvModel := config.LoadAIConfig().CVModel
	llmScoring := services.NewLLMScoringService(cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, cvModel, redisClient)
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
	cvGenerationService := services.NewCVGenerationService(contentProvider, cfg.AnthropicBaseURL, cfg.AnthropicAPIKey, cvModel, pdfService)
	if cvGenerationService != nil {
		log.Info().Msg("CV generation service initialized")
	} else {
		log.Warn().Msg("CV generation service not available — Anthropic credentials not configured")
	}

	// 7b. Gitea Stats Service : créé plus haut (consommé par le content provider pour la vedette).

	// Relai SMTP interne, partagé par les trois émetteurs de courrier (transfert mailbox, rappel
	// Malt, newsletter blog). Lu UNE seule fois ici, puis réutilisé : trois lectures indépendantes
	// de la même variable, c'est précisément ce qui avait permis à trois replis codés en dur de
	// survivre sans que personne ne les voie.
	// L'erreur n'est pas fatale ICI — elle ne l'est que dans la branche d'un émetteur réellement
	// configuré (cf. plus bas), pour qu'un poste de dev sans relai continue de démarrer.
	smtpRelayAddr, smtpRelayErr := config.SMTPRelayAddr()

	// Surveillance du courrier sortant : compte les envois par origine et alerte sur Discord quand
	// un envoi casse. Sans webhook configuré, les compteurs tournent quand même — c'est la
	// notification qui est optionnelle, pas la mesure.
	mailWatchConfig := config.LoadMailWatchConfig()
	mailWatch := services.NewMailWatch(
		redisClient,
		services.NewDiscordNotifier(mailWatchConfig.WebhookURL, mailWatchConfig.OwnerUserID),
	)
	if mailWatchConfig.WebhookURL == "" {
		log.Warn().Msg("Mail watch: aucune notification — DISCORD_WEBHOOK_URL non configuré")
	} else {
		log.Info().Int("seuil_blocage_h", mailWatchConfig.StuckHours).
			Int("heure_digest_utc", mailWatchConfig.DigestHourUTC).Msg("Mail watch initialized")
	}

	// Mailbox Service : ingestion IMAP de la boîte pro existante (où Malt notifie déjà) + dispatch
	// auto vers une adresse tierce fixe. nil si MAILBOX_IMAP_USER/MAILBOX_IMAP_APP_PASSWORD/
	// MAILBOX_FORWARD_TO absents (même convention que Gitea/GitLab stats) — le handler admin reste
	// enregistré pour la consultation, seul le job de polling ne démarre pas (cf. plus bas).
	mailboxConfig := config.LoadMailboxConfig()
	var mailboxService *services.MailboxService
	if mailboxConfig.Configured() {
		// Mailbox configurée mais relai absent = les mails seraient captés puis jamais transférés,
		// en silence. On refuse de démarrer : c'est le mode de panne qu'on vient de payer 13 jours.
		if smtpRelayErr != nil {
			log.Fatal().Err(smtpRelayErr).Msg("Mailbox configurée mais relai SMTP absent — refus de démarrer plutôt que de perdre les transferts en silence")
		}
		mailboxFetcher, err := services.NewGmailImapFetcher(
			mailboxConfig.ImapHost, mailboxConfig.ImapPort,
			mailboxConfig.ImapUser, mailboxConfig.ImapAppPassword, mailboxConfig.ImapFolder,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Mailbox service not available — connexion IMAP échouée")
		} else {
			mailboxAllowlist := services.ParseMailboxAllowlist(mailboxConfig.AllowedDomains)

			// Filtre de pertinence LLM (cf. services.mailbox_relevance.go) — juge si une opportunité
			// Malt captée correspond au profil avant transfert auto. nil (mêmes credentials absents
			// que le reste des features IA) → filtre désactivé, tout est transféré comme avant lui.
			mailboxRelevanceService := services.NewMailboxRelevanceService(
				cfg.AnthropicBaseURL, cfg.AnthropicAPIKey,
				services.NewPortfolioService(), mailboxConfig.RelevanceThreshold,
			)
			// IMPORTANT : ne JAMAIS passer mailboxRelevanceService (typé *MailboxRelevanceService,
			// potentiellement nil) directement à NewMailboxService — un pointeur nil affecté à un
			// paramètre d'interface produit une interface NON-nil (classique piège Go), et
			// `s.relevance == nil` dans MailboxService ne détecterait jamais ce cas → panic au premier
			// appel. On ne construit l'interface que si le service concret est réellement non-nil.
			var mailboxRelevance services.MailboxRelevanceEvaluator
			if mailboxRelevanceService != nil {
				mailboxRelevance = mailboxRelevanceService
				log.Info().Int("threshold", mailboxConfig.RelevanceThreshold).Msg("Mailbox relevance filter initialized")
			} else {
				log.Warn().Msg("Mailbox relevance filter not available — Anthropic credentials not configured (tout sera transféré sans filtrage)")
			}

			mailboxService = services.NewMailboxService(
				db, mailboxFetcher, mailboxAllowlist,
				smtpRelayAddr,
				mailboxConfig.ForwardTo, mailboxConfig.ForwardFromEmail, mailboxConfig.ForwardFromName,
				nil, // relay nil → services.SendViaRelay (le vrai relai)
				mailboxRelevance,
			)
			mailboxService.SetMailWatch(mailWatch)
			// Lien « ✓ 已处理 » + rappels : signés avec le SESSION_SECRET (domaine séparé par préfixe,
			// cf. ackSigPrefix). Secret vide (dev) → ni lien ni rappel, le transfert reste identique.
			mailboxService.SetAckLink(mailboxConfig.AckBaseURL, cfg.SessionSecret)
			log.Info().Msg("Mailbox service initialized")
		}
	} else {
		log.Warn().Msg("Mailbox service not available — MAILBOX_IMAP_USER/MAILBOX_IMAP_APP_PASSWORD/MAILBOX_FORWARD_TO not configured")
	}

	// Traduction à la demande des mails captés (cf. services.mailbox_translation.go) — indépendante de
	// mailboxConfig.Configured() : utile même sans IMAP actif tant que des mails existent déjà en DB
	// (backfill). nil (mêmes credentials Anthropic absentes que le reste des features IA) → handler
	// répond 503 sur /translation, jamais de panic (même piège nil-interface que mailboxRelevance ci-
	// dessus).
	mailboxTranslationService := services.NewMailboxTranslationService(cfg.AnthropicBaseURL, cfg.AnthropicAPIKey)
	var mailboxTranslator services.MailboxTranslator
	if mailboxTranslationService != nil {
		mailboxTranslator = mailboxTranslationService
	} else {
		log.Warn().Msg("Mailbox translation not available — Anthropic credentials not configured")
	}

	// 8. Initialiser handlers
	healthHandler := api.NewHealthHandler(db, redisClient)
	cvHandler := api.NewCVHandler(cvService, tailoringService, cvGenerationService, redisClient)
	analyticsHandler := api.NewAnalyticsHandler(analyticsService)
	lettersHandler := api.NewLettersHandler(db, redisClient, letterQueueService, letterGenerator, aiConfig.OwnerAPIKey, aiConfig.OwnerModel, aiConfig.ClaudeModel)
	messagesHandler := api.NewMessagesHandler(db, redisClient, letterGenerator, aiConfig.OwnerAPIKey, aiConfig.OwnerModel, aiConfig.ClaudeModel)
	githubHandler := api.NewGitHubHandler(githubOAuthService, githubSyncService)
	activityHandler := api.NewActivityHandler(repoScanner)
	// Client HTTP vers maiProFiles — utilisé par blogHandler pour le CRUD blog
	mpfClient := services.NewMaiProFilesClient()

	// blogHandler utilise mpfClient pour list/get/create/update/delete/publish
	// et blogGeneratorService uniquement pour la génération IA (GeneratePost)
	blogHandler := api.NewBlogHandler(mpfClient, blogGeneratorService, redisClient, db, aiConfig.OwnerAPIKey)
	blogHandler.SetMailWatch(mailWatch)
	timelineHandler := api.NewTimelineHandler(db, contentProvider)
	profileHandler := api.NewProfileHandler(db, redisClient, profileDetector)
	gitStatsHandler := api.NewGitStatsHandler(giteaStatsService, gitlabStatsService, partnerGiteaService)
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
	// Résolution des prospecteurs (cookie signé + compte actif relu en base à chaque requête).
	prospectAuth := api.NewProspectAuth(db, cfg.SessionSecret)

	// Plafonds visiteurs du budget "ai", partagés avec les générations EN FOND du CV adapté (cf.
	// cvContextService plus bas) : une seule source pour les deux chemins.
	const aiVisitorMaxPerDay, aiVisitorMaxPerDayPerIP = 5, 3
	aiRateLimitMW := middleware.AIRateLimit(middleware.AIRateLimitConfig{
		Redis:            redisClient,
		MaxPerDay:        aiVisitorMaxPerDay,      // 5 générations/jour par session
		MaxPerDayPerIP:   aiVisitorMaxPerDayPerIP, // 3 générations/jour par IP réelle (anti-bypass incognito)
		CooldownDuration: 2 * time.Minute,         // cooldown entre 2 générations d'une même session
		GlobalDailyMax:   aiGlobalDailyMax,        // circuit-breaker coût (0 = désactivé)
		OwnerAPIKey:      aiConfig.OwnerAPIKey,    // bypass total pour l'owner
		SessionSecret:    cfg.SessionSecret,       // valide aussi le cookie admin (maicivy_admin) = owner
		Prospects:        prospectAuth,            // prospecteur : plafond personnel, jamais owner
		ProspectDailyMax: cfg.ProspectDailyAIMax,
	})

	// Routes CV (Phase 2) — /cv/generate & /cv/tailor passent derrière le rate-limit IA
	// (appellent Claude). indeed-outreach doit envoyer X-Owner-Key (bypass + tier Opus).
	cvHandler.RegisterRoutes(app, aiRateLimitMW)
	gitStatsHandler.RegisterRoutes(app)

	// CV adapté au besoin exprimé dans le chat public ou un dossier vivant (cf. services/cv_context.go).
	// Budget = EXACTEMENT les plafonds visiteurs d'aiRateLimitMW (mêmes clés, mêmes nombres) : une
	// génération lancée en fond coûte autant qu'un POST /cv/generate et tire sur le même quota.
	// Générateur absent (IA non configurée) → interface nil explicite : un *CVGenerationService nil
	// emballé dans l'interface la rendrait non nil et ferait paniquer la première génération.
	var cvContextGen services.CVContextGenerator
	if cvGenerationService != nil {
		cvContextGen = cvGenerationService
	}
	cvContextService := services.NewCVContextService(redisClient, cvContextGen, middleware.AIBudget{
		Redis:          redisClient,
		MaxPerDay:      aiVisitorMaxPerDay,
		MaxPerDayPerIP: aiVisitorMaxPerDayPerIP,
		GlobalDailyMax: aiGlobalDailyMax,
	})
	var renderAdapted func(*services.AdaptiveCVResponse, string) ([]byte, error)
	if cvGenerationService != nil {
		renderAdapted = cvGenerationService.RenderDynamicPDF
	}
	cvHandler.SetContext(cvContextService, renderAdapted)

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

	// Routes Portfolio — les projets CURÉS (maiProFiles data/book.yaml), pas le catalogue.
	// Même source que le PDF du portfolio book : maicivy en rend les pages, bookgen en rend le PDF.
	// Cache Redis stale-while-revalidate devant maiProFiles (cf. services/portfolio_cache.go),
	// préchauffé en fond au démarrage : même le premier visiteur ne paie pas maiProFiles en direct.
	portfolioCache := services.NewPortfolioCache(mpfClient, redisClient)
	go portfolioCache.Warm(context.Background())
	api.NewPortfolioHandler(portfolioCache).RegisterRoutes(apiV1)

	// Route Montagne — les projets en réseau « sert à construire » (cf. services/montagne_service.go).
	// Volontairement SANS préchauffage : aucun appel à maiProFiles au démarrage ; le cache Redis
	// (stale-while-revalidate) se remplit à la première visite de chaque langue.
	montagneService := services.NewMontagneService(mpfClient, redisClient)
	api.NewMontagneHandler(montagneService).RegisterRoutes(apiV1)

	// PDF du dossier vivant — rendu A4 du dossier remis, sur le Chromium du CV. Publiques : le PDF
	// du REJEU (avec sa mention) et celui du run réel d'un visiteur, tous deux sur la copie SERVEUR
	// du dossier (jamais sur un dossier envoyé par le navigateur) ; la route owner reste avec /admin.
	// Icônes des projets — l'icon.png de leur dépôt Gitea, lu avec le jeton des stats (jamais exposé),
	// réduit et caché en Redis ; GITEA_STATS_EXCLUDE s'y applique (cf. services/project_icons.go).
	// nil sans jeton : monogrammes partout, routes vivantes.
	projectIcons := services.NewProjectIconService(mpfClient, redisClient, cfg.GiteaStatsURL, cfg.GiteaStatsToken, cfg.GiteaStatsUser, cfg.GiteaStatsExclude)
	api.NewProjectIconsHandler(projectIcons).RegisterRoutes(apiV1)

	// Dossiers remis par les runs réels : la copie serveur que le PDF public imprime (Redis, 24 h).
	dossierStore := services.NewDossierStore(redisClient)
	dossierPDF := services.NewDossierPDFService(services.NewDossierFactsSource(montagneService, mpfClient)).WithIcons(projectIcons)
	api.NewDossierPDFHandler(dossierPDF, dossierStore, cfg.SessionSecret, aiConfig.OwnerAPIKey).RegisterPublicRoutes(apiV1)

	// Panneau /admin — TOUTES ses routes passent par api.RegisterAdminSurface, derrière le garde de
	// rôle (défaut-refus pour les prospecteurs, cf. middleware.AdminRoleGuard) :
	//   - login owner (mot de passe → cookie maicivy_admin = privilèges owner, reconnu par
	//     aiRateLimitMW ; désactivé si ADMIN_PASSWORD vide) ;
	//   - stats privées, agent, mailbox (owner-only) ;
	//   - Composer : /compose/verify seulement si aiService existe (sans lui traduction et jugement
	//     sont impossibles) ; la persistance des ateliers, elle, est toujours là ;
	//   - comptes prospecteurs (invitation, login, gestion, attribution).
	var composeService *services.ComposeService
	// Dossier vivant : runs réels de l'équipe d'agents, sur la montagne et les fiches. La route
	// PUBLIQUE (tout visiteur, sans connexion, sans quota — décision d'Alexis) est posée ici, HORS de
	// /admin et de son garde de rôle ; la route owner suit la surface /admin.
	var dossierVivant *dossiervivant.Service
	if aiService != nil {
		dossierVivant = dossiervivant.NewService(aiService,
			dossiervivant.NewLiveSources(montagneService, mpfClient),
			dossiervivant.Options{Model: aiConfig.DossierModel, SchemaModel: aiConfig.DossierSchemaModel})
		dossierVivantHandler := api.NewDossierVivantHandler(dossierVivant, dossierStore, cfg.SessionSecret, aiConfig.OwnerAPIKey)
		// Le besoin saisi prépare aussi le CV adapté du visiteur (cf. cvContextService).
		dossierVivantHandler.SetNeedRecorder(cvContextService)
		dossierVivantHandler.RegisterPublicRoutes(apiV1)
		// Le Composer contrôle les liens vers les pages projet (lien mort = case rouge) : il lit la
		// même liste que l'inventaire de l'agent, le livre du portfolio.
		composeService = services.NewComposeService(aiService)
		composeService.SetPortfolioPages(services.NewPagesPortfolio(portfolioCache))
	}
	prospectsHandler := api.RegisterAdminSurface(apiV1, api.AdminSurfaceDeps{
		DB: db, Redis: redisClient, SessionSecret: cfg.SessionSecret, AdminPassword: cfg.AdminPassword,
		ComposeService: composeService, MailboxService: mailboxService, MailboxTranslator: mailboxTranslator,
		// Plafond des vérifications Composer d'un prospecteur, réglé par PROSPECT_VERIFY_DAILY_MAX.
		// Levé (-1) par défaut depuis le 2026-10-15 ; valeur de référence pour le rétablir : 3× le
		// budget principal (la vérification part après chaque tour, 1 à 2 par génération).
		ProspectVerifyDailyMax: cfg.ProspectVerifyDailyMax,
		DossierVivant:          dossierVivant,
		DossierPDF:             dossierPDF,
		OwnerAPIKey:            aiConfig.OwnerAPIKey,
	})
	// Attribution des CV générés par un prospecteur.
	cvHandler.SetProspectCVRecorder(prospectsHandler.RecordCV)

	// Routes Timeline (Phase 5 - IMPLEMENTED)
	apiV1.Get("/timeline", timelineHandler.GetTimeline)
	apiV1.Get("/timeline/categories", timelineHandler.GetCategories)
	apiV1.Get("/timeline/milestones", timelineHandler.GetMilestones)

	// Routes Profile (Phase 5 - IMPLEMENTED)
	apiV1.Get("/profile/detect", profileHandler.GetDetect)
	apiV1.Get("/profile/current", profileHandler.GetCurrentProfile)

	// Routes Chat portfolio (interface conversationnelle avec tool_use)
	// Budget dédié (PAS aiRateLimitMW/CV-lettres) : un chat conversationnel a un usage très différent
	// d'une génération one-shot — épuiser son quota CV ne doit plus jamais bloquer une conversation
	// en cours (bug découvert : chat.go n'incrémentait aucun compteur mais héritait quand même des
	// blocages CV/lettres via le namespace partagé "ratelimit:ai:..."). KeyPrefix "chat" isole
	// complètement les compteurs ; le circuit-breaker coût GLOBAL (aiGlobalDailyMax), lui, reste
	// partagé à dessein — cf. doc de AIRateLimitConfig.KeyPrefix.
	chatRateLimitMW := middleware.AIRateLimit(middleware.AIRateLimitConfig{
		Redis:          redisClient,
		KeyPrefix:      "chat",
		MaxPerDay:      20, // 20 messages/jour par session — budget dédié conversationnel
		MaxPerDayPerIP: 15, // 15/jour par IP — filet anti-incognito, volontairement < MaxPerDay
		GlobalDailyMax: aiGlobalDailyMax,
		OwnerAPIKey:    aiConfig.OwnerAPIKey,
		SessionSecret:  cfg.SessionSecret,
		// Même budget prospecteur que CV : la clé Redis est par personne, pas par feature.
		Prospects:        prospectAuth,
		ProspectDailyMax: cfg.ProspectDailyAIMax,
	})

	portfolioService := services.NewPortfolioService()
	chatService := services.NewChatService(aiConfig, portfolioService, blogGeneratorService)
	// Inventaire marqué [page] : l'agent sait quels projets ont une page à lier (cf. PagesPortfolio).
	chatService.SetPortfolioPages(services.NewPagesPortfolio(portfolioCache))
	// L'ancrage tarifaire n'est transmis qu'aux tours compose (cf. StreamChat) — le chat public n'a
	// aucune raison de connaître les tarifs pratiqués.
	chatHandler := api.NewChatHandler(chatService, redisClient, aiConfig.OwnerAPIKey, aiConfig.OwnerModel, aiConfig.ClaudeModel,
		cfg.ComposeRateCurrent, cfg.ComposeRateTarget, cfg.ComposeRateFloor)
	// Outil note_visitor_need du chat public : le besoin exprimé prépare le CV adapté du visiteur.
	chatHandler.SetNeedRecorder(cvContextService)
	chatGroup := apiV1.Group("/chat")
	chatGroup.Post("/stream", chatRateLimitMW, chatHandler.StreamChat)

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

	// Job 4: Mailbox poll (ingestion IMAP + retry transferts) — démarre SEULEMENT si le service a pu
	// être construit (credentials présents ET connexion IMAP réussie, cf. plus haut).
	var mailboxPollJob *jobs.MailboxPollJob
	if mailboxService != nil {
		mailboxPollJob = jobs.NewMailboxPollJob(mailboxService, time.Duration(mailboxConfig.PollIntervalSeconds)*time.Second)
		go mailboxPollJob.Start(ctx)
		log.Info().Msg("Mailbox poll job started")
	}

	// Job 6: veille du courrier sortant — repère les transferts bloqués et publie le récapitulatif
	// quotidien. Indépendant des émetteurs : il interroge l'ÉTAT (du courrier attend-il depuis trop
	// longtemps ?), ce qui reste vrai même quand plus personne ne retente d'envoyer.
	if mailWatchJob := jobs.NewMailWatchJob(
		db, mailWatch,
		time.Duration(mailWatchConfig.StuckHours)*time.Hour,
		mailWatchConfig.DigestHourUTC,
	); mailWatchJob != nil {
		go mailWatchJob.Start(ctx)
	}

	// Job 5: rappel disponibilité Malt (2x/semaine par défaut) — indépendant de l'IMAP, juste un
	// envoi SMTP périodique. Même relai interne que mailbox/newsletter (cf. config.SMTPRelayAddr).
	maltReminderConfig := config.LoadMaltReminderConfig()
	var maltReminderJob *jobs.MaltReminderJob
	if maltReminderConfig.Configured() {
		// Même raison que pour la mailbox : un rappel configuré sans relai ne partirait jamais.
		if smtpRelayErr != nil {
			log.Fatal().Err(smtpRelayErr).Msg("Rappel Malt configuré mais relai SMTP absent — refus de démarrer")
		}
		maltReminderService := services.NewMaltReminderService(
			redisClient, smtpRelayAddr,
			maltReminderConfig.To, maltReminderConfig.FromEmail, maltReminderConfig.FromName,
			maltReminderConfig.Days, maltReminderConfig.HourUTC,
			nil, // relay nil → services.SendViaRelay (le vrai relai)
		)
		maltReminderService.SetMailWatch(mailWatch)
		maltReminderJob = jobs.NewMaltReminderJob(maltReminderService)
		go maltReminderJob.Start(ctx)
		log.Info().Msg("Malt reminder job started")
	} else {
		log.Warn().Msg("Malt reminder not available — MALT_REMINDER_TO not configured")
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
	if mailboxPollJob != nil {
		mailboxPollJob.Stop()
		if err := mailboxService.Close(); err != nil {
			log.Warn().Err(err).Msg("mailbox: échec fermeture connexion IMAP")
		}
	}
	if maltReminderJob != nil {
		maltReminderJob.Stop()
	}

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
