# 02. BACKEND FOUNDATION

## 📋 Métadonnées

- **Phase:** 1
- **Priorité:** CRITIQUE
- **Complexité:** ⭐⭐⭐ (3/5)
- **Prérequis:** 01. SETUP_INFRASTRUCTURE.md
- **Temps estimé:** 2-3 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Mettre en place la fondation du backend Go avec le framework Fiber, incluant la configuration de base, les connexions aux bases de données (PostgreSQL via GORM et Redis), le système de logging structuré, la gestion des erreurs et la configuration de l'application.

Cette étape établit l'architecture de base du projet Go et fournit tous les composants essentiels pour le développement des fonctionnalités métier.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
backend/
├── cmd/
│   └── main.go                 # Point d'entrée de l'application
├── internal/
│   ├── config/
│   │   └── config.go           # Gestion configuration (env vars)
│   ├── database/
│   │   ├── postgres.go         # Connexion PostgreSQL + GORM
│   │   └── redis.go            # Connexion Redis
│   ├── api/
│   │   └── health.go           # Health check endpoint
│   └── utils/
│       └── errors.go           # Error handling custom
├── pkg/
│   └── logger/
│       └── logger.go           # Logger structuré (zerolog)
├── go.mod
├── go.sum
├── .env.example
└── Dockerfile
```

### Design Decisions

**1. Framework Web: Fiber**
- Choix justifié : Performance élevée (Express-like pour Go)
- API simple et intuitive
- Middleware ecosystem riche
- Meilleure performance que Gin pour ce use case

**2. ORM: GORM**
- Abstraction des requêtes SQL
- Migrations automatiques
- Relations complexes simplifiées
- Large adoption dans l'écosystème Go

**3. Logger: zerolog**
- Logging structuré JSON
- Zero allocation (performance)
- Facile à parser pour monitoring
- Compatible avec Loki/ELK

**4. Configuration: Variables d'environnement**
- Simple et standard (12-factor app)
- Pas de dépendance lourde (viper optionnel)
- Compatible Docker/Kubernetes

**5. Structure `internal/`**
- Package non exportable (bonnes pratiques Go)
- Séparation claire des responsabilités
- Évolutivité facilitée

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Framework web
go get github.com/gofiber/fiber/v2

# Base de données
go get gorm.io/gorm
go get gorm.io/driver/postgres

# Redis
go get github.com/redis/go-redis/v9

# Logger
go get github.com/rs/zerolog

# Configuration
go get github.com/joho/godotenv

# Validation
go get github.com/go-playground/validator/v10
```

### Services Externes

- **PostgreSQL** : Base de données principale (fournie par docker-compose)
- **Redis** : Cache et sessions (fournie par docker-compose)

---

## 🔨 Implémentation

### Étape 1: Initialisation du Module Go

**Description:** Créer le module Go et installer les dépendances de base

**Code:**

```bash
cd backend
go mod init github.com/yourusername/maicivy
```

**Fichier `go.mod` initial:**

```go
module github.com/yourusername/maicivy

go 1.21

require (
    github.com/gofiber/fiber/v2 v2.51.0
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    github.com/redis/go-redis/v9 v9.3.0
    github.com/rs/zerolog v1.31.0
    github.com/joho/godotenv v1.5.1
    github.com/go-playground/validator/v10 v10.16.0
)
```

**Explications:**
- Module path doit correspondre à votre repository
- Version Go 1.21+ recommandée
- Versions des packages à jour au moment de l'implémentation

---

### Étape 2: Configuration Management

**Description:** Créer le système de configuration via variables d'environnement

**Fichier `internal/config/config.go`:**

```go
package config

import (
    "fmt"
    "os"
    "strconv"

    "github.com/joho/godotenv"
    "github.com/rs/zerolog/log"
)

type Config struct {
    // Server
    ServerPort string
    ServerHost string
    Environment string

    // Database
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    DBSSLMode  string

    // Redis
    RedisHost     string
    RedisPort     string
    RedisPassword string
    RedisDB       int

    // API Keys (pour Phase 3)
    ClaudeAPIKey string
    OpenAIAPIKey string
}

func Load() (*Config, error) {
    // Charger .env en développement (ignore si non présent)
    _ = godotenv.Load()

    cfg := &Config{
        // Server
        ServerPort:  getEnv("SERVER_PORT", "8080"),
        ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
        Environment: getEnv("ENVIRONMENT", "development"),

        // PostgreSQL
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBPort:     getEnv("DB_PORT", "5432"),
        DBUser:     getEnv("DB_USER", "maicivy"),
        DBPassword: getEnv("DB_PASSWORD", ""),
        DBName:     getEnv("DB_NAME", "maicivy"),
        DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

        // Redis
        RedisHost:     getEnv("REDIS_HOST", "localhost"),
        RedisPort:     getEnv("REDIS_PORT", "6379"),
        RedisPassword: getEnv("REDIS_PASSWORD", ""),
        RedisDB:       getEnvAsInt("REDIS_DB", 0),

        // API Keys
        ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
        OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}

func (c *Config) Validate() error {
    // Validation minimale (Phase 1)
    if c.DBPassword == "" {
        log.Warn().Msg("DB_PASSWORD is empty (not recommended for production)")
    }
    return nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    valueStr := os.Getenv(key)
    if value, err := strconv.Atoi(valueStr); err == nil {
        return value
    }
    return defaultValue
}
```

**Fichier `.env.example`:**

```env
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
ENVIRONMENT=development

# PostgreSQL
DB_HOST=postgres
DB_PORT=5432
DB_USER=maicivy
DB_PASSWORD=your_secure_password
DB_NAME=maicivy
DB_SSL_MODE=disable

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# AI API Keys (Phase 3)
CLAUDE_API_KEY=
OPENAI_API_KEY=
```

**Explications:**
- Pattern getEnv avec valeurs par défaut pour développement local
- Validation de base (extensible)
- Support .env pour dev, variables système pour prod
- API keys préparées pour Phase 3 (optionnelles en Phase 1)

---

### Étape 3: Logger Structuré

**Description:** Configurer zerolog pour logging structuré JSON

**Fichier `pkg/logger/logger.go`:**

```go
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
```

**Explications:**
- Output console coloré en dev pour lisibilité
- Output JSON en production pour parsing par Prometheus/Loki
- Niveau de log adapté (Debug en dev, Info en prod)
- Helpers pour simplifier l'usage dans le code

---

### Étape 4: Connexion PostgreSQL avec GORM

**Description:** Établir la connexion à PostgreSQL avec GORM et gestion du pool de connexions

**Fichier `internal/database/postgres.go`:**

```go
package database

import (
    "fmt"
    "time"

    "github.com/rs/zerolog/log"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"

    "github.com/yourusername/maicivy/internal/config"
)

func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
        cfg.DBHost,
        cfg.DBUser,
        cfg.DBPassword,
        cfg.DBName,
        cfg.DBPort,
        cfg.DBSSLMode,
    )

    // Configuration GORM logger
    gormLogger := logger.Default.LogMode(logger.Silent)
    if cfg.Environment == "development" {
        gormLogger = logger.Default.LogMode(logger.Info)
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: gormLogger,
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })

    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Configuration du pool de connexions
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get database instance: %w", err)
    }

    // Pool settings (selon IMPLEMENTATION_PLAN.md)
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)

    // Test de connexion
    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    log.Info().
        Str("host", cfg.DBHost).
        Str("database", cfg.DBName).
        Msg("PostgreSQL connected successfully")

    return db, nil
}
```

**Explications:**
- DSN (Data Source Name) construit depuis la config
- Logger GORM désactivé en production (utiliser zerolog à la place)
- Pool de connexions configuré pour performance (valeurs standards)
- Ping pour vérifier la connexion au démarrage
- Timestamps en UTC (bonne pratique)

---

### Étape 5: Connexion Redis

**Description:** Configurer le client Redis pour cache et sessions

**Fichier `internal/database/redis.go`:**

```go
package database

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog/log"

    "github.com/yourusername/maicivy/internal/config"
)

func ConnectRedis(cfg *config.Config) (*redis.Client, error) {
    addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

    client := redis.NewClient(&redis.Options{
        Addr:         addr,
        Password:     cfg.RedisPassword,
        DB:           cfg.RedisDB,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolSize:     10,
        MinIdleConns: 5,
    })

    // Test de connexion
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    log.Info().
        Str("addr", addr).
        Int("db", cfg.RedisDB).
        Msg("Redis connected successfully")

    return client, nil
}
```

**Explications:**
- Client Redis avec timeouts configurés
- Pool de connexions pour performance
- Ping avec timeout pour fail-fast au démarrage
- Configuration adaptée pour haute disponibilité

---

### Étape 6: Error Handling Global

**Description:** Créer un système d'erreurs custom pour l'application

**Fichier `internal/utils/errors.go`:**

```go
package utils

import (
    "fmt"

    "github.com/gofiber/fiber/v2"
)

// AppError représente une erreur applicative
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}

// Constructeurs d'erreurs communes
func NewBadRequestError(message string) *AppError {
    return &AppError{
        Code:    fiber.StatusBadRequest,
        Message: message,
    }
}

func NewNotFoundError(resource string) *AppError {
    return &AppError{
        Code:    fiber.StatusNotFound,
        Message: fmt.Sprintf("%s not found", resource),
    }
}

func NewInternalError(message string) *AppError {
    return &AppError{
        Code:    fiber.StatusInternalServerError,
        Message: message,
    }
}

func NewUnauthorizedError(message string) *AppError {
    return &AppError{
        Code:    fiber.StatusUnauthorized,
        Message: message,
    }
}

func NewForbiddenError(message string) *AppError {
    return &AppError{
        Code:    fiber.StatusForbidden,
        Message: message,
    }
}

func NewRateLimitError(message string) *AppError {
    return &AppError{
        Code:    fiber.StatusTooManyRequests,
        Message: message,
    }
}

// ErrorResponse format de réponse d'erreur standardisé
type ErrorResponse struct {
    Error string `json:"error"`
    Code  int    `json:"code"`
}

// SendError envoie une erreur formatée au client
func SendError(c *fiber.Ctx, err error) error {
    if appErr, ok := err.(*AppError); ok {
        return c.Status(appErr.Code).JSON(ErrorResponse{
            Error: appErr.Message,
            Code:  appErr.Code,
        })
    }

    // Erreur non typée = 500
    return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
        Error: "Internal server error",
        Code:  fiber.StatusInternalServerError,
    })
}
```

**Explications:**
- Erreurs typées avec codes HTTP appropriés
- Format de réponse JSON standardisé
- Helper SendError pour simplifier le code des handlers
- Extensible pour ajout de nouveaux types d'erreurs

---

### Étape 7: Health Check Endpoint

**Description:** Créer l'endpoint `/health` pour vérifier l'état de l'application

**Fichier `internal/api/health.go`:**

```go
package api

import (
    "context"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

type HealthHandler struct {
    db    *gorm.DB
    redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
    return &HealthHandler{
        db:    db,
        redis: redisClient,
    }
}

type HealthResponse struct {
    Status   string            `json:"status"`
    Services map[string]string `json:"services"`
}

// Health - Shallow health check (rapide)
func (h *HealthHandler) Health(c *fiber.Ctx) error {
    return c.JSON(HealthResponse{
        Status: "ok",
        Services: map[string]string{
            "api": "up",
        },
    })
}

// HealthDeep - Deep health check (vérifie DB et Redis)
func (h *HealthHandler) HealthDeep(c *fiber.Ctx) error {
    services := make(map[string]string)
    status := "ok"

    // Check PostgreSQL
    sqlDB, err := h.db.DB()
    if err != nil {
        services["postgres"] = "down"
        status = "degraded"
    } else {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()

        if err := sqlDB.PingContext(ctx); err != nil {
            services["postgres"] = "down"
            status = "degraded"
        } else {
            services["postgres"] = "up"
        }
    }

    // Check Redis
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := h.redis.Ping(ctx).Err(); err != nil {
        services["redis"] = "down"
        status = "degraded"
    } else {
        services["redis"] = "up"
    }

    services["api"] = "up"

    httpStatus := fiber.StatusOK
    if status == "degraded" {
        httpStatus = fiber.StatusServiceUnavailable
    }

    return c.Status(httpStatus).JSON(HealthResponse{
        Status:   status,
        Services: services,
    })
}
```

**Explications:**
- Deux endpoints : `/health` (rapide) et `/health/deep` (complet)
- Health shallow pour load balancers (pas de latence DB)
- Health deep pour monitoring détaillé
- Timeouts pour éviter les blocages
- Status HTTP 503 si services dégradés (pour alerting)

---

### Étape 8: Application Fiber Setup

**Description:** Créer le point d'entrée principal avec configuration Fiber

**Fichier `cmd/main.go`:**

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/compress"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "github.com/gofiber/fiber/v2/middleware/requestid"
    "github.com/rs/zerolog/log"

    "github.com/yourusername/maicivy/internal/api"
    "github.com/yourusername/maicivy/internal/config"
    "github.com/yourusername/maicivy/internal/database"
    "github.com/yourusername/maicivy/pkg/logger"
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

    // 6. Middlewares globaux
    app.Use(recover.New(recover.Config{
        EnableStackTrace: cfg.Environment == "development",
    }))
    app.Use(requestid.New())
    app.Use(compress.New(compress.Config{
        Level: compress.LevelBestSpeed,
    }))
    app.Use(cors.New(cors.Config{
        AllowOrigins:     "http://localhost:3000", // À configurer via env en prod
        AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
        AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
        AllowCredentials: true,
    }))

    // Middleware de logging custom
    app.Use(func(c *fiber.Ctx) error {
        start := time.Now()

        err := c.Next()

        log.Info().
            Str("method", c.Method()).
            Str("path", c.Path()).
            Int("status", c.Response().StatusCode()).
            Dur("duration_ms", time.Since(start)).
            Str("request_id", c.GetRespHeader("X-Request-ID")).
            Msg("HTTP request")

        return err
    })

    // 7. Routes
    healthHandler := api.NewHealthHandler(db, redisClient)

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

    // 8. Graceful shutdown
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

    // 9. Attendre signal de shutdown
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
```

**Explications:**
- Initialisation dans l'ordre logique (config → logger → DB → app)
- Middlewares de base : recover, requestid, compress, cors, logging
- Configuration Fiber avec timeouts appropriés
- Graceful shutdown pour terminer proprement les connexions
- Error handler centralisé
- Structure prête pour ajout de routes (groupes API)
- Logging structuré de chaque requête HTTP

---

### Étape 9: Dockerfile Backend

**Description:** Créer le Dockerfile pour containeriser le backend

**Fichier `backend/Dockerfile`:**

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Installer les dépendances système
RUN apk add --no-cache git

# Copier go.mod et go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copier le code source
COPY . .

# Build l'application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copier le binaire depuis le builder
COPY --from=builder /app/main .

# Exposer le port
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Lancer l'application
CMD ["./main"]
```

**Explications:**
- Multi-stage build pour image légère (Alpine ~15MB vs Ubuntu ~200MB)
- CGO_ENABLED=0 pour build statique (pas de dépendances dynamiques)
- ca-certificates pour HTTPS (API IA Phase 3)
- Healthcheck Docker natif
- Timezone data pour logs précis

---

## 🧪 Tests

### Tests Unitaires

**Fichier `internal/config/config_test.go`:**

```go
package config

import (
    "os"
    "testing"
)

func TestLoad(t *testing.T) {
    // Set env vars for test
    os.Setenv("DB_PASSWORD", "test123")
    defer os.Unsetenv("DB_PASSWORD")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }

    if cfg.DBPassword != "test123" {
        t.Errorf("Expected DB_PASSWORD=test123, got %s", cfg.DBPassword)
    }

    if cfg.ServerPort != "8080" {
        t.Errorf("Expected default ServerPort=8080, got %s", cfg.ServerPort)
    }
}

func TestGetEnv(t *testing.T) {
    os.Setenv("TEST_VAR", "value")
    defer os.Unsetenv("TEST_VAR")

    result := getEnv("TEST_VAR", "default")
    if result != "value" {
        t.Errorf("Expected 'value', got '%s'", result)
    }

    result = getEnv("NON_EXISTENT", "default")
    if result != "default" {
        t.Errorf("Expected 'default', got '%s'", result)
    }
}
```

**Fichier `internal/utils/errors_test.go`:**

```go
package utils

import (
    "testing"

    "github.com/gofiber/fiber/v2"
)

func TestAppError(t *testing.T) {
    err := NewBadRequestError("invalid input")

    if err.Code != fiber.StatusBadRequest {
        t.Errorf("Expected code 400, got %d", err.Code)
    }

    if err.Message != "invalid input" {
        t.Errorf("Expected message 'invalid input', got '%s'", err.Message)
    }
}

func TestErrorConstructors(t *testing.T) {
    tests := []struct {
        name     string
        err      *AppError
        wantCode int
    }{
        {"BadRequest", NewBadRequestError("test"), 400},
        {"NotFound", NewNotFoundError("user"), 404},
        {"Unauthorized", NewUnauthorizedError("test"), 401},
        {"Forbidden", NewForbiddenError("test"), 403},
        {"RateLimit", NewRateLimitError("test"), 429},
        {"Internal", NewInternalError("test"), 500},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.err.Code != tt.wantCode {
                t.Errorf("Expected code %d, got %d", tt.wantCode, tt.err.Code)
            }
        })
    }
}
```

### Tests Integration

**Fichier `internal/database/postgres_test.go`:**

```go
package database

import (
    "testing"

    "github.com/yourusername/maicivy/internal/config"
)

// Note: Ce test nécessite une instance PostgreSQL de test
// Utiliser testcontainers en Phase 6 pour tests isolés
func TestConnectPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    cfg := &config.Config{
        DBHost:     "localhost",
        DBPort:     "5432",
        DBUser:     "test",
        DBPassword: "test",
        DBName:     "test",
        DBSSLMode:  "disable",
        Environment: "test",
    }

    db, err := ConnectPostgres(cfg)
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }

    sqlDB, _ := db.DB()
    if err := sqlDB.Ping(); err != nil {
        t.Errorf("Failed to ping database: %v", err)
    }
}
```

### Commandes

```bash
# Tests unitaires uniquement (rapides)
go test -v -short ./...

# Tous les tests (incluant integration)
go test -v ./...

# Coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Linter
go fmt ./...
go vet ./...

# Optionnel: golangci-lint (Phase 6)
golangci-lint run
```

---

## ⚠️ Points d'Attention

- **Piège 1: Connection leaks**
  - Toujours fermer les connexions DB dans les handlers
  - GORM gère le pool automatiquement, mais attention aux transactions manuelles
  - Utiliser `defer tx.Rollback()` après `db.Begin()`

- **Piège 2: Variables d'environnement en production**
  - Ne JAMAIS commit le fichier `.env`
  - Utiliser secrets manager en production (Phase 6)
  - Valider les variables critiques au démarrage (fail-fast)

- **Edge case: Redis indisponible**
  - L'application doit démarrer même si Redis est down (mode dégradé)
  - Implémenter fallback pour features non-critiques (caching)
  - Features critiques (rate limiting) doivent fail-safe

- **Astuce 1: GORM Debug mode**
  - Activer en dev pour voir les requêtes SQL générées
  - Désactiver en prod pour performance
  - Utiliser zerolog pour logs structurés à la place

- **Astuce 2: Fiber prefork mode**
  - Ne PAS utiliser en dev (complique debugging)
  - Activer en prod pour performance (`Prefork: true`)
  - Attention : incompatible avec certains middlewares

- **Astuce 3: Context timeout**
  - Toujours utiliser `context.WithTimeout` pour DB/Redis
  - Timeout recommandé : 5-10s pour DB, 2-3s pour Redis
  - Évite les goroutines zombies

---

## 📚 Ressources

- [Fiber Documentation](https://docs.gofiber.io/)
- [GORM Documentation](https://gorm.io/docs/)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [zerolog Documentation](https://github.com/rs/zerolog)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [12-Factor App Methodology](https://12factor.net/)
- [Effective Go](https://go.dev/doc/effective_go)

---

## ✅ Checklist de Complétion

- [ ] Module Go initialisé avec toutes les dépendances
- [ ] Système de configuration (config package) fonctionnel
- [ ] Logger zerolog configuré (dev + prod modes)
- [ ] Connexion PostgreSQL avec GORM opérationnelle
- [ ] Connexion Redis opérationnelle
- [ ] Error handling custom implémenté
- [ ] Health check endpoints (`/health`, `/health/deep`) fonctionnels
- [ ] Application Fiber démarre correctement
- [ ] Dockerfile backend créé et testé
- [ ] Tests unitaires écrits (config, errors)
- [ ] Tests integration PostgreSQL/Redis passants
- [ ] Logging HTTP de chaque requête actif
- [ ] Graceful shutdown fonctionnel
- [ ] Documentation code (commentaires Go)
- [ ] `.env.example` créé avec toutes les variables
- [ ] Review sécurité (pas de secrets hardcodés)
- [ ] Review performance (pool sizes, timeouts)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
