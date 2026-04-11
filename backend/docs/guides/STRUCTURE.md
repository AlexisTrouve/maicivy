# Structure du Backend - Maicivy

## 📁 Architecture des Dossiers

```
backend/
├── cmd/                                # Entry points
│   └── main.go                         # Application principale
│
├── internal/                           # Code privé (non exportable)
│   ├── api/                           # HTTP Handlers
│   │   └── health.go                  # Health check endpoints
│   │
│   ├── config/                        # Configuration
│   │   ├── config.go                  # Gestion env vars
│   │   └── config_test.go             # Tests config
│   │
│   ├── database/                      # Connexions DB
│   │   ├── postgres.go                # GORM PostgreSQL
│   │   ├── redis.go                   # go-redis client
│   │   └── postgres_test.go           # Tests integration
│   │
│   └── utils/                         # Utilitaires
│       ├── errors.go                  # Error handling custom
│       └── errors_test.go             # Tests errors
│
├── pkg/                               # Code réutilisable (exportable)
│   └── logger/                        # Logger structuré
│       └── logger.go                  # zerolog wrapper
│
├── tmp/                               # Binaires temporaires (air)
│   └── main                           # (généré, ignoré par git)
│
├── .air.toml                          # Config hot reload
├── .env                               # Variables d'environnement (ignoré)
├── .env.example                       # Template .env
├── .gitignore                         # Fichiers ignorés
├── Dockerfile                         # Image Docker backend
├── go.mod                             # Dépendances Go
├── go.sum                             # Checksums (généré)
├── Makefile                           # Commandes développement
├── README.md                          # Documentation principale
├── QUICK_START.md                     # Guide démarrage rapide
├── STRUCTURE.md                       # Ce fichier
├── TEST_INSTRUCTIONS.md               # Instructions de test
└── VALIDATION.md                      # Guide validation complète
```

## 🏗️ Architecture Logique

### Couches

```
┌─────────────────────────────────────────────────┐
│                 HTTP Layer                       │
│  (Fiber Router + Middlewares)                   │
│  - Recover, RequestID, Compress, CORS, Logger   │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│              Handler Layer                       │
│  (internal/api/)                                │
│  - HealthHandler                                │
│  - (Future: CVHandler, LetterHandler, etc.)     │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│             Service Layer                        │
│  (Future: internal/services/)                   │
│  - CVService, AIService, AnalyticsService       │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│           Repository Layer                       │
│  (Future: internal/repository/)                 │
│  - VisitorRepo, InteractionRepo, etc.           │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│             Database Layer                       │
│  (internal/database/)                           │
│  - PostgreSQL (GORM)                            │
│  - Redis (go-redis)                             │
└─────────────────────────────────────────────────┘
```

### Flow de Requête HTTP

```
1. Client → HTTP Request
              ↓
2. Fiber Router (matching route)
              ↓
3. Middlewares Chain
   - Recover (panic recovery)
   - RequestID (assign unique ID)
   - Compress (gzip/brotli)
   - CORS (cross-origin)
   - Logger (log request)
              ↓
4. Handler (ex: HealthHandler.Health)
   - Accès DB/Redis si nécessaire
   - Business logic
   - Prepare response
              ↓
5. Response → Client (JSON)
              ↓
6. Logger (log response + duration)
```

## 📦 Packages et Responsabilités

### cmd/

**Rôle:** Points d'entrée de l'application

**Fichiers:**
- `main.go` - Entry point principal
  - Charge la config
  - Initialise le logger
  - Connecte DB et Redis
  - Configure Fiber
  - Démarre le serveur
  - Gère graceful shutdown

### internal/config/

**Rôle:** Gestion de la configuration

**Responsabilités:**
- Charger variables d'environnement
- Fournir valeurs par défaut
- Valider la configuration
- Exposer struct `Config`

**Fichiers:**
- `config.go` - Config management
- `config_test.go` - Tests unitaires

### internal/database/

**Rôle:** Connexions aux bases de données

**Responsabilités:**
- Établir connexion PostgreSQL (GORM)
- Établir connexion Redis (go-redis)
- Configurer pools de connexions
- Gérer timeouts
- Fail-fast au démarrage

**Fichiers:**
- `postgres.go` - PostgreSQL + GORM
- `redis.go` - Redis client
- `postgres_test.go` - Tests integration

### internal/api/

**Rôle:** HTTP Handlers (contrôleurs)

**Responsabilités:**
- Recevoir requêtes HTTP
- Valider input
- Appeler services
- Formatter réponses
- Gérer erreurs

**Fichiers actuels:**
- `health.go` - Health checks

**Fichiers futurs (Phase 2+):**
- `cv.go` - CV API
- `letters.go` - Lettres API
- `analytics.go` - Analytics API

### internal/utils/

**Rôle:** Utilitaires transversaux

**Responsabilités:**
- Error handling custom
- Helpers divers
- Validators

**Fichiers:**
- `errors.go` - AppError, constructeurs, SendError
- `errors_test.go` - Tests

### pkg/logger/

**Rôle:** Logging structuré

**Responsabilités:**
- Configurer zerolog
- Adapter output selon environnement
- Fournir helpers (Error, Info, Debug, Warn)

**Fichiers:**
- `logger.go` - Logger wrapper

**Pourquoi dans `pkg/` ?**
- Peut être utilisé par d'autres projets
- Pas de dépendances internes

## 🔧 Fichiers de Configuration

### .env.example

Template pour variables d'environnement. Copier en `.env` et remplir.

**Sections:**
- Server (PORT, HOST, ENVIRONMENT)
- PostgreSQL (HOST, PORT, USER, PASSWORD, DB, SSL)
- Redis (HOST, PORT, PASSWORD, DB)
- API Keys (CLAUDE, OPENAI)

### .air.toml

Configuration Air pour hot reload en développement.

**Options:**
- Watch `.go` files
- Exclude `_test.go`
- Rebuild on change
- Delay: 1s

### .gitignore

Fichiers à ne pas commiter:
- Binaires (`*.exe`, `main`)
- Tests (`*.test`, `coverage.out`)
- Environnement (`.env`)
- Dossiers (`vendor/`, `tmp/`)
- IDEs (`.vscode/`, `.idea/`)

### Makefile

Commandes de développement:
- `make run` - Lancer l'app
- `make test` - Tests
- `make build` - Compiler
- `make lint` - Linter
- `make dev` - Hot reload

## 🧪 Tests

### Organisation

Les tests sont à côté du code source:
```
internal/config/
├── config.go
└── config_test.go      # Tests du fichier config.go
```

### Types de Tests

**Tests Unitaires** (`-short`):
- Ne nécessitent pas de DB
- Rapides (<1s)
- Exemples: config, errors

**Tests Integration**:
- Nécessitent PostgreSQL/Redis
- Plus lents (2-5s)
- Exemples: database connections

### Commandes

```bash
# Tests unitaires seulement
go test -v -short ./...

# Tous les tests
go test -v ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📝 Conventions de Code

### Naming

**Files:**
- Lowercase avec underscore: `postgres_test.go`
- Package name = directory name

**Functions:**
- Exported (public): PascalCase `ConnectPostgres()`
- Private: camelCase `getEnv()`

**Variables:**
- Exported: PascalCase `Config`
- Private: camelCase `cfg`, `db`, `redis`

**Constants:**
- PascalCase ou UPPER_SNAKE_CASE

### Structure

**1. Imports groupés:**
```go
import (
    // Standard library
    "fmt"
    "os"

    // External packages
    "github.com/gofiber/fiber/v2"

    // Internal packages
    "github.com/yourusername/maicivy/internal/config"
)
```

**2. Order dans les fichiers:**
```go
// 1. Package declaration
package api

// 2. Imports
import (...)

// 3. Types/Structs
type HealthHandler struct {...}

// 4. Constructors
func NewHealthHandler(...) {...}

// 5. Methods
func (h *HealthHandler) Health(...) {...}

// 6. Private helpers
func doSomething(...) {...}
```

### Error Handling

**Toujours propager les erreurs:**
```go
if err != nil {
    return nil, fmt.Errorf("context: %w", err)
}
```

**Utiliser AppError pour erreurs métier:**
```go
return utils.NewBadRequestError("invalid input")
```

### Logging

**Utiliser zerolog partout:**
```go
log.Info().Str("key", "value").Msg("message")
log.Error().Err(err).Msg("error occurred")
```

## 🚀 Prochaines Additions

### Phase 2 (Database Schema)

Ajouts prévus:
```
internal/
└── models/
    ├── visitor.go       # Model Visitor
    ├── interaction.go   # Model Interaction
    ├── theme.go         # Model Theme
    └── migrations/
        └── 001_initial.sql
```

### Phase 3 (Middlewares)

Ajouts prévus:
```
internal/
└── middleware/
    ├── tracking.go      # Visitor tracking
    ├── ratelimit.go     # Rate limiting
    └── auth.go          # Authentication (future)
```

### Phase 4 (Services)

Ajouts prévus:
```
internal/
├── services/
│   ├── cv.go           # CV business logic
│   ├── ai.go           # AI services
│   └── analytics.go    # Analytics
└── repository/
    ├── visitor.go      # Visitor repository
    └── interaction.go  # Interaction repository
```

## 📊 Statistiques

**Fichiers Go actuels:** 10 fichiers (7 main + 3 tests)

**Lines of Code (approximatif):**
- `cmd/main.go`: ~150 lignes
- `internal/`: ~400 lignes
- `pkg/`: ~50 lignes
- **Total:** ~600 lignes

**Dépendances directes:** 7 packages
- fiber, gorm, go-redis, zerolog, godotenv, validator, websocket

**Coverage actuel:** ~60% (avec tests unitaires)

---

**Cette structure suit:**
- ✅ Go project layout standard
- ✅ Clean Architecture principles
- ✅ Separation of Concerns
- ✅ Testability
- ✅ Scalability

**Prêt pour l'ajout de nouvelles fonctionnalités !** 🚀
