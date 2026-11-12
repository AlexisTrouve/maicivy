package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

// MessagesHandler handler pour la génération de messages plateforme (sync)
type MessagesHandler struct {
	db              *gorm.DB
	redis           *redis.Client
	letterGenerator *services.LetterGenerator // pour accéder au PromptBuilder + AIService
	ownerAPIKey     string
	// Modèles des deux tiers (cf. config.OwnerModel / config.ClaudeModel) — plus de littéral en dur.
	ownerModel   string
	visitorModel string
}

func NewMessagesHandler(db *gorm.DB, redis *redis.Client, letterGenerator *services.LetterGenerator, ownerAPIKey, ownerModel, visitorModel string) *MessagesHandler {
	return &MessagesHandler{
		db:              db,
		redis:           redis,
		letterGenerator: letterGenerator,
		ownerAPIKey:     ownerAPIKey,
		ownerModel:      ownerModel,
		visitorModel:    visitorModel,
	}
}

// GenerateMessage génère un message de prospection (sync, pas de queue)
// POST /api/v1/messages/generate
func (h *MessagesHandler) GenerateMessage(c *fiber.Ctx) error {
	var req models.PlatformMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	if len(req.Mission) < 20 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Mission description too short (min 20 characters)",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Normaliser
	if req.Lang == "" {
		req.Lang = "fr"
	}
	if req.Platform == "" {
		req.Platform = "malt"
	}

	// Sélection modèle : owner → Opus, visiteur → Haiku
	model := h.visitorModel
	if isOwner, _ := c.Locals("is_owner").(bool); isOwner {
		model = h.ownerModel
	}

	// Incrémenter rate limit
	cooldownDuration := 2 * time.Minute
	if err := middleware.IncrementAIRateLimit(c, h.redis, cooldownDuration); err != nil {
		fmt.Printf("Failed to increment rate limit: %v\n", err)
	}

	// Générer via LetterGenerator (accès au PromptBuilder + AIService)
	content, metrics, err := h.letterGenerator.GeneratePlatformMessage(c.Context(), req, model)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Generation failed",
			"code":    "GENERATION_ERROR",
			"details": err.Error(),
		})
	}

	return c.JSON(models.PlatformMessageResponse{
		Content:       content,
		Platform:      req.Platform,
		TokensUsed:    metrics.TotalTokens,
		EstimatedCost: metrics.EstimatedCost,
		Model:         metrics.Model,
	})
}
