package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// VisitorHandler gère les endpoints liés aux visiteurs
type VisitorHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewVisitorHandler crée une nouvelle instance
func NewVisitorHandler(db *gorm.DB, redisClient *redis.Client) *VisitorHandler {
	return &VisitorHandler{
		db:    db,
		redis: redisClient,
	}
}

// VisitorStatusResponse représente le statut du visiteur actuel
type VisitorStatusResponse struct {
	SessionID       string `json:"session_id"`
	VisitCount      int    `json:"visit_count"`
	ProfileDetected string `json:"profile_detected"`
	HasAccessToAI   bool   `json:"has_access_to_ai"`
	IsTargetProfile bool   `json:"is_target_profile"`
}

// GetVisitorStatus retourne le statut du visiteur actuel
// @Summary Get visitor status
// @Description Returns current visitor session information
// @Tags visitor
// @Accept json
// @Produce json
// @Success 200 {object} VisitorStatusResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/visitor/status [get]
func (vh *VisitorHandler) GetVisitorStatus(c *fiber.Ctx) error {
	// Récupérer session_id depuis context (set by tracking middleware)
	sessionID := c.Locals("session_id")
	if sessionID == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "No session found",
		})
	}

	sessionIDStr := sessionID.(string)

	// Récupérer visitor depuis DB
	var visitor models.Visitor
	result := vh.db.Where("session_id = ?", sessionIDStr).First(&visitor)
	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Visitor not found",
		})
	}

	// Construire réponse
	response := VisitorStatusResponse{
		SessionID:       visitor.SessionID,
		VisitCount:      visitor.VisitCount,
		ProfileDetected: string(visitor.ProfileDetected),
		HasAccessToAI:   visitor.HasAccessToAI(),
		IsTargetProfile: visitor.IsTargetProfile(),
	}

	return c.JSON(response)
}

// VisitorCheckResponse représente le statut pour le frontend
type VisitorCheckResponse struct {
	SessionID       string `json:"sessionId"`
	VisitCount      int    `json:"visitCount"`
	HasAccess       bool   `json:"hasAccess"`
	ProfileDetected string `json:"profileDetected,omitempty"`
	RemainingVisits int    `json:"remainingVisits"`
}

// CheckVisitorStatus retourne le statut du visiteur pour le frontend
// @Summary Check visitor status
// @Description Returns current visitor status with access gate info
// @Tags visitor
// @Accept json
// @Produce json
// @Success 200 {object} VisitorCheckResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/visitors/check [get]
func (vh *VisitorHandler) CheckVisitorStatus(c *fiber.Ctx) error {
	// Récupérer session_id depuis context (set by tracking middleware)
	sessionID := c.Locals("session_id")
	if sessionID == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "No session found",
		})
	}

	sessionIDStr := sessionID.(string)

	// Récupérer visitor depuis DB
	var visitor models.Visitor
	result := vh.db.Where("session_id = ?", sessionIDStr).First(&visitor)
	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Visitor not found",
		})
	}

	// Calculer remaining visits (access gate: 3 visits before teaser)
	const maxFreeVisits = 3
	remainingVisits := maxFreeVisits - visitor.VisitCount
	if remainingVisits < 0 {
		remainingVisits = 0
	}

	// Construire réponse
	response := VisitorCheckResponse{
		SessionID:       visitor.SessionID,
		VisitCount:      visitor.VisitCount,
		HasAccess:       visitor.HasAccessToAI(),
		ProfileDetected: string(visitor.ProfileDetected),
		RemainingVisits: remainingVisits,
	}

	return c.JSON(response)
}
