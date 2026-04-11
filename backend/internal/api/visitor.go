package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// VisitorHandler gère les endpoints liés aux visiteurs
type VisitorHandler struct {
	db               *gorm.DB
	redis            *redis.Client
	analyticsService *services.AnalyticsService
}

// NewVisitorHandler crée une nouvelle instance
func NewVisitorHandler(db *gorm.DB, redisClient *redis.Client, analyticsService *services.AnalyticsService) *VisitorHandler {
	return &VisitorHandler{
		db:               db,
		redis:            redisClient,
		analyticsService: analyticsService,
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

// HeartbeatRequest représente une requête de heartbeat
type HeartbeatRequest struct {
	PageURL   string                 `json:"page_url,omitempty"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
}

// HeartbeatResponse représente la réponse d'un heartbeat
type HeartbeatResponse struct {
	Success       bool  `json:"success"`
	Timestamp     int64 `json:"timestamp"`
	ActiveVisitors int  `json:"active_visitors,omitempty"`
}

// Heartbeat endpoint pour marquer un visiteur comme actif
// @Summary Send visitor heartbeat
// @Description Marks visitor as active and returns current active visitor count
// @Tags visitor
// @Accept json
// @Produce json
// @Param body body HeartbeatRequest false "Heartbeat data"
// @Success 200 {object} HeartbeatResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/visitors/heartbeat [post]
func (vh *VisitorHandler) Heartbeat(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer visitor_id depuis context (set by tracking middleware)
	visitorID := c.Locals("visitor_id")
	if visitorID == nil {
		log.Warn().Msg("Heartbeat called without visitor_id in context")
		return c.Status(404).JSON(fiber.Map{
			"error": "No visitor session found",
		})
	}

	visitorUUID, ok := visitorID.(uuid.UUID)
	if !ok || visitorUUID == uuid.Nil {
		log.Warn().Interface("visitor_id", visitorID).Msg("Invalid visitor_id in heartbeat")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid visitor session",
		})
	}

	// Parser le body (optionnel)
	var req HeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		// Body parsing est optionnel, continuer même si erreur
		log.Debug().Err(err).Msg("Failed to parse heartbeat body, continuing anyway")
	}

	// Marquer le visiteur comme actif dans Redis
	if err := vh.analyticsService.MarkVisitorActive(ctx, visitorUUID); err != nil {
		log.Error().Err(err).Str("visitor_id", visitorUUID.String()).Msg("Failed to mark visitor as active")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update visitor status",
		})
	}

	// Optionnel: tracker un événement analytics si page_url fourni
	if req.PageURL != "" {
		event := &models.AnalyticsEvent{
			VisitorID: visitorUUID,
			EventType: models.EventTypePageView,
			PageURL:   req.PageURL,
			EventData: "{}",
		}

		// Ajouter event_data si présent
		if len(req.EventData) > 0 {
			eventDataJSON, err := json.Marshal(req.EventData)
			if err == nil {
				event.EventData = string(eventDataJSON)
			}
		}

		// Tracker l'événement (non bloquant)
		go func() {
			bgCtx, bgCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer bgCancel()
			if err := vh.analyticsService.TrackEvent(bgCtx, event); err != nil {
				log.Warn().Err(err).Msg("Failed to track heartbeat event")
			}
		}()
	}

	// Récupérer le nombre de visiteurs actifs (optionnel, peut être coûteux)
	stats, err := vh.analyticsService.GetRealtimeStats(ctx)
	activeVisitors := 0
	if err == nil {
		if cv, ok := stats["current_visitors"].(int64); ok {
			activeVisitors = int(cv)
		}
	}

	response := HeartbeatResponse{
		Success:        true,
		Timestamp:      time.Now().Unix(),
		ActiveVisitors: activeVisitors,
	}

	log.Debug().
		Str("visitor_id", visitorUUID.String()).
		Int("active_visitors", activeVisitors).
		Msg("Heartbeat received")

	return c.JSON(response)
}
