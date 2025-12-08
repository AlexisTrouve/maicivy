# Letters API Implementation Summary

**Document:** 09_BACKEND_LETTERS_API.md
**Date:** 2025-12-08
**Status:** ✅ Implémenté (avec mocks pour services IA)

---

## 🎯 Objectif

Implémentation complète de l'API REST pour la génération de lettres de motivation et anti-motivation par IA, avec système de contrôle d'accès basé sur le tracking des visiteurs, rate limiting strict, et queue asynchrone.

---

## 📦 Livrables

### ✅ Fichiers Créés

#### Middlewares
1. **`internal/middleware/access_gate.go`** (130 lignes)
   - Vérification d'accès IA (3 visites minimum)
   - Bypass pour profils détectés (recruiter, tech_lead, cto, ceo)
   - Messages d'erreur clairs avec teaser

2. **`internal/middleware/ai_ratelimit.go`** (180 lignes)
   - Rate limiting journalier (5 générations/jour)
   - Cooldown entre générations (2 minutes)
   - Expiration intelligente (reset à minuit)
   - Headers HTTP informatifs

#### DTOs
3. **`internal/api/dto/letters.go`** (120 lignes)
   - `GenerateLetterRequest` avec validation
   - `LetterGenerationResponse`, `LetterJobStatus`
   - `LetterDetailResponse`, `LetterPairResponse`
   - `LetterHistoryResponse`, `AccessStatusResponse`
   - `RateLimitStatusResponse`

#### Services
4. **`internal/services/letter_queue.go`** (220 lignes)
   - Service de queue Redis FIFO
   - Gestion des jobs (enqueue, pop, status)
   - States: queued → processing → completed/failed
   - Retry logic (max 3 tentatives)
   - Estimation temps restant

#### Workers
5. **`internal/workers/letter_worker.go`** (250 lignes)
   - Worker background avec goroutine
   - Traitement asynchrone des jobs
   - Progress tracking (10% → 100%)
   - Mock de génération IA (en attente Doc 08)
   - Sauvegarde des deux lettres en DB

#### API Handlers
6. **`internal/api/letters.go`** (450 lignes)
   - `POST /api/v1/letters/generate` - Génération asynchrone
   - `GET /api/v1/letters/jobs/:jobId` - Status du job
   - `GET /api/v1/letters/:id` - Récupérer une lettre
   - `GET /api/v1/letters/pair?company=X` - Paire de lettres
   - `GET /api/v1/letters/history` - Historique avec pagination
   - `GET /api/v1/letters/:id/pdf` - Téléchargement PDF (mock)
   - `GET /api/v1/letters/access-status` - Status d'accès IA
   - `GET /api/v1/letters/rate-limit-status` - Status rate limiting

#### Tests
7. **`internal/middleware/access_gate_test.go`** (130 lignes)
   - Tests visites insuffisantes
   - Tests bypass profil
   - Tests session manquante
   - Tests visites suffisantes

8. **`internal/middleware/ai_ratelimit_test.go`** (130 lignes)
   - Tests première requête
   - Tests limite journalière
   - Tests cooldown
   - Tests incrémentation

9. **`internal/services/letter_queue_test.go`** (200 lignes)
   - Tests enqueue/dequeue
   - Tests status updates
   - Tests completion/failure
   - Tests retry logic
   - Tests estimation temps

#### Documentation
10. **`backend/LETTERS_API_IMPLEMENTATION_SUMMARY.md`** (ce fichier)

---

## 🏗️ Architecture

### Flow de Génération Asynchrone

```
┌─────────────┐
│  Frontend   │
│ POST /generate
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│  Access Gate Middleware         │
│  ✓ Vérif 3 visites              │
│  ✓ OU profil détecté            │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│  AI Rate Limit Middleware       │
│  ✓ Max 5/jour                   │
│  ✓ Cooldown 2min                │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│  Letters Handler                │
│  • Enqueue job → Redis          │
│  • Return 202 + jobID           │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│  Redis Queue                    │
│  queue:letters [job1, job2...]  │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│  Letter Worker (Background)     │
│  1. Pop job (BLPOP)             │
│  2. Update status → processing  │
│  3. Mock generate letters       │
│  4. Save to PostgreSQL          │
│  5. Update status → completed   │
└─────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────┐
│  PostgreSQL                     │
│  • generated_letters (2 rows)   │
│    - motivation                 │
│    - anti_motivation            │
└─────────────────────────────────┘
```

### Polling Frontend

```
POST /generate
    ↓
Receive jobID
    ↓
While status != completed:
    GET /jobs/:jobId
    Wait 2 seconds
    ↓
GET /letters/:id (motivation)
GET /letters/:id (anti-motivation)
```

---

## 🔒 Règles de Sécurité

### Access Gate (3 Visites)
- **Règle:** >= 3 visites OU profil cible
- **Profils cibles:** recruiter, tech_lead, cto, ceo
- **Erreur:** HTTP 403 avec message teaser
- **Bypass:** Détection automatique via Redis

### Rate Limiting IA
- **Limite journalière:** 5 générations/jour/session
- **Reset:** Minuit (expiration dynamique)
- **Cooldown:** 2 minutes entre générations
- **Headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`

### Ownership Check
- Toutes les opérations vérifient `visitor_id`
- Impossible d'accéder aux lettres d'un autre visiteur
- Session ID validé via cookie HTTPOnly

---

## 📊 Endpoints API

### POST /api/v1/letters/generate
**Description:** Génère une paire de lettres (motivation + anti-motivation)

**Request:**
```json
{
  "company_name": "Google",
  "job_title": "Senior Backend Engineer",
  "theme": "backend"
}
```

**Response (202 Accepted):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "message": "Génération en cours. Encore 4 génération(s) disponible(s) aujourd'hui.",
  "rate_limit_remaining": 4
}
```

**Middlewares:** AccessGate + AIRateLimit

---

### GET /api/v1/letters/jobs/:jobId
**Description:** Récupère le status d'un job

**Response:**
```json
{
  "job_id": "550e8400...",
  "status": "processing",
  "progress": 60,
  "estimated_time": 12,
  "letter_motivation_id": null,
  "letter_anti_motivation_id": null,
  "error": null
}
```

**Status possibles:** `queued`, `processing`, `completed`, `failed`

---

### GET /api/v1/letters/:id
**Description:** Récupère une lettre générée

**Response:**
```json
{
  "id": 1,
  "company_name": "Google",
  "letter_type": "motivation",
  "content": "Madame, Monsieur...",
  "created_at": "2025-12-08 18:30:00",
  "ai_model": "claude-3-sonnet",
  "tokens_used": 500,
  "generation_ms": 2850,
  "cost": 0.005,
  "pdf_url": "https://api.example.com/api/v1/letters/1/pdf"
}
```

---

### GET /api/v1/letters/pair?company=Google
**Description:** Récupère une paire de lettres pour une entreprise

**Response:**
```json
{
  "motivation_letter": { /* LetterDetailResponse */ },
  "anti_motivation_letter": { /* LetterDetailResponse */ },
  "company_name": "Google",
  "company_info": {
    "name": "Google",
    "industry": "Technology",
    "size": "10000+"
  }
}
```

---

### GET /api/v1/letters/history?page=1&per_page=10
**Description:** Historique des lettres générées (pagination)

**Response:**
```json
{
  "letters": [
    {
      "id": 2,
      "company_name": "Meta",
      "letter_type": "anti_motivation",
      "created_at": "2025-12-08 18:00:00",
      "downloaded": false
    },
    {
      "id": 1,
      "company_name": "Google",
      "letter_type": "motivation",
      "created_at": "2025-12-08 17:30:00",
      "downloaded": true
    }
  ],
  "total": 10,
  "page": 1,
  "per_page": 10
}
```

---

### GET /api/v1/letters/:id/pdf
**Description:** Télécharge le PDF d'une lettre

**Response:** Fichier PDF (ou texte en mode mock)

**Headers:**
- `Content-Type: application/pdf`
- `Content-Disposition: attachment; filename="lettre_motivation_Google.pdf"`

---

### GET /api/v1/letters/access-status
**Description:** Status d'accès aux fonctionnalités IA

**Response:**
```json
{
  "has_access": true,
  "current_visits": 5,
  "required_visits": 3,
  "visits_remaining": 0,
  "profile_detected": "recruiter",
  "access_granted_by": "profile",
  "message": "Accès aux fonctionnalités IA accordé"
}
```

---

### GET /api/v1/letters/rate-limit-status
**Description:** Status du rate limiting IA

**Response:**
```json
{
  "daily_limit": 5,
  "daily_used": 2,
  "daily_remaining": 3,
  "reset_at": "2025-12-09 00:00:00",
  "cooldown_active": false,
  "cooldown_remaining": 0
}
```

---

## 🧪 Tests

### Couverture

```bash
# Tests middlewares
go test -v ./internal/middleware/
# ✓ access_gate_test.go (4 tests)
# ✓ ai_ratelimit_test.go (5 tests)

# Tests services
go test -v ./internal/services/
# ✓ letter_queue_test.go (11 tests)

# Tests coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Résultats Attendus
- **Middlewares:** 100% couverture (tous les cas d'erreur testés)
- **Services:** 95% couverture (retry logic, queue operations)
- **Handlers:** 80% couverture (à compléter avec tests d'intégration)

---

## 🔧 Intégration dans main.go

```go
package main

import (
    "log"
    "time"

    "github.com/gofiber/fiber/v2"

    "maicivy/internal/api"
    "maicivy/internal/middleware"
    "maicivy/internal/services"
    "maicivy/internal/workers"
)

func main() {
    // ... Setup DB, Redis, Config ...

    app := fiber.New()

    // Services
    queueService := services.NewLetterQueueService(redisClient)

    // Handlers
    lettersHandler := api.NewLettersHandler(db, redisClient, queueService)

    // Start worker en background
    worker := workers.NewLetterWorker(db, queueService)
    go worker.Start()
    defer worker.Stop()

    // Routes Letters API
    lettersGroup := app.Group("/api/v1/letters")

    // Génération (avec middlewares)
    lettersGroup.Post("/generate",
        middleware.AccessGate(middleware.AccessGateConfig{
            Redis:           redisClient,
            DB:              db,
            MinVisits:       3,
            BypassOnProfile: true,
        }),
        middleware.AIRateLimit(middleware.AIRateLimitConfig{
            Redis:            redisClient,
            MaxPerDay:        5,
            CooldownDuration: 2 * time.Minute,
        }),
        lettersHandler.GenerateLetter,
    )

    // Status et consultation (pas de middlewares)
    lettersGroup.Get("/jobs/:jobId", lettersHandler.GetJobStatus)
    lettersGroup.Get("/:id", lettersHandler.GetLetter)
    lettersGroup.Get("/pair", lettersHandler.GetLetterPair)
    lettersGroup.Get("/history", lettersHandler.GetHistory)
    lettersGroup.Get("/:id/pdf", lettersHandler.DownloadPDF)
    lettersGroup.Get("/access-status", lettersHandler.GetAccessStatus)
    lettersGroup.Get("/rate-limit-status", lettersHandler.GetRateLimitStatus)

    log.Fatal(app.Listen(":8080"))
}
```

---

## 📝 Commandes de Validation

```bash
# 1. Lancer les tests
cd backend
go test -v ./internal/middleware/
go test -v ./internal/services/
go test -v ./internal/api/

# 2. Vérifier la compilation
go build ./cmd/main.go

# 3. Linter
golangci-lint run

# 4. Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 5. Tests d'intégration (après Docker up)
# Test génération
curl -X POST http://localhost:8080/api/v1/letters/generate \
  -H "Content-Type: application/json" \
  -H "Cookie: session_id=test-session" \
  -d '{"company_name": "Google"}'

# Test status job
curl http://localhost:8080/api/v1/letters/jobs/{jobId} \
  -H "Cookie: session_id=test-session"

# Test historique
curl http://localhost:8080/api/v1/letters/history \
  -H "Cookie: session_id=test-session"

# Test access status
curl http://localhost:8080/api/v1/letters/access-status \
  -H "Cookie: session_id=test-session"

# Test rate limit status
curl http://localhost:8080/api/v1/letters/rate-limit-status \
  -H "Cookie: session_id=test-session"
```

---

## ⚠️ Notes Importantes

### 1. Services IA (Doc 08)
Les services suivants sont **mockés** en attendant l'implémentation du Document 08 :
- `AIService.GenerateMotivationLetter()`
- `AIService.GenerateAntiMotivationLetter()`
- `ScraperService.GetCompanyInfo()`
- `PDFLetterService.GeneratePDFs()`

Le worker utilise des fonctions `mock*` qui génèrent du contenu de placeholder.

### 2. PDF Generation
Le endpoint `/letters/:id/pdf` retourne actuellement du texte brut (`.txt`).
Une fois le service PDF implémenté (Doc 08), il retournera de vrais PDFs.

### 3. Queue Scaling
Le worker actuel est **single-threaded** (1 job à la fois).
Pour scaler, lancer plusieurs instances du worker :
```go
for i := 0; i < 3; i++ {
    worker := workers.NewLetterWorker(db, queueService)
    go worker.Start()
}
```

### 4. Monitoring
Ajouter des métriques Prometheus :
- `letters_generated_total{status="completed|failed"}`
- `letters_queue_length`
- `letters_generation_duration_seconds`
- `rate_limit_hits_total{reason="daily|cooldown"}`

---

## 🎯 Prochaines Étapes

### Immédiat
1. ✅ **Intégrer dans main.go** - Routes + Worker startup
2. ⏳ **Attendre Doc 08** - Implémentation vrais services IA
3. ⏳ **Tests end-to-end** - Avec PostgreSQL et Redis réels

### Phase 3 Complète
4. **Frontend (Doc 10)** - Interface de génération de lettres
5. **WebSocket** (optionnel) - Push temps réel du progress
6. **Monitoring** - Grafana dashboard des générations

### Production
7. **Cache Redis** - Cacher lettres par `hash(company + theme)`
8. **Cleanup Job** - Nettoyer vieux jobs Redis (>24h)
9. **Alerting** - Alertes si queue > 100 jobs
10. **Cost Tracking** - Dashboard des coûts API IA

---

## ✅ Checklist de Complétion

- [x] Model `GeneratedLetter` existant et compatible
- [x] Middleware `AccessGate` implémenté et testé
- [x] Middleware `AIRateLimit` implémenté et testé
- [x] DTOs créés avec validation
- [x] Service `LetterQueueService` implémenté et testé
- [x] Worker `LetterWorker` implémenté avec mocks
- [x] Handler `GenerateLetter` (POST) créé
- [x] Handler `GetJobStatus` (GET) créé
- [x] Handler `GetLetter` (GET) créé
- [x] Handler `GetLetterPair` (GET) créé
- [x] Handler `DownloadPDF` (GET) créé (mock)
- [x] Handler `GetHistory` (GET) créé
- [x] Handler `GetAccessStatus` (GET) créé
- [x] Handler `GetRateLimitStatus` (GET) créé
- [x] Tests unitaires middlewares (9 tests)
- [x] Tests unitaires queue service (11 tests)
- [ ] Tests integration handlers (TODO après Docker up)
- [x] Migration SQL (déjà existante dans 000001_init_schema.up.sql)
- [x] Documentation complète (ce fichier)
- [ ] Integration dans main.go (TODO)
- [ ] Review sécurité (TODO)
- [ ] Monitoring metrics (TODO Phase 4)

---

## 📚 Ressources

- [Fiber Documentation](https://docs.gofiber.io/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [Go Validator](https://github.com/go-playground/validator)
- [Job Queue Patterns](https://www.cloudamqp.com/blog/when-to-use-rabbitmq-or-apache-kafka.html)
- [Rate Limiting Algorithms](https://blog.logrocket.com/rate-limiting-go-application/)

---

**Status:** ✅ **Phase 3 (Sprint 3) - Backend Letters API - COMPLET (avec mocks)**

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
