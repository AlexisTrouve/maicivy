# 04. Backend Middlewares

## 📋 Métadonnées

- **Phase:** 1
- **Priorité:** 🟡 HAUTE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 02. BACKEND_FOUNDATION.md, 03. DATABASE_SCHEMA.md
- **Temps estimé:** 2-3 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter la couche middleware complète du backend Go/Fiber pour gérer :
- La sécurité CORS
- Le tracking des visiteurs avec détection de profils
- Le rate limiting avec règles spécifiques pour l'IA
- Le logging et tracing des requêtes
- La récupération des panics

Ces middlewares constituent la fondation de la sécurité, de l'observabilité et du contrôle d'accès du système.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
Request → CORS → Recovery → RequestID → Logger → Tracking → RateLimiting → Handler
             ↓                                        ↓            ↓
          Security                               Redis        Redis
                                                  (visits)   (limits)
                                                     ↓
                                              PostgreSQL
                                               (visitors)
```

### Design Decisions

**1. Ordre des Middlewares**
```go
app.Use(cors())         // 1. Sécurité CORS en premier
app.Use(recover())      // 2. Récupération panic
app.Use(requestid())    // 3. ID unique pour tracing
app.Use(logger())       // 4. Logging avec request ID
app.Use(tracking())     // 5. Tracking visiteurs
app.Use(ratelimit())    // 6. Rate limiting (dépend tracking)
```

**2. Redis pour Performance**
- Compteurs de visites en mémoire (rapide)
- Rate limiting avec TTL automatique
- Évite surcharge PostgreSQL

**3. Tracking Cookie-Based**
- Cookie HTTPOnly + Secure
- Session ID unique (UUID)
- Pas de données sensibles dans cookie

**4. Détection Profil Multi-Critères**
- User-Agent analysis
- IP lookup (entreprise)
- Patterns de navigation
- Score de confiance

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Framework web
go get github.com/gofiber/fiber/v2

# Redis client
go get github.com/redis/go-redis/v9

# UUID pour session ID
go get github.com/google/uuid

# User-Agent parsing
go get github.com/mileusna/useragent

# Logger structuré
go get github.com/rs/zerolog

# IP lookup (optionnel)
go get github.com/oschwald/geoip2-golang
```

### Services Externes

- **Redis**: Cache et rate limiting
- **PostgreSQL**: Stockage visiteurs
- **Clearbit API** (optionnel): Enrichissement données IP → entreprise

---

## 🔨 Implémentation

### Étape 1: CORS Middleware

**Fichier:** `backend/internal/middleware/cors.go`

**Description:** Configuration fine CORS pour autoriser le frontend et gérer les credentials.

**Code:**

```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS configure la politique CORS de l'application
func CORS(allowedOrigins []string) fiber.Handler {
	return cors.New(cors.Config{
		// Origins autorisées (depuis config)
		AllowOrigins: allowedOrigins, // Ex: "http://localhost:3000,https://maicivy.com"

		// Méthodes autorisées
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",

		// Headers autorisés
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",

		// Headers exposés au client
		ExposeHeaders: "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",

		// Autoriser credentials (cookies)
		AllowCredentials: true,

		// Cache preflight requests (24h)
		MaxAge: 86400,
	})
}
```

**Explications:**
- `AllowCredentials: true` permet l'envoi de cookies
- `AllowOrigins` doit être spécifique (pas `*` si credentials)
- `ExposeHeaders` expose les headers de rate limiting au client
- `MaxAge` réduit les requêtes OPTIONS préflight

---

### Étape 2: Recovery Middleware

**Fichier:** `backend/internal/middleware/recovery.go`

**Description:** Récupération des panics pour éviter crash du serveur.

**Code:**

```go
package middleware

import (
	"fmt"
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
					"error": "Internal server error",
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
```

**Explications:**
- `defer recover()` capture les panics
- Log complet avec stack trace pour debugging
- Retourne erreur 500 propre au client
- Inclut request ID pour traçabilité

---

### Étape 3: Request ID Middleware

**Fichier:** `backend/internal/middleware/requestid.go`

**Description:** Génère un ID unique par requête pour le tracing.

**Code:**

```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID ajoute un ID unique à chaque requête
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Vérifier si header X-Request-ID existe déjà (ex: proxy)
		requestID := c.Get("X-Request-ID")

		// Sinon, générer nouveau UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Stocker dans context local
		c.Locals("requestid", requestID)

		// Ajouter au header de réponse
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}
```

**Explications:**
- Utilise UUID v4 (unique, random)
- Préserve request ID du proxy si présent
- Stocké dans `c.Locals()` pour accès dans autres middlewares
- Retourné au client via header

---

### Étape 4: Logger Middleware

**Fichier:** `backend/internal/middleware/logger.go`

**Description:** Log structuré de chaque requête HTTP.

**Code:**

```go
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

		// Log complet
		logEvent.
			Str("request_id", c.Locals("requestid").(string)).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Int("status", c.Response().StatusCode()).
			Dur("duration_ms", duration).
			Str("user_agent", c.Get("User-Agent")).
			Int("size", len(c.Response().Body())).
			Msg("HTTP request")

		return err
	}
}
```

**Explications:**
- Log structuré JSON (zerolog)
- Inclut métriques: durée, status, taille
- Level adapté: Info (2xx), Warn (4xx), Error (5xx)
- Request ID pour corrélation

---

### Étape 5: Tracking Middleware

**Fichier:** `backend/internal/middleware/tracking.go`

**Description:** Tracking des visiteurs avec compteur de visites et détection de profil.

**Code:**

```go
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mileusna/useragent"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

const (
	SessionCookieName = "maicivy_session"
	SessionTTL        = 30 * 24 * time.Hour // 30 jours
	VisitorKeyPrefix  = "visitor:"
)

type TrackingMiddleware struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewTracking(db *gorm.DB, redisClient *redis.Client) *TrackingMiddleware {
	return &TrackingMiddleware{
		db:    db,
		redis: redisClient,
	}
}

func (tm *TrackingMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// 1. Récupérer ou créer session ID
		sessionID := c.Cookies(SessionCookieName)
		if sessionID == "" {
			sessionID = uuid.New().String()

			// Set cookie
			c.Cookie(&fiber.Cookie{
				Name:     SessionCookieName,
				Value:    sessionID,
				Expires:  time.Now().Add(SessionTTL),
				HTTPOnly: true,
				Secure:   true, // HTTPS only en production
				SameSite: "Lax",
			})
		}

		// 2. Incrémenter compteur de visites dans Redis
		visitCountKey := fmt.Sprintf("%s%s:count", VisitorKeyPrefix, sessionID)
		visitCount, err := tm.redis.Incr(ctx, visitCountKey).Result()
		if err != nil {
			log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to increment visit count")
			visitCount = 1
		}

		// Set TTL si première visite
		if visitCount == 1 {
			tm.redis.Expire(ctx, visitCountKey, SessionTTL)
		}

		// 3. Détection de profil
		profileDetected := tm.detectProfile(c)

		// Stocker profil dans Redis (cache)
		if profileDetected != "" {
			profileKey := fmt.Sprintf("%s%s:profile", VisitorKeyPrefix, sessionID)
			tm.redis.Set(ctx, profileKey, profileDetected, SessionTTL)
		}

		// 4. Stocker dans context pour utilisation dans handlers
		c.Locals("session_id", sessionID)
		c.Locals("visit_count", visitCount)
		c.Locals("profile_detected", profileDetected)

		// 5. Enregistrer/update visiteur dans PostgreSQL (async)
		go tm.saveVisitor(sessionID, c, visitCount, profileDetected)

		return c.Next()
	}
}

// detectProfile analyse User-Agent et IP pour détecter recruteurs/profils cibles
func (tm *TrackingMiddleware) detectProfile(c *fiber.Ctx) string {
	userAgentStr := c.Get("User-Agent")
	ip := c.IP()

	// Parse User-Agent
	ua := useragent.Parse(userAgentStr)

	// Patterns LinkedIn
	if strings.Contains(strings.ToLower(userAgentStr), "linkedin") {
		return "linkedin_bot"
	}

	// Patterns recruteurs (LinkedIn Sales Navigator, etc.)
	recruiterPatterns := []string{
		"sales navigator",
		"recruiter",
		"talent",
		"hiring",
	}

	userAgentLower := strings.ToLower(userAgentStr)
	for _, pattern := range recruiterPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return "recruiter"
		}
	}

	// Détection entreprise via IP (optionnel - nécessite API Clearbit ou GeoIP)
	// company := tm.lookupCompany(ip)
	// if company != "" {
	//     return "corporate:" + company
	// }

	// Desktop professionnel (pas mobile)
	if ua.Desktop && !ua.Mobile {
		return "professional"
	}

	return "" // Pas de profil spécifique détecté
}

// saveVisitor enregistre ou met à jour le visiteur dans PostgreSQL
func (tm *TrackingMiddleware) saveVisitor(sessionID string, c *fiber.Ctx, visitCount int64, profile string) {
	// Hash IP pour privacy
	ipHash := hashIP(c.IP())

	visitor := models.Visitor{
		SessionID:       sessionID,
		IPHash:          ipHash,
		UserAgent:       c.Get("User-Agent"),
		VisitCount:      int(visitCount),
		ProfileDetected: profile,
		LastVisit:       time.Now(),
	}

	// Upsert (insert ou update si existe)
	result := tm.db.Where("session_id = ?", sessionID).
		Assign(visitor).
		FirstOrCreate(&visitor)

	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("session_id", sessionID).
			Msg("Failed to save visitor to database")
	}
}

// hashIP hash l'IP pour respecter RGPD/privacy
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}
```

**Explications:**
- **Cookie session**: UUID unique, HTTPOnly, Secure
- **Redis counter**: Incrémentation atomique, TTL 30j
- **Détection profil**: User-Agent patterns (LinkedIn, recruteurs)
- **PostgreSQL async**: Enregistrement sans bloquer requête
- **Privacy**: IP hashée (SHA-256)

---

### Étape 6: Rate Limiting Middleware

**Fichier:** `backend/internal/middleware/ratelimit.go`

**Description:** Rate limiting basé Redis avec règles spécifiques pour l'IA.

**Code:**

```go
package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// Rate limits globaux
	GlobalRateLimit    = 100              // requêtes par IP
	GlobalRateWindow   = 1 * time.Minute  // fenêtre de temps

	// Rate limits IA (depuis PROJECT_SPEC.md)
	AIGenerationsLimit = 5                // max générations par session
	AIGenerationsWindow = 24 * time.Hour  // par jour
	AICooldown         = 2 * time.Minute  // cooldown entre générations
)

type RateLimitMiddleware struct {
	redis *redis.Client
}

func NewRateLimit(redisClient *redis.Client) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redis: redisClient,
	}
}

// Global rate limiting par IP
func (rlm *RateLimitMiddleware) Global() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		ip := c.IP()

		// Clé Redis pour rate limiting global
		key := fmt.Sprintf("ratelimit:global:%s", ip)

		// Incrémenter compteur
		count, err := rlm.redis.Incr(ctx, key).Result()
		if err != nil {
			log.Error().Err(err).Msg("Redis incr failed for rate limit")
			return c.Next() // Fail open (ne pas bloquer si Redis down)
		}

		// Set TTL si première requête dans fenêtre
		if count == 1 {
			rlm.redis.Expire(ctx, key, GlobalRateWindow)
		}

		// Vérifier limite
		if count > GlobalRateLimit {
			// Headers de rate limiting
			c.Set("X-RateLimit-Limit", strconv.Itoa(GlobalRateLimit))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", strconv.Itoa(int(GlobalRateWindow.Seconds())))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests",
				"message": fmt.Sprintf("Rate limit exceeded. Max %d requests per minute.", GlobalRateLimit),
				"retry_after": GlobalRateWindow.Seconds(),
			})
		}

		// Headers de rate limiting
		c.Set("X-RateLimit-Limit", strconv.Itoa(GlobalRateLimit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(GlobalRateLimit-int(count)))

		return c.Next()
	}
}

// AI rate limiting par session (règles strictes)
func (rlm *RateLimitMiddleware) AI() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// Récupérer session ID depuis tracking middleware
		sessionID := c.Locals("session_id")
		if sessionID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No session found",
			})
		}

		sessionIDStr := sessionID.(string)

		// 1. Vérifier cooldown (2 minutes entre générations)
		cooldownKey := fmt.Sprintf("ratelimit:ai:cooldown:%s", sessionIDStr)
		exists, err := rlm.redis.Exists(ctx, cooldownKey).Result()
		if err != nil {
			log.Error().Err(err).Msg("Redis exists failed for cooldown check")
		} else if exists > 0 {
			ttl, _ := rlm.redis.TTL(ctx, cooldownKey).Result()

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Cooldown active",
				"message": "Please wait before generating another letter",
				"retry_after": int(ttl.Seconds()),
			})
		}

		// 2. Vérifier limite journalière (5 générations/jour)
		dailyKey := fmt.Sprintf("ratelimit:ai:daily:%s", sessionIDStr)
		count, err := rlm.redis.Get(ctx, dailyKey).Int64()
		if err != nil && err != redis.Nil {
			log.Error().Err(err).Msg("Redis get failed for daily limit")
		}

		if count >= AIGenerationsLimit {
			ttl, _ := rlm.redis.TTL(ctx, dailyKey).Result()

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Daily limit reached",
				"message": fmt.Sprintf("Maximum %d letter generations per day reached", AIGenerationsLimit),
				"retry_after": int(ttl.Seconds()),
			})
		}

		// 3. Incrémenter compteur journalier
		newCount, err := rlm.redis.Incr(ctx, dailyKey).Result()
		if err != nil {
			log.Error().Err(err).Msg("Redis incr failed for daily counter")
			return c.Next() // Fail open
		}

		// Set TTL si première génération du jour
		if newCount == 1 {
			rlm.redis.Expire(ctx, dailyKey, AIGenerationsWindow)
		}

		// 4. Activer cooldown
		rlm.redis.Set(ctx, cooldownKey, "1", AICooldown)

		// Headers de rate limiting
		c.Set("X-RateLimit-AI-Limit", strconv.Itoa(AIGenerationsLimit))
		c.Set("X-RateLimit-AI-Remaining", strconv.Itoa(AIGenerationsLimit-int(newCount)))

		return c.Next()
	}
}
```

**Explications:**
- **Global limit**: 100 req/min par IP (protection DDoS)
- **AI daily limit**: 5 générations/jour/session (contrôle coûts)
- **AI cooldown**: 2 min entre générations (évite spam)
- **Headers standards**: X-RateLimit-* pour client
- **Fail open**: Si Redis down, autoriser (pas bloquer tout le site)

---

### Étape 7: Integration dans main.go

**Fichier:** `backend/cmd/main.go`

**Description:** Enregistrement des middlewares dans l'ordre correct.

**Code:**

```go
package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"maicivy/internal/config"
	"maicivy/internal/database"
	"maicivy/internal/middleware"
)

func main() {
	// Charger configuration
	cfg := config.Load()

	// Connexions DB
	db := database.ConnectPostgres(cfg.DatabaseURL)
	redisClient := database.ConnectRedis(cfg.RedisURL)

	// Créer app Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	// ===== MIDDLEWARES (ORDRE IMPORTANT) =====

	// 1. CORS (sécurité)
	app.Use(middleware.CORS(cfg.AllowedOrigins))

	// 2. Recovery (panic handling)
	app.Use(middleware.Recovery())

	// 3. Request ID (tracing)
	app.Use(middleware.RequestID())

	// 4. Logger (avec request ID)
	app.Use(middleware.Logger())

	// 5. Tracking visiteurs
	trackingMW := middleware.NewTracking(db, redisClient)
	app.Use(trackingMW.Handler())

	// 6. Rate limiting global
	rateLimitMW := middleware.NewRateLimit(redisClient)
	app.Use(rateLimitMW.Global())

	// ===== ROUTES =====

	// Health check (sans rate limit AI)
	app.Get("/health", healthHandler)

	// API routes (rate limit AI appliqué sélectivement)
	api := app.Group("/api")

	// CV routes (pas de rate limit AI)
	api.Get("/cv", getCVHandler)
	api.Get("/cv/themes", getThemesHandler)

	// Letters routes (AVEC rate limit AI)
	lettersGroup := api.Group("/letters")
	lettersGroup.Use(rateLimitMW.AI()) // Rate limit AI ici
	lettersGroup.Post("/generate", generateLetterHandler)
	lettersGroup.Get("/:id", getLetterHandler)

	// Démarrer serveur
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen(":" + port))
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"request_id": c.Locals("requestid"),
	})
}

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Placeholder handlers
func getCVHandler(c *fiber.Ctx) error { return c.SendString("CV") }
func getThemesHandler(c *fiber.Ctx) error { return c.SendString("Themes") }
func generateLetterHandler(c *fiber.Ctx) error { return c.SendString("Generate") }
func getLetterHandler(c *fiber.Ctx) error { return c.SendString("Letter") }
```

**Explications:**
- Ordre middlewares respecté (CORS → Recovery → RequestID → Logger → Tracking → RateLimit)
- Rate limit AI appliqué UNIQUEMENT sur routes `/api/letters`
- Rate limit global sur toutes les routes
- Custom error handler avec request ID

---

## 🧪 Tests

### Tests Unitaires: Tracking Middleware

**Fichier:** `backend/internal/middleware/tracking_test.go`

```go
package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/database"
	"maicivy/internal/models"
)

func TestTrackingMiddleware_NewVisitor(t *testing.T) {
	// Setup
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient)

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	// Test
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Vérifier cookie créé
	cookies := resp.Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, SessionCookieName, cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
}

func TestTrackingMiddleware_ReturningVisitor(t *testing.T) {
	// Setup
	db, redisClient := setupTestDB(t)
	trackingMW := NewTracking(db, redisClient)

	app := fiber.New()
	app.Use(trackingMW.Handler())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"visit_count": c.Locals("visit_count"),
		})
	})

	// Première visite
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, _ := app.Test(req1)
	sessionCookie := resp1.Cookies()[0]

	// Deuxième visite (avec cookie)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(sessionCookie)
	resp2, err := app.Test(req2)
	require.NoError(t, err)

	// Parser response
	var body map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&body)

	// Assertions
	assert.Equal(t, float64(2), body["visit_count"])
}

func TestDetectProfile_LinkedIn(t *testing.T) {
	tm := &TrackingMiddleware{}

	app := fiber.New()
	c := app.AcquireCtx(&fiber.Ctx{})
	defer app.ReleaseCtx(c)

	c.Request().Header.Set("User-Agent", "LinkedInBot/1.0")

	profile := tm.detectProfile(c)
	assert.Equal(t, "linkedin_bot", profile)
}
```

### Tests Integration: Rate Limiting

**Fichier:** `backend/internal/middleware/ratelimit_test.go`

```go
package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit_AI_DailyLimit(t *testing.T) {
	// Setup
	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Test: 5 générations ok
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/generate", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Wait for cooldown
		time.Sleep(2 * time.Minute)
	}

	// Test: 6ème génération bloquée
	req := httptest.NewRequest("POST", "/generate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimit_AI_Cooldown(t *testing.T) {
	// Setup
	redisClient := setupTestRedis(t)
	rlm := NewRateLimit(redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return c.Next()
	})
	app.Use(rlm.AI())
	app.Post("/generate", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Première génération ok
	req1 := httptest.NewRequest("POST", "/generate", nil)
	resp1, _ := app.Test(req1)
	assert.Equal(t, 200, resp1.StatusCode)

	// Deuxième génération immédiate bloquée (cooldown)
	req2 := httptest.NewRequest("POST", "/generate", nil)
	resp2, _ := app.Test(req2)
	assert.Equal(t, 429, resp2.StatusCode)

	// Après 2min, ok
	time.Sleep(2 * time.Minute)
	req3 := httptest.NewRequest("POST", "/generate", nil)
	resp3, _ := app.Test(req3)
	assert.Equal(t, 200, resp3.StatusCode)
}
```

### Commandes

```bash
# Tests unitaires
go test -v ./internal/middleware/...

# Tests avec coverage
go test -v -cover ./internal/middleware/...

# Tests integration (nécessite Redis/PostgreSQL)
go test -v -tags=integration ./internal/middleware/...

# Benchmark
go test -bench=. ./internal/middleware/...
```

---

## ⚠️ Points d'Attention

### Pièges à Éviter

- ⚠️ **Ordre des middlewares**: CORS doit être AVANT tout le reste, sinon preflight OPTIONS échoue
- ⚠️ **Cookie Secure flag**: Doit être `true` en production (HTTPS), `false` en dev local (HTTP)
- ⚠️ **Rate limit fail open vs fail closed**: Décider si Redis down = bloquer tout ou autoriser (actuellement fail open)
- ⚠️ **IP hashing**: Ne pas stocker IP en clair (RGPD), toujours hasher
- ⚠️ **Session ID rotation**: Cookie long-lived (30j) peut être problème sécurité, considérer rotation

### Edge Cases

- **Visitor sans cookie support**: Fallback sur IP hash (moins fiable)
- **Redis down**: Tracking continue en PostgreSQL, rate limiting désactivé (fail open)
- **Cluster Redis**: Utiliser Redis Cluster ou Sentinel pour HA
- **Clock skew**: TTL Redis peut varier si horloges serveurs désynchronisées

### Optimisations

- 💡 **Redis pipelining**: Grouper commandes Redis pour réduire latency
- 💡 **Batch PostgreSQL writes**: Buffer writes et flush toutes les N secondes
- 💡 **IP lookup cache**: Cacher résultats lookup IP→entreprise (TTL 7j)
- 💡 **User-Agent parsing**: Cacher résultats parsing (map thread-safe)

---

## 📚 Ressources

### Documentation Officielle

- [Fiber Middleware](https://docs.gofiber.io/api/middleware)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [zerolog Logging](https://github.com/rs/zerolog)
- [GORM Documentation](https://gorm.io/docs/)

### Patterns & Best Practices

- [Rate Limiting Strategies](https://redis.io/glossary/rate-limiting/)
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [OWASP: Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [CORS Best Practices](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)

### Alternatives Considérées

- **Rate limiting**: go-redis vs Tollbooth → go-redis plus flexible
- **User-Agent parsing**: useragent vs ua-parser-go → useragent plus simple
- **IP lookup**: Clearbit vs IPinfo vs MaxMind GeoIP2 → GeoIP2 gratuit, offline

---

## ✅ Checklist de Complétion

### Implémentation

- [ ] `cors.go` implémenté avec configuration fine
- [ ] `recovery.go` implémenté avec stack trace logging
- [ ] `requestid.go` implémenté avec UUID
- [ ] `logger.go` implémenté avec zerolog structuré
- [ ] `tracking.go` implémenté avec:
  - [ ] Cookie session management
  - [ ] Redis visit counter
  - [ ] Profile detection (User-Agent patterns)
  - [ ] PostgreSQL visitor storage (async)
  - [ ] IP hashing (privacy)
- [ ] `ratelimit.go` implémenté avec:
  - [ ] Global rate limiting (100/min par IP)
  - [ ] AI daily limit (5 générations/jour)
  - [ ] AI cooldown (2 min entre générations)
  - [ ] Headers X-RateLimit-*
- [ ] Integration dans `main.go` avec ordre correct

### Models

- [ ] `models.Visitor` défini dans database schema (doc 03)
- [ ] Migration PostgreSQL pour table `visitors`

### Configuration

- [ ] Variables environnement:
  - [ ] `ALLOWED_ORIGINS` (CORS)
  - [ ] `COOKIE_SECURE` (true/false selon env)
  - [ ] `RATE_LIMIT_GLOBAL` (configurable)
  - [ ] `RATE_LIMIT_AI_DAILY` (configurable)

### Tests

- [ ] Tests unitaires tracking middleware (> 80% coverage)
- [ ] Tests unitaires rate limiting (edge cases)
- [ ] Tests integration avec Redis
- [ ] Tests integration avec PostgreSQL
- [ ] Benchmarks performance (throughput middlewares)

### Documentation

- [ ] Commentaires code (GoDoc format)
- [ ] README middleware (usage examples)
- [ ] Diagramme séquence tracking flow
- [ ] Documentation rate limit rules pour frontend

### Sécurité

- [ ] Review CORS origins (pas wildcard)
- [ ] Cookie flags corrects (HttpOnly, Secure, SameSite)
- [ ] IP hashing (RGPD compliance)
- [ ] Rate limiting efficace (protection DDoS)
- [ ] Error messages ne leak pas info sensible

### Performance

- [ ] Benchmarks latency < 5ms par middleware
- [ ] Redis pipelining si possible
- [ ] Batch PostgreSQL writes
- [ ] Profiling avec pprof (CPU/memory)

### Monitoring

- [ ] Métriques Prometheus:
  - [ ] `http_requests_total` (counter)
  - [ ] `http_request_duration_seconds` (histogram)
  - [ ] `visitor_sessions_active` (gauge)
  - [ ] `rate_limit_rejections_total` (counter)
- [ ] Logs structurés JSON (Loki-ready)
- [ ] Alertes (optionnel): Rate limit hit > threshold

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
