# 09. BACKEND_LETTERS_API.md

## 📋 Métadonnées

- **Phase:** 3
- **Priorité:** HAUTE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 08. BACKEND_AI_SERVICES.md
- **Temps estimé:** 3-4 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter l'API REST pour la génération de lettres de motivation et anti-motivation par IA, avec un système de contrôle d'accès basé sur le tracking des visiteurs, du rate limiting strict, et un système de queue pour les générations asynchrones.

**Fonctionnalités clés:**
- Génération de lettres duales (motivation + anti-motivation)
- Contrôle d'accès : 3 visites OU profil détecté
- Rate limiting : 5 générations/jour, cooldown 2 minutes
- Queue système pour jobs asynchrones
- Historique des générations
- Export PDF des lettres

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────┐
│   Frontend      │
│  LetterForm     │
└────────┬────────┘
         │ POST /api/letters/generate
         ▼
┌─────────────────────────────────────┐
│   Letters API Handler               │
│  ┌──────────────────────────────┐  │
│  │ 1. Access Gate Middleware    │  │
│  │    - Vérif 3 visites         │  │
│  │    - OU profil détecté       │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 2. Rate Limit Middleware     │  │
│  │    - Max 5/jour              │  │
│  │    - Cooldown 2min           │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 3. Job Queue                 │  │
│  │    - Async generation        │  │
│  │    - Status tracking         │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│   AI Services (08)                  │
│   - Scraper Service                 │
│   - Claude/GPT Service              │
│   - PDF Generation Service          │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│   Storage                           │
│   - PostgreSQL (historique)         │
│   - Redis (cache, rate limit)       │
└─────────────────────────────────────┘
```

### Design Decisions

**1. Accès basé sur visites (pas d'auth):**
- Tracking via cookies de session
- Compteur Redis incrémenté à chaque visite
- Permet accès progressif sans friction d'inscription
- Détection de profils pour accès immédiat (recruteurs, CTOs)

**2. Rate limiting strict:**
- Protection coûts API IA
- Limite 5 générations/jour par session
- Cooldown 2 minutes entre chaque génération
- Implémentation Redis avec expiration automatique

**3. Queue système asynchrone:**
- Génération IA peut prendre 10-30 secondes
- Évite timeouts HTTP
- Permet scaling horizontal
- Status polling ou WebSocket pour notifications

**4. Caching intelligent:**
- Cache par entreprise + hash du profil utilisateur
- TTL 24h (données entreprise relativement stables)
- Évite régénérations inutiles
- Économie coûts API

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Framework web (déjà installé en Phase 1)
go get github.com/gofiber/fiber/v2

# Validation
go get github.com/go-playground/validator/v10

# UUID pour job IDs
go get github.com/google/uuid

# WebSocket (optionnel, pour notifications temps réel)
go get github.com/gofiber/websocket/v2
```

### Services Externes

- **AI Services** : Implémentés dans doc 08
- **PostgreSQL** : Table `generated_letters`
- **Redis** : Cache, rate limiting, queue jobs

### Prérequis du Doc 08

- `internal/services/ai.go` : Service génération IA
- `internal/services/scraper.go` : Scraping infos entreprises
- `internal/services/pdf_letters.go` : Génération PDF lettres

---

## 🔨 Implémentation

### Étape 1: Models et Migrations

**Description:** Créer le model GORM pour l'historique des lettres générées.

**Fichier:** `backend/internal/models/generated_letter.go`

```go
package models

import (
    "time"
    "gorm.io/gorm"
)

type LetterType string

const (
    LetterTypeMotivation     LetterType = "motivation"
    LetterTypeAntiMotivation LetterType = "anti_motivation"
)

type GeneratedLetter struct {
    ID              uint           `gorm:"primarykey" json:"id"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

    // Référence visiteur (session ID)
    VisitorID       string         `gorm:"index;not null" json:"visitor_id"`

    // Infos entreprise
    CompanyName     string         `gorm:"not null;index" json:"company_name"`
    CompanyInfo     string         `gorm:"type:text" json:"company_info"` // JSON des infos scrapées

    // Lettres générées
    MotivationLetter      string `gorm:"type:text;not null" json:"motivation_letter"`
    AntiMotivationLetter  string `gorm:"type:text;not null" json:"anti_motivation_letter"`

    // Métadonnées génération
    AIModel         string         `json:"ai_model"`  // "claude" ou "gpt4"
    TokensUsed      int            `json:"tokens_used"`
    GenerationTime  int            `json:"generation_time"` // en millisecondes

    // PDF paths (stockage local ou S3)
    PDFMotivationPath     string `json:"pdf_motivation_path,omitempty"`
    PDFAntiMotivationPath string `json:"pdf_anti_motivation_path,omitempty"`
    PDFDualPath           string `json:"pdf_dual_path,omitempty"`
}

func (GeneratedLetter) TableName() string {
    return "generated_letters"
}
```

**Migration:** `backend/migrations/000005_create_generated_letters.up.sql`

```sql
CREATE TABLE IF NOT EXISTS generated_letters (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    visitor_id VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    company_info TEXT,

    motivation_letter TEXT NOT NULL,
    anti_motivation_letter TEXT NOT NULL,

    ai_model VARCHAR(50),
    tokens_used INTEGER DEFAULT 0,
    generation_time INTEGER DEFAULT 0,

    pdf_motivation_path VARCHAR(512),
    pdf_anti_motivation_path VARCHAR(512),
    pdf_dual_path VARCHAR(512)
);

CREATE INDEX idx_generated_letters_visitor_id ON generated_letters(visitor_id);
CREATE INDEX idx_generated_letters_company_name ON generated_letters(company_name);
CREATE INDEX idx_generated_letters_deleted_at ON generated_letters(deleted_at);
```

**Migration down:** `backend/migrations/000005_create_generated_letters.down.sql`

```sql
DROP TABLE IF EXISTS generated_letters;
```

**Explications:**
- `VisitorID` : permet de retrouver l'historique par session
- `CompanyInfo` : stocke les données scrapées (JSON) pour analyse ultérieure
- `TokensUsed` et `GenerationTime` : tracking coûts et performances
- Champs PDF paths : stockage des chemins vers PDFs générés

---

### Étape 2: Access Gate Middleware

**Description:** Middleware vérifiant l'accès aux fonctionnalités IA (3 visites OU profil détecté).

**Fichier:** `backend/internal/middleware/access_gate.go`

```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/go-redis/redis/v8"
    "context"
    "fmt"
)

type AccessGateConfig struct {
    Redis              *redis.Client
    MinVisits          int  // 3 visites minimum
    BypassOnProfile    bool // true = profils détectés bypass
}

// AccessGate vérifie si le visiteur a accès aux fonctionnalités IA
func AccessGate(config AccessGateConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        ctx := context.Background()

        // Récupérer le session ID depuis le cookie
        sessionID := c.Cookies("session_id")
        if sessionID == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Session non trouvée",
                "code":  "SESSION_REQUIRED",
            })
        }

        // Vérifier si profil détecté (bypass)
        if config.BypassOnProfile {
            profileKey := fmt.Sprintf("visitor:%s:profile", sessionID)
            profile, err := config.Redis.Get(ctx, profileKey).Result()

            if err == nil && profile != "" {
                // Profil détecté (recruiter, tech_lead, cto, ceo)
                c.Locals("access_granted_reason", "profile_detected")
                c.Locals("profile_type", profile)
                return c.Next()
            }
        }

        // Récupérer le compteur de visites
        visitKey := fmt.Sprintf("visitor:%s:count", sessionID)
        visitCount, err := config.Redis.Get(ctx, visitKey).Int()

        if err != nil && err != redis.Nil {
            return c.Status(500).JSON(fiber.Map{
                "error": "Erreur lors de la vérification d'accès",
                "code":  "INTERNAL_ERROR",
            })
        }

        // Vérifier le nombre de visites
        if visitCount < config.MinVisits {
            visitsRemaining := config.MinVisits - visitCount

            return c.Status(403).JSON(fiber.Map{
                "error": "Accès refusé",
                "code":  "INSUFFICIENT_VISITS",
                "details": fiber.Map{
                    "current_visits":  visitCount,
                    "required_visits": config.MinVisits,
                    "visits_remaining": visitsRemaining,
                },
                "message": fmt.Sprintf(
                    "Les fonctionnalités IA sont disponibles à partir de la %dème visite. "+
                    "Encore %d visite(s) nécessaire(s).",
                    config.MinVisits,
                    visitsRemaining,
                ),
                "teaser": "Découvrez la génération automatique de lettres de motivation " +
                         "et anti-motivation personnalisées par IA !",
            })
        }

        // Accès accordé
        c.Locals("access_granted_reason", "visits_threshold")
        c.Locals("visit_count", visitCount)
        return c.Next()
    }
}
```

**Explications:**
- Vérifie d'abord si un profil est détecté (bypass immédiat)
- Sinon vérifie le compteur de visites (minimum 3)
- Retourne 403 avec message clair si accès refusé
- Inclut un "teaser" pour encourager les visites supplémentaires
- Stocke la raison d'accès dans `c.Locals()` pour logging

---

### Étape 3: Rate Limiting Spécifique IA

**Description:** Rate limiter dédié pour l'API IA (5/jour, cooldown 2min).

**Fichier:** `backend/internal/middleware/ai_ratelimit.go`

```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/go-redis/redis/v8"
    "context"
    "fmt"
    "time"
)

type AIRateLimitConfig struct {
    Redis              *redis.Client
    MaxPerDay          int           // 5 générations par jour
    CooldownDuration   time.Duration // 2 minutes entre générations
}

// AIRateLimit limite les générations IA par session
func AIRateLimit(config AIRateLimitConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        ctx := context.Background()

        sessionID := c.Cookies("session_id")
        if sessionID == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Session non trouvée",
                "code":  "SESSION_REQUIRED",
            })
        }

        // --- Vérification Cooldown (2 minutes) ---
        cooldownKey := fmt.Sprintf("ratelimit:ai:%s:cooldown", sessionID)
        cooldownExists, err := config.Redis.Exists(ctx, cooldownKey).Result()

        if err != nil {
            return c.Status(500).JSON(fiber.Map{
                "error": "Erreur rate limiting",
                "code":  "INTERNAL_ERROR",
            })
        }

        if cooldownExists > 0 {
            ttl, _ := config.Redis.TTL(ctx, cooldownKey).Result()

            return c.Status(429).JSON(fiber.Map{
                "error": "Cooldown actif",
                "code":  "COOLDOWN_ACTIVE",
                "retry_after": int(ttl.Seconds()),
                "message": fmt.Sprintf(
                    "Merci de patienter %d secondes avant la prochaine génération.",
                    int(ttl.Seconds()),
                ),
            })
        }

        // --- Vérification Limite Journalière (5/jour) ---
        dailyKey := fmt.Sprintf("ratelimit:ai:%s:daily", sessionID)
        count, err := config.Redis.Get(ctx, dailyKey).Int()

        if err != nil && err != redis.Nil {
            return c.Status(500).JSON(fiber.Map{
                "error": "Erreur rate limiting",
                "code":  "INTERNAL_ERROR",
            })
        }

        if count >= config.MaxPerDay {
            ttl, _ := config.Redis.TTL(ctx, dailyKey).Result()

            return c.Status(429).JSON(fiber.Map{
                "error": "Limite journalière atteinte",
                "code":  "DAILY_LIMIT_REACHED",
                "details": fiber.Map{
                    "max_per_day": config.MaxPerDay,
                    "used":        count,
                    "reset_in":    int(ttl.Seconds()),
                },
                "message": fmt.Sprintf(
                    "Vous avez atteint la limite de %d générations par jour. "+
                    "Réinitialisation dans %s.",
                    config.MaxPerDay,
                    formatDuration(ttl),
                ),
            })
        }

        // Passer les infos au handler pour incrémentation après succès
        c.Locals("rate_limit_session_id", sessionID)
        c.Locals("rate_limit_daily_key", dailyKey)
        c.Locals("rate_limit_cooldown_key", cooldownKey)
        c.Locals("rate_limit_remaining", config.MaxPerDay - count - 1)

        return c.Next()
    }
}

// IncrementAIRateLimit à appeler APRÈS génération réussie
func IncrementAIRateLimit(c *fiber.Ctx, redis *redis.Client, cooldownDuration time.Duration) error {
    ctx := context.Background()

    dailyKey := c.Locals("rate_limit_daily_key").(string)
    cooldownKey := c.Locals("rate_limit_cooldown_key").(string)

    // Incrémenter compteur journalier
    count, err := redis.Incr(ctx, dailyKey).Result()
    if err != nil {
        return err
    }

    // Si premier incrémentation, set expiration à minuit
    if count == 1 {
        now := time.Now()
        midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
        ttl := midnight.Sub(now)
        redis.Expire(ctx, dailyKey, ttl)
    }

    // Activer cooldown (2 minutes)
    redis.Set(ctx, cooldownKey, "1", cooldownDuration)

    return nil
}

func formatDuration(d time.Duration) string {
    hours := int(d.Hours())
    minutes := int(d.Minutes()) % 60

    if hours > 0 {
        return fmt.Sprintf("%dh%dm", hours, minutes)
    }
    return fmt.Sprintf("%dm", minutes)
}
```

**Explications:**
- **Cooldown** : empêche générations successives rapides (2 min)
- **Limite journalière** : max 5 générations par jour par session
- **Expiration intelligente** : daily counter expire à minuit (TTL dynamique)
- **Incrémentation post-génération** : n'incrémente que si génération réussie
- Headers `Retry-After` pour indiquer au client quand réessayer

---

### Étape 4: Request et Response DTOs

**Description:** Structures de données pour les requêtes et réponses API.

**Fichier:** `backend/internal/api/dto/letters.go`

```go
package dto

import (
    "github.com/go-playground/validator/v10"
)

// --- REQUESTS ---

type GenerateLetterRequest struct {
    CompanyName string `json:"company_name" validate:"required,min=2,max=200"`
}

func (r *GenerateLetterRequest) Validate() error {
    validate := validator.New()
    return validate.Struct(r)
}

// --- RESPONSES ---

type LetterGenerationResponse struct {
    JobID     string `json:"job_id"`
    Status    string `json:"status"` // "queued", "processing", "completed", "failed"
    Message   string `json:"message"`
}

type LetterJobStatus struct {
    JobID      string  `json:"job_id"`
    Status     string  `json:"status"`
    Progress   int     `json:"progress"` // 0-100

    // Si completed
    LetterID   *uint   `json:"letter_id,omitempty"`

    // Si failed
    Error      *string `json:"error,omitempty"`

    // Temps estimé restant (secondes)
    EstimatedTime *int `json:"estimated_time,omitempty"`
}

type LetterDetailResponse struct {
    ID                    uint      `json:"id"`
    CompanyName           string    `json:"company_name"`
    MotivationLetter      string    `json:"motivation_letter"`
    AntiMotivationLetter  string    `json:"anti_motivation_letter"`
    CreatedAt             string    `json:"created_at"`

    // URLs de téléchargement PDF
    PDFMotivationURL      string    `json:"pdf_motivation_url"`
    PDFAntiMotivationURL  string    `json:"pdf_anti_motivation_url"`
    PDFDualURL            string    `json:"pdf_dual_url"`
}

type LetterHistoryResponse struct {
    Letters []LetterHistoryItem `json:"letters"`
    Total   int                 `json:"total"`
    Page    int                 `json:"page"`
    PerPage int                 `json:"per_page"`
}

type LetterHistoryItem struct {
    ID          uint   `json:"id"`
    CompanyName string `json:"company_name"`
    CreatedAt   string `json:"created_at"`
}
```

---

### Étape 5: Job Queue System

**Description:** Système de queue pour générations asynchrones avec Redis.

**Fichier:** `backend/internal/services/letter_queue.go`

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
)

type JobStatus string

const (
    JobStatusQueued     JobStatus = "queued"
    JobStatusProcessing JobStatus = "processing"
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
)

type LetterJob struct {
    JobID       string    `json:"job_id"`
    VisitorID   string    `json:"visitor_id"`
    CompanyName string    `json:"company_name"`
    Status      JobStatus `json:"status"`
    Progress    int       `json:"progress"` // 0-100

    // Résultats
    LetterID    *uint     `json:"letter_id,omitempty"`
    Error       *string   `json:"error,omitempty"`

    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type LetterQueueService struct {
    redis *redis.Client
    ctx   context.Context
}

func NewLetterQueueService(redis *redis.Client) *LetterQueueService {
    return &LetterQueueService{
        redis: redis,
        ctx:   context.Background(),
    }
}

// EnqueueJob ajoute un job dans la queue
func (s *LetterQueueService) EnqueueJob(visitorID, companyName string) (string, error) {
    jobID := uuid.New().String()

    job := LetterJob{
        JobID:       jobID,
        VisitorID:   visitorID,
        CompanyName: companyName,
        Status:      JobStatusQueued,
        Progress:    0,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    // Sérialiser le job
    jobJSON, err := json.Marshal(job)
    if err != nil {
        return "", err
    }

    // Stocker le job dans Redis (Hash)
    jobKey := fmt.Sprintf("job:letter:%s", jobID)
    err = s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err() // TTL 24h
    if err != nil {
        return "", err
    }

    // Ajouter à la queue (List)
    queueKey := "queue:letters"
    err = s.redis.RPush(s.ctx, queueKey, jobID).Err()
    if err != nil {
        return "", err
    }

    return jobID, nil
}

// GetJobStatus récupère le status d'un job
func (s *LetterQueueService) GetJobStatus(jobID string) (*LetterJob, error) {
    jobKey := fmt.Sprintf("job:letter:%s", jobID)

    jobJSON, err := s.redis.Get(s.ctx, jobKey).Result()
    if err == redis.Nil {
        return nil, fmt.Errorf("job non trouvé")
    }
    if err != nil {
        return nil, err
    }

    var job LetterJob
    err = json.Unmarshal([]byte(jobJSON), &job)
    if err != nil {
        return nil, err
    }

    return &job, nil
}

// UpdateJobStatus met à jour le status d'un job
func (s *LetterQueueService) UpdateJobStatus(jobID string, status JobStatus, progress int) error {
    job, err := s.GetJobStatus(jobID)
    if err != nil {
        return err
    }

    job.Status = status
    job.Progress = progress
    job.UpdatedAt = time.Now()

    jobJSON, err := json.Marshal(job)
    if err != nil {
        return err
    }

    jobKey := fmt.Sprintf("job:letter:%s", jobID)
    return s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err()
}

// CompleteJob marque un job comme complété
func (s *LetterQueueService) CompleteJob(jobID string, letterID uint) error {
    job, err := s.GetJobStatus(jobID)
    if err != nil {
        return err
    }

    job.Status = JobStatusCompleted
    job.Progress = 100
    job.LetterID = &letterID
    job.UpdatedAt = time.Now()

    jobJSON, err := json.Marshal(job)
    if err != nil {
        return err
    }

    jobKey := fmt.Sprintf("job:letter:%s", jobID)
    return s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err()
}

// FailJob marque un job comme échoué
func (s *LetterQueueService) FailJob(jobID string, errorMsg string) error {
    job, err := s.GetJobStatus(jobID)
    if err != nil {
        return err
    }

    job.Status = JobStatusFailed
    job.Error = &errorMsg
    job.UpdatedAt = time.Now()

    jobJSON, err := json.Marshal(job)
    if err != nil {
        return err
    }

    jobKey := fmt.Sprintf("job:letter:%s", jobID)
    return s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err()
}

// PopJob récupère le prochain job de la queue (pour worker)
func (s *LetterQueueService) PopJob() (string, error) {
    queueKey := "queue:letters"

    result, err := s.redis.LPop(s.ctx, queueKey).Result()
    if err == redis.Nil {
        return "", nil // Queue vide
    }
    if err != nil {
        return "", err
    }

    return result, nil
}
```

**Explications:**
- Jobs stockés dans Redis avec TTL 24h (auto-cleanup)
- Queue Redis List (FIFO) pour traitement séquentiel
- Status tracking avec progress (pour UI progress bar)
- Worker peut pop jobs de la queue et les traiter

---

### Étape 6: Letter Worker (Background Processing)

**Description:** Worker qui traite les jobs de génération de lettres.

**Fichier:** `backend/internal/workers/letter_worker.go`

```go
package workers

import (
    "context"
    "log"
    "time"
    "gorm.io/gorm"

    "maicivy/internal/services"
    "maicivy/internal/models"
)

type LetterWorker struct {
    db              *gorm.DB
    queueService    *services.LetterQueueService
    aiService       *services.AIService
    scraperService  *services.ScraperService
    pdfService      *services.PDFLetterService

    stopChan        chan bool
}

func NewLetterWorker(
    db *gorm.DB,
    queueService *services.LetterQueueService,
    aiService *services.AIService,
    scraperService *services.ScraperService,
    pdfService *services.PDFLetterService,
) *LetterWorker {
    return &LetterWorker{
        db:             db,
        queueService:   queueService,
        aiService:      aiService,
        scraperService: scraperService,
        pdfService:     pdfService,
        stopChan:       make(chan bool),
    }
}

// Start démarre le worker
func (w *LetterWorker) Start() {
    log.Println("Letter Worker started")

    ticker := time.NewTicker(2 * time.Second) // Poll queue every 2s
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            w.processNextJob()
        case <-w.stopChan:
            log.Println("Letter Worker stopped")
            return
        }
    }
}

// Stop arrête le worker
func (w *LetterWorker) Stop() {
    w.stopChan <- true
}

// processNextJob traite le prochain job dans la queue
func (w *LetterWorker) processNextJob() {
    jobID, err := w.queueService.PopJob()
    if err != nil {
        log.Printf("Error popping job: %v", err)
        return
    }

    if jobID == "" {
        return // Queue vide
    }

    log.Printf("Processing job: %s", jobID)

    // Récupérer les détails du job
    job, err := w.queueService.GetJobStatus(jobID)
    if err != nil {
        log.Printf("Error getting job status: %v", err)
        return
    }

    // Marquer comme en cours
    w.queueService.UpdateJobStatus(jobID, services.JobStatusProcessing, 10)

    // Exécuter la génération
    letterID, err := w.generateLetter(job)
    if err != nil {
        log.Printf("Error generating letter: %v", err)
        w.queueService.FailJob(jobID, err.Error())
        return
    }

    // Marquer comme complété
    w.queueService.CompleteJob(jobID, letterID)
    log.Printf("Job %s completed. Letter ID: %d", jobID, letterID)
}

// generateLetter génère les lettres et PDFs
func (w *LetterWorker) generateLetter(job *services.LetterJob) (uint, error) {
    ctx := context.Background()

    // 1. Scraper infos entreprise (20% progress)
    w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 20)

    companyInfo, err := w.scraperService.GetCompanyInfo(ctx, job.CompanyName)
    if err != nil {
        log.Printf("Scraper warning: %v (continuing with limited data)", err)
        companyInfo = &services.CompanyInfo{Name: job.CompanyName}
    }

    // 2. Génération lettre motivation (40% progress)
    w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 40)

    motivationLetter, err := w.aiService.GenerateMotivationLetter(ctx, companyInfo)
    if err != nil {
        return 0, err
    }

    // 3. Génération lettre anti-motivation (60% progress)
    w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 60)

    antiMotivationLetter, err := w.aiService.GenerateAntiMotivationLetter(ctx, companyInfo)
    if err != nil {
        return 0, err
    }

    // 4. Sauvegarder en DB (70% progress)
    w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 70)

    letter := models.GeneratedLetter{
        VisitorID:            job.VisitorID,
        CompanyName:          companyInfo.Name,
        CompanyInfo:          companyInfo.ToJSON(),
        MotivationLetter:     motivationLetter.Content,
        AntiMotivationLetter: antiMotivationLetter.Content,
        AIModel:              motivationLetter.Model,
        TokensUsed:           motivationLetter.TokensUsed + antiMotivationLetter.TokensUsed,
        GenerationTime:       int(time.Since(job.CreatedAt).Milliseconds()),
    }

    result := w.db.Create(&letter)
    if result.Error != nil {
        return 0, result.Error
    }

    // 5. Générer PDFs (80-100% progress)
    w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 80)

    pdfPaths, err := w.pdfService.GenerateLetterPDFs(ctx, &letter)
    if err != nil {
        log.Printf("PDF generation warning: %v (letters saved without PDFs)", err)
    } else {
        // Mettre à jour les paths PDFs
        letter.PDFMotivationPath = pdfPaths.MotivationPath
        letter.PDFAntiMotivationPath = pdfPaths.AntiMotivationPath
        letter.PDFDualPath = pdfPaths.DualPath
        w.db.Save(&letter)
    }

    return letter.ID, nil
}
```

**Explications:**
- Worker tourne en background (goroutine)
- Poll la queue toutes les 2 secondes
- Traite les jobs séquentiellement
- Met à jour le progress pour UI
- Gère les erreurs gracefully (continue même si scraper échoue)

---

### Étape 7: API Handlers

**Description:** Handlers HTTP pour les endpoints de génération et consultation de lettres.

**Fichier:** `backend/internal/api/letters.go`

```go
package api

import (
    "fmt"
    "strconv"
    "github.com/gofiber/fiber/v2"
    "gorm.io/gorm"

    "maicivy/internal/api/dto"
    "maicivy/internal/models"
    "maicivy/internal/services"
    "maicivy/internal/middleware"
)

type LettersHandler struct {
    db           *gorm.DB
    queueService *services.LetterQueueService
}

func NewLettersHandler(db *gorm.DB, queueService *services.LetterQueueService) *LettersHandler {
    return &LettersHandler{
        db:           db,
        queueService: queueService,
    }
}

// GenerateLetter génère une lettre de motivation (asynchrone)
// POST /api/letters/generate
func (h *LettersHandler) GenerateLetter(c *fiber.Ctx) error {
    // Parser request
    var req dto.GenerateLetterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
            "code":  "INVALID_REQUEST",
        })
    }

    // Valider
    if err := req.Validate(); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Validation failed",
            "code":  "VALIDATION_ERROR",
            "details": err.Error(),
        })
    }

    // Récupérer session ID
    sessionID := c.Cookies("session_id")

    // Enqueue job
    jobID, err := h.queueService.EnqueueJob(sessionID, req.CompanyName)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to enqueue generation job",
            "code":  "QUEUE_ERROR",
        })
    }

    // Incrémenter rate limit (APRÈS enqueue success)
    redisClient := c.Locals("redis").(*redis.Client)
    cooldownDuration := 2 * time.Minute

    if err := middleware.IncrementAIRateLimit(c, redisClient, cooldownDuration); err != nil {
        // Log error but don't fail request (job already queued)
        log.Printf("Failed to increment rate limit: %v", err)
    }

    // Retourner job ID pour polling
    remaining := c.Locals("rate_limit_remaining").(int)

    return c.Status(202).JSON(dto.LetterGenerationResponse{
        JobID:   jobID,
        Status:  "queued",
        Message: fmt.Sprintf(
            "Génération en cours. Encore %d génération(s) disponible(s) aujourd'hui.",
            remaining,
        ),
    })
}

// GetJobStatus récupère le status d'un job
// GET /api/letters/jobs/:jobId
func (h *LettersHandler) GetJobStatus(c *fiber.Ctx) error {
    jobID := c.Params("jobId")

    job, err := h.queueService.GetJobStatus(jobID)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{
            "error": "Job non trouvé",
            "code":  "JOB_NOT_FOUND",
        })
    }

    // Calculer temps estimé si en cours
    var estimatedTime *int
    if job.Status == services.JobStatusProcessing || job.Status == services.JobStatusQueued {
        // Estimation basique: 30s total, progress donne position
        remaining := int((100 - job.Progress) * 30 / 100)
        estimatedTime = &remaining
    }

    return c.JSON(dto.LetterJobStatus{
        JobID:         job.JobID,
        Status:        string(job.Status),
        Progress:      job.Progress,
        LetterID:      job.LetterID,
        Error:         job.Error,
        EstimatedTime: estimatedTime,
    })
}

// GetLetter récupère une lettre générée
// GET /api/letters/:id
func (h *LettersHandler) GetLetter(c *fiber.Ctx) error {
    letterID, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid letter ID",
            "code":  "INVALID_ID",
        })
    }

    // Récupérer session ID pour vérifier ownership
    sessionID := c.Cookies("session_id")

    var letter models.GeneratedLetter
    result := h.db.Where("id = ? AND visitor_id = ?", letterID, sessionID).First(&letter)

    if result.Error != nil {
        if result.Error == gorm.ErrRecordNotFound {
            return c.Status(404).JSON(fiber.Map{
                "error": "Lettre non trouvée",
                "code":  "LETTER_NOT_FOUND",
            })
        }
        return c.Status(500).JSON(fiber.Map{
            "error": "Database error",
            "code":  "DB_ERROR",
        })
    }

    // Construire URLs de téléchargement PDF
    baseURL := c.BaseURL()

    return c.JSON(dto.LetterDetailResponse{
        ID:                   letter.ID,
        CompanyName:          letter.CompanyName,
        MotivationLetter:     letter.MotivationLetter,
        AntiMotivationLetter: letter.AntiMotivationLetter,
        CreatedAt:            letter.CreatedAt.Format("2006-01-02 15:04:05"),
        PDFMotivationURL:     fmt.Sprintf("%s/api/letters/%d/pdf?type=motivation", baseURL, letter.ID),
        PDFAntiMotivationURL: fmt.Sprintf("%s/api/letters/%d/pdf?type=anti_motivation", baseURL, letter.ID),
        PDFDualURL:           fmt.Sprintf("%s/api/letters/%d/pdf?type=dual", baseURL, letter.ID),
    })
}

// DownloadPDF télécharge un PDF de lettre
// GET /api/letters/:id/pdf?type=motivation|anti_motivation|dual
func (h *LettersHandler) DownloadPDF(c *fiber.Ctx) error {
    letterID, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid letter ID",
            "code":  "INVALID_ID",
        })
    }

    pdfType := c.Query("type", "dual")
    sessionID := c.Cookies("session_id")

    var letter models.GeneratedLetter
    result := h.db.Where("id = ? AND visitor_id = ?", letterID, sessionID).First(&letter)

    if result.Error != nil {
        return c.Status(404).JSON(fiber.Map{
            "error": "Lettre non trouvée",
            "code":  "LETTER_NOT_FOUND",
        })
    }

    // Déterminer quel PDF servir
    var pdfPath string
    var filename string

    switch pdfType {
    case "motivation":
        pdfPath = letter.PDFMotivationPath
        filename = fmt.Sprintf("lettre_motivation_%s.pdf", letter.CompanyName)
    case "anti_motivation":
        pdfPath = letter.PDFAntiMotivationPath
        filename = fmt.Sprintf("lettre_anti_motivation_%s.pdf", letter.CompanyName)
    case "dual":
        pdfPath = letter.PDFDualPath
        filename = fmt.Sprintf("lettres_%s.pdf", letter.CompanyName)
    default:
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid PDF type",
            "code":  "INVALID_TYPE",
        })
    }

    if pdfPath == "" {
        return c.Status(404).JSON(fiber.Map{
            "error": "PDF non disponible",
            "code":  "PDF_NOT_FOUND",
        })
    }

    // Headers pour téléchargement
    c.Set("Content-Type", "application/pdf")
    c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

    return c.SendFile(pdfPath)
}

// GetHistory récupère l'historique des lettres générées
// GET /api/letters/history?page=1&per_page=10
func (h *LettersHandler) GetHistory(c *fiber.Ctx) error {
    sessionID := c.Cookies("session_id")

    page, _ := strconv.Atoi(c.Query("page", "1"))
    perPage, _ := strconv.Atoi(c.Query("per_page", "10"))

    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 50 {
        perPage = 10
    }

    offset := (page - 1) * perPage

    var letters []models.GeneratedLetter
    var total int64

    h.db.Model(&models.GeneratedLetter{}).Where("visitor_id = ?", sessionID).Count(&total)

    result := h.db.Where("visitor_id = ?", sessionID).
        Order("created_at DESC").
        Limit(perPage).
        Offset(offset).
        Find(&letters)

    if result.Error != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Database error",
            "code":  "DB_ERROR",
        })
    }

    // Mapper vers DTO
    items := make([]dto.LetterHistoryItem, len(letters))
    for i, letter := range letters {
        items[i] = dto.LetterHistoryItem{
            ID:          letter.ID,
            CompanyName: letter.CompanyName,
            CreatedAt:   letter.CreatedAt.Format("2006-01-02 15:04:05"),
        }
    }

    return c.JSON(dto.LetterHistoryResponse{
        Letters: items,
        Total:   int(total),
        Page:    page,
        PerPage: perPage,
    })
}
```

**Explications:**
- `GenerateLetter` : enqueue job, retourne 202 Accepted
- `GetJobStatus` : polling endpoint pour UI (progress bar)
- `GetLetter` : récupère lettre complète après génération
- `DownloadPDF` : streaming fichier PDF
- `GetHistory` : pagination de l'historique
- Vérification ownership via `visitor_id` (sécurité)

---

### Étape 8: Routes Setup

**Description:** Configuration des routes avec middlewares.

**Fichier:** `backend/cmd/main.go` (excerpt)

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
    "maicivy/internal/database"
)

func main() {
    // ... Setup DB, Redis, services ...

    app := fiber.New()

    // Services
    queueService := services.NewLetterQueueService(redisClient)
    aiService := services.NewAIService(/* config */)
    scraperService := services.NewScraperService(/* config */)
    pdfService := services.NewPDFLetterService(/* config */)

    // Handlers
    lettersHandler := api.NewLettersHandler(db, queueService)

    // Start worker
    worker := workers.NewLetterWorker(db, queueService, aiService, scraperService, pdfService)
    go worker.Start()
    defer worker.Stop()

    // --- ROUTES LETTRES IA ---
    lettersGroup := app.Group("/api/letters")

    // Middleware Access Gate + Rate Limit pour génération
    lettersGroup.Post("/generate",
        middleware.AccessGate(middleware.AccessGateConfig{
            Redis:           redisClient,
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

    // Status job (pas de rate limit)
    lettersGroup.Get("/jobs/:jobId", lettersHandler.GetJobStatus)

    // Récupération lettre (pas de rate limit, mais ownership check)
    lettersGroup.Get("/:id", lettersHandler.GetLetter)

    // Téléchargement PDF (pas de rate limit)
    lettersGroup.Get("/:id/pdf", lettersHandler.DownloadPDF)

    // Historique (pas de rate limit)
    lettersGroup.Get("/history", lettersHandler.GetHistory)

    log.Fatal(app.Listen(":3000"))
}
```

**Explications:**
- Access Gate + Rate Limit UNIQUEMENT sur `/generate`
- Worker démarre en goroutine au lancement app
- Autres endpoints accessibles sans restrictions (ownership check suffit)

---

## 🧪 Tests

### Tests Unitaires

**Fichier:** `backend/internal/middleware/access_gate_test.go`

```go
package middleware

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/gofiber/fiber/v2"
    "github.com/alicebob/miniredis/v2"
    "github.com/go-redis/redis/v8"
)

func TestAccessGate_InsufficientVisits(t *testing.T) {
    // Setup mini Redis
    mr, _ := miniredis.Run()
    defer mr.Close()

    redisClient := redis.NewClient(&redis.Options{
        Addr: mr.Addr(),
    })

    // Setup Fiber app
    app := fiber.New()

    app.Use(AccessGate(AccessGateConfig{
        Redis:     redisClient,
        MinVisits: 3,
    }))

    app.Get("/test", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // Simuler 2 visites (insuffisant)
    sessionID := "test-session"
    redisClient.Set(ctx, "visitor:"+sessionID+":count", 2, 0)

    // Request
    req := httptest.NewRequest("GET", "/test", nil)
    req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})

    resp, _ := app.Test(req)

    assert.Equal(t, 403, resp.StatusCode)
}

func TestAccessGate_ProfileBypass(t *testing.T) {
    mr, _ := miniredis.Run()
    defer mr.Close()

    redisClient := redis.NewClient(&redis.Options{
        Addr: mr.Addr(),
    })

    app := fiber.New()

    app.Use(AccessGate(AccessGateConfig{
        Redis:           redisClient,
        MinVisits:       3,
        BypassOnProfile: true,
    }))

    app.Get("/test", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // Simuler profil détecté (0 visites mais profil = recruiter)
    sessionID := "recruiter-session"
    redisClient.Set(ctx, "visitor:"+sessionID+":count", 0, 0)
    redisClient.Set(ctx, "visitor:"+sessionID+":profile", "recruiter", 0)

    req := httptest.NewRequest("GET", "/test", nil)
    req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})

    resp, _ := app.Test(req)

    assert.Equal(t, 200, resp.StatusCode) // Bypass réussi
}
```

### Tests Integration

**Fichier:** `backend/internal/api/letters_test.go`

```go
package api

import (
    "testing"
    "bytes"
    "encoding/json"
    "github.com/stretchr/testify/assert"
)

func TestGenerateLetter_Success(t *testing.T) {
    // Setup test DB, Redis, services
    // ...

    reqBody := map[string]string{
        "company_name": "Anthropic",
    }
    jsonBody, _ := json.Marshal(reqBody)

    req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})

    resp, _ := app.Test(req)

    assert.Equal(t, 202, resp.StatusCode)

    var response map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&response)

    assert.NotEmpty(t, response["job_id"])
    assert.Equal(t, "queued", response["status"])
}

func TestGenerateLetter_RateLimitExceeded(t *testing.T) {
    // Simuler 5 générations déjà faites aujourd'hui
    sessionID := "limited-session"
    redisClient.Set(ctx, "ratelimit:ai:"+sessionID+":daily", 5, 24*time.Hour)

    reqBody := map[string]string{
        "company_name": "Test Corp",
    }
    jsonBody, _ := json.Marshal(reqBody)

    req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})

    resp, _ := app.Test(req)

    assert.Equal(t, 429, resp.StatusCode)

    var response map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&response)

    assert.Equal(t, "DAILY_LIMIT_REACHED", response["code"])
}
```

### Commandes

```bash
# Tests unitaires
go test -v ./internal/middleware/...
go test -v ./internal/api/...

# Tests integration
go test -v -tags=integration ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## ⚠️ Points d'Attention

### Sécurité

- ⚠️ **Validation Input** : Toujours valider `company_name` (XSS, injection)
- ⚠️ **Ownership Check** : Vérifier `visitor_id` avant de retourner lettres
- ⚠️ **Rate Limit Bypass** : Surveiller tentatives de contournement (rotation session_id)
- ⚠️ **Cost Control** : Monitorer tokens IA utilisés (alertes si dépassement budget)

### Performance

- 💡 **Cache Redis** : Cacher lettres par `hash(company_name + profile)` (éviter régénérations)
- 💡 **Queue Size** : Monitorer taille queue (alerte si > 100 jobs en attente)
- 💡 **Worker Scaling** : Possibilité de lancer plusieurs workers en parallèle
- ⚠️ **PDF Storage** : Nettoyer vieux PDFs (cronjob cleanup > 30 jours)

### Edge Cases

- ⚠️ **Scraper Failure** : Continuer génération même si scraper échoue (fallback données minimales)
- ⚠️ **AI Timeout** : Timeout 60s sur appels API IA (retry avec fallback model)
- ⚠️ **Job Expiration** : Jobs Redis TTL 24h → nettoyer automatiquement
- ⚠️ **Concurrent Requests** : Un seul job actif par session (vérifier avant enqueue)

### UX

- 💡 **Progress Updates** : WebSocket optionnel pour push temps réel vs polling
- 💡 **Error Messages** : Messages clairs et actionnables (retry, contact, etc.)
- 💡 **Estimated Time** : Calculer temps restant basé sur progress actuel
- 💡 **Teaser Design** : Rendre le message de blocage (3 visites) engageant

---

## 📚 Ressources

- [Fiber Documentation](https://docs.gofiber.io/)
- [GORM Guide](https://gorm.io/docs/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [Go Validator](https://github.com/go-playground/validator)
- [Job Queue Patterns](https://www.cloudamqp.com/blog/when-to-use-rabbitmq-or-apache-kafka.html)
- [Rate Limiting Algorithms](https://blog.logrocket.com/rate-limiting-go-application/)
- [WebSocket avec Fiber](https://docs.gofiber.io/api/middleware/websocket)

---

## ✅ Checklist de Complétion

- [ ] Model `GeneratedLetter` créé et migré
- [ ] Middleware `AccessGate` implémenté et testé
- [ ] Middleware `AIRateLimit` implémenté et testé
- [ ] Service `LetterQueueService` implémenté
- [ ] Worker `LetterWorker` implémenté et fonctionnel
- [ ] Handler `GenerateLetter` (POST) testé
- [ ] Handler `GetJobStatus` (GET) testé
- [ ] Handler `GetLetter` (GET) testé
- [ ] Handler `DownloadPDF` (GET) testé
- [ ] Handler `GetHistory` (GET) testé
- [ ] Tests unitaires (coverage > 80%)
- [ ] Tests integration end-to-end
- [ ] Documentation OpenAPI mise à jour
- [ ] Review sécurité (validation, ownership)
- [ ] Review performance (caching, queue size)
- [ ] Monitoring metrics ajoutés (Prometheus)
- [ ] Logs structurés ajoutés
- [ ] Worker démarre automatiquement avec app
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
