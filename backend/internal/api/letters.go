package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/api/dto"
	"maicivy/internal/middleware"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

// LettersHandler handler pour les endpoints de génération de lettres
type LettersHandler struct {
	db           *gorm.DB
	redis        *redis.Client
	queueService services.LetterQueueServiceInterface
}

// NewLettersHandler crée une nouvelle instance du handler
func NewLettersHandler(db *gorm.DB, redis *redis.Client, queueService services.LetterQueueServiceInterface) *LettersHandler {
	return &LettersHandler{
		db:           db,
		redis:        redis,
		queueService: queueService,
	}
}

// GenerateLetter génère une paire de lettres (motivation + anti-motivation) de façon asynchrone
// POST /api/v1/letters/generate
func (h *LettersHandler) GenerateLetter(c *fiber.Ctx) error {
	// Parser request
	var req dto.GenerateLetterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"code":    "INVALID_REQUEST",
			"details": err.Error(),
		})
	}

	// Valider
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"code":    "VALIDATION_ERROR",
			"details": err.Error(),
		})
	}

	// Récupérer session ID depuis les locals (mis par tracking middleware)
	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}

	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Enqueue job
	jobID, err := h.queueService.EnqueueJob(sessionID, req.CompanyName, req.JobTitle, req.Theme)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to enqueue generation job",
			"code":    "QUEUE_ERROR",
			"details": err.Error(),
		})
	}

	// Incrémenter rate limit (APRÈS enqueue success)
	cooldownDuration := 2 * time.Minute
	if err := middleware.IncrementAIRateLimit(c, h.redis, cooldownDuration); err != nil {
		// Log error but don't fail request (job already queued)
		fmt.Printf("Failed to increment rate limit: %v\n", err)
	}

	// Récupérer remaining depuis les locals
	remaining, ok := c.Locals("rate_limit_remaining").(int)
	if !ok {
		remaining = 0
	}

	// Retourner job ID pour polling
	return c.Status(fiber.StatusAccepted).JSON(dto.LetterGenerationResponse{
		JobID:  jobID,
		Status: "queued",
		Message: fmt.Sprintf(
			"Génération en cours. Encore %d génération(s) disponible(s) aujourd'hui.",
			remaining,
		),
		RateLimitRemaining: remaining,
	})
}

// GetJobStatus récupère le status d'un job de génération
// GET /api/v1/letters/jobs/:jobId
func (h *LettersHandler) GetJobStatus(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	job, err := h.queueService.GetJobStatus(jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Job non trouvé",
			"code":    "JOB_NOT_FOUND",
			"details": err.Error(),
		})
	}

	// Calculer temps estimé si en cours
	var estimatedTime *int
	if job.Status == services.JobStatusProcessing || job.Status == services.JobStatusQueued {
		remaining := job.EstimateRemainingTime()
		estimatedTime = &remaining
	}

	// Convert uint IDs to string if present
	var motivationIDStr, antiMotivationIDStr *string
	if job.LetterMotivationID != nil {
		idStr := fmt.Sprintf("%d", *job.LetterMotivationID)
		motivationIDStr = &idStr
	}
	if job.LetterAntiMotivationID != nil {
		idStr := fmt.Sprintf("%d", *job.LetterAntiMotivationID)
		antiMotivationIDStr = &idStr
	}

	return c.JSON(dto.LetterJobStatus{
		JobID:                  job.JobID,
		Status:                 string(job.Status),
		Progress:               job.Progress,
		LetterMotivationID:     motivationIDStr,
		LetterAntiMotivationID: antiMotivationIDStr,
		Error:                  job.Error,
		EstimatedTime:          estimatedTime,
	})
}

// GetLetter récupère une lettre générée par ID
// GET /api/v1/letters/:id
func (h *LettersHandler) GetLetter(c *fiber.Ctx) error {
	letterID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid letter ID",
			"code":  "INVALID_ID",
		})
	}

	// Récupérer session ID pour vérifier ownership
	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Récupérer le visiteur
	var visitor models.Visitor
	result := h.db.Where("session_id = ?", sessionID).First(&visitor)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Visiteur non trouvé",
			"code":  "VISITOR_NOT_FOUND",
		})
	}

	// Récupérer la lettre
	var letter models.GeneratedLetter
	result = h.db.Where("id = ? AND visitor_id = ?", letterID, visitor.ID).First(&letter)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Lettre non trouvée",
				"code":  "LETTER_NOT_FOUND",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Database error",
			"code":    "DB_ERROR",
			"details": result.Error.Error(),
		})
	}

	// Construire URL de téléchargement PDF
	baseURL := c.BaseURL()
	pdfURL := fmt.Sprintf("%s/api/v1/letters/%s/pdf", baseURL, letter.ID.String())

	return c.JSON(dto.LetterDetailResponse{
		ID:           letter.ID.String(),
		CompanyName:  letter.CompanyName,
		LetterType:   string(letter.LetterType),
		Content:      letter.Content,
		CreatedAt:    letter.CreatedAt.Format("2006-01-02 15:04:05"),
		AIModel:      letter.AIModel,
		TokensUsed:   letter.TokensUsed,
		GenerationMS: letter.GenerationMS,
		Cost:         letter.EstimatedCost(),
		PDFURL:       pdfURL,
	})
}

// GetLetterPair récupère une paire de lettres par nom d'entreprise
// GET /api/v1/letters/pair?company=Google
func (h *LettersHandler) GetLetterPair(c *fiber.Ctx) error {
	companyName := c.Query("company")
	if companyName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company name required",
			"code":  "MISSING_COMPANY",
		})
	}

	sessionID := c.Cookies("maicivy_session")
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Récupérer le visiteur
	var visitor models.Visitor
	result := h.db.Where("session_id = ?", sessionID).First(&visitor)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Visiteur non trouvé",
			"code":  "VISITOR_NOT_FOUND",
		})
	}

	// Récupérer les deux lettres
	var letters []models.GeneratedLetter
	result = h.db.Where("visitor_id = ? AND company_name = ?", visitor.ID, companyName).
		Order("created_at DESC").
		Limit(2).
		Find(&letters)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DB_ERROR",
		})
	}

	if len(letters) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Aucune lettre trouvée pour cette entreprise",
			"code":  "LETTERS_NOT_FOUND",
		})
	}

	// Construire la réponse
	var motivationLetter, antiMotivationLetter *dto.LetterDetailResponse

	for _, letter := range letters {
		baseURL := c.BaseURL()
		pdfURL := fmt.Sprintf("%s/api/v1/letters/%s/pdf", baseURL, letter.ID.String())

		letterDetail := &dto.LetterDetailResponse{
			ID:           letter.ID.String(),
			CompanyName:  letter.CompanyName,
			LetterType:   string(letter.LetterType),
			Content:      letter.Content,
			CreatedAt:    letter.CreatedAt.Format("2006-01-02 15:04:05"),
			AIModel:      letter.AIModel,
			TokensUsed:   letter.TokensUsed,
			GenerationMS: letter.GenerationMS,
			Cost:         letter.EstimatedCost(),
			PDFURL:       pdfURL,
		}

		if letter.LetterType == models.LetterTypeMotivation {
			motivationLetter = letterDetail
		} else {
			antiMotivationLetter = letterDetail
		}
	}

	return c.JSON(dto.LetterPairResponse{
		MotivationLetter:     motivationLetter,
		AntiMotivationLetter: antiMotivationLetter,
		CompanyName:          companyName,
	})
}

// GetHistory récupère l'historique des lettres générées
// GET /api/v1/letters/history?page=1&per_page=10
func (h *LettersHandler) GetHistory(c *fiber.Ctx) error {
	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Récupérer le visiteur
	var visitor models.Visitor
	result := h.db.Where("session_id = ?", sessionID).First(&visitor)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Visiteur non trouvé",
			"code":  "VISITOR_NOT_FOUND",
		})
	}

	// Pagination
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

	h.db.Model(&models.GeneratedLetter{}).Where("visitor_id = ?", visitor.ID).Count(&total)

	result = h.db.Where("visitor_id = ?", visitor.ID).
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&letters)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DB_ERROR",
		})
	}

	// Mapper vers DTO
	items := make([]dto.LetterHistoryItem, len(letters))
	for i, letter := range letters {
		items[i] = dto.LetterHistoryItem{
			ID:          letter.ID.String(),
			CompanyName: letter.CompanyName,
			LetterType:  string(letter.LetterType),
			CreatedAt:   letter.CreatedAt.Format("2006-01-02 15:04:05"),
			Downloaded:  letter.Downloaded,
		}
	}

	return c.JSON(dto.LetterHistoryResponse{
		Letters: items,
		Total:   int(total),
		Page:    page,
		PerPage: perPage,
	})
}

// DownloadPDF télécharge le PDF d'une lettre
// GET /api/v1/letters/:id/pdf
func (h *LettersHandler) DownloadPDF(c *fiber.Ctx) error {
	letterID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid letter ID",
			"code":  "INVALID_ID",
		})
	}

	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Récupérer le visiteur
	var visitor models.Visitor
	result := h.db.Where("session_id = ?", sessionID).First(&visitor)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Visiteur non trouvé",
			"code":  "VISITOR_NOT_FOUND",
		})
	}

	var letter models.GeneratedLetter
	result = h.db.Where("id = ? AND visitor_id = ?", letterID, visitor.ID).First(&letter)

	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lettre non trouvée",
			"code":  "LETTER_NOT_FOUND",
		})
	}

	// Marquer comme téléchargée
	if !letter.Downloaded {
		letter.Downloaded = true
		h.db.Save(&letter)
	}

	// TODO: Générer ou servir le PDF (sera implémenté dans Doc 08)
	// Pour l'instant, retourner le texte en tant que "PDF"
	filename := fmt.Sprintf("lettre_%s_%s.txt", letter.LetterType, letter.CompanyName)

	c.Set("Content-Type", "text/plain")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.SendString(letter.Content)
}

// GetAccessStatus récupère le status d'accès IA du visiteur
// GET /api/v1/letters/access-status
func (h *LettersHandler) GetAccessStatus(c *fiber.Ctx) error {
	ctx := context.Background()
	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	// Vérifier profil détecté
	profileKey := fmt.Sprintf("visitor:%s:profile", sessionID)
	profile, _ := h.redis.Get(ctx, profileKey).Result()

	// Récupérer visiteur
	var visitor models.Visitor
	result := h.db.Where("session_id = ?", sessionID).First(&visitor)

	hasAccess := false
	accessGrantedBy := ""
	visitsRemaining := 3

	if result.Error == nil {
		hasAccess = visitor.HasAccessToAI()
		if hasAccess {
			if visitor.IsTargetProfile() {
				accessGrantedBy = "profile"
			} else {
				accessGrantedBy = "visits"
			}
		}
		visitsRemaining = 3 - visitor.VisitCount
		if visitsRemaining < 0 {
			visitsRemaining = 0
		}
	}

	message := ""
	if hasAccess {
		message = "Accès aux fonctionnalités IA accordé"
	} else {
		message = fmt.Sprintf("Encore %d visite(s) nécessaire(s) pour débloquer l'IA", visitsRemaining)
	}

	return c.JSON(dto.AccessStatusResponse{
		HasAccess:       hasAccess,
		CurrentVisits:   visitor.VisitCount,
		RequiredVisits:  3,
		VisitsRemaining: visitsRemaining,
		ProfileDetected: profile,
		AccessGrantedBy: accessGrantedBy,
		Message:         message,
	})
}

// GetRateLimitStatus récupère le status du rate limiting IA
// GET /api/v1/letters/rate-limit-status
func (h *LettersHandler) GetRateLimitStatus(c *fiber.Ctx) error {
	ctx := context.Background()
	sessionID, ok := c.Locals("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = c.Cookies("maicivy_session")
	}
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Session requise",
			"code":  "SESSION_REQUIRED",
		})
	}

	dailyKey := fmt.Sprintf("ratelimit:ai:%s:daily", sessionID)
	cooldownKey := fmt.Sprintf("ratelimit:ai:%s:cooldown", sessionID)

	// Récupérer compteur journalier
	dailyUsedStr, err := h.redis.Get(ctx, dailyKey).Result()
	dailyUsed := 0
	if err == nil {
		fmt.Sscanf(dailyUsedStr, "%d", &dailyUsed)
	}

	dailyLimit := 5
	dailyRemaining := dailyLimit - dailyUsed
	if dailyRemaining < 0 {
		dailyRemaining = 0
	}

	// Récupérer TTL reset
	ttl, _ := h.redis.TTL(ctx, dailyKey).Result()
	resetAt := time.Now().Add(ttl).Format("2006-01-02 15:04:05")

	// Vérifier cooldown
	cooldownExists, _ := h.redis.Exists(ctx, cooldownKey).Result()
	cooldownActive := cooldownExists > 0
	cooldownRemaining := 0
	if cooldownActive {
		ttl, _ := h.redis.TTL(ctx, cooldownKey).Result()
		cooldownRemaining = int(ttl.Seconds())
	}

	return c.JSON(dto.RateLimitStatusResponse{
		DailyLimit:        dailyLimit,
		DailyUsed:         dailyUsed,
		DailyRemaining:    dailyRemaining,
		ResetAt:           resetAt,
		CooldownActive:    cooldownActive,
		CooldownRemaining: cooldownRemaining,
	})
}
