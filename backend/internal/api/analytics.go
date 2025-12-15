package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// AnalyticsHandler gère les endpoints analytics
type AnalyticsHandler struct {
	service *services.AnalyticsService
}

// NewAnalyticsHandler crée un nouveau handler analytics
func NewAnalyticsHandler(service *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
	}
}

// RegisterRoutes enregistre les routes analytics
func (h *AnalyticsHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	analytics := api.Group("/analytics")

	// Routes publiques (pas d'auth requise)
	analytics.Get("/realtime", h.GetRealtimeStats)
	analytics.Get("/stats", h.GetStats)
	analytics.Get("/themes", h.GetTopThemes)
	analytics.Get("/letters", h.GetLettersStats)
	analytics.Get("/timeline", h.GetTimeline)
	analytics.Get("/heatmap", h.GetHeatmap)

	// Route pour tracker des événements custom
	analytics.Post("/event", h.TrackEvent)
}

// GetRealtimeStats récupère les métriques temps réel
// GET /api/v1/analytics/realtime
func (h *AnalyticsHandler) GetRealtimeStats(c *fiber.Ctx) error {
	stats, err := h.service.GetRealtimeStats(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to get realtime stats")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve realtime stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetStats récupère statistiques agrégées
// GET /api/v1/analytics/stats?period=day|week|month
func (h *AnalyticsHandler) GetStats(c *fiber.Ctx) error {
	period := c.Query("period", "day")

	// Validation du paramètre period
	if period != "day" && period != "week" && period != "month" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid period. Must be: day, week, or month",
		})
	}

	stats, err := h.service.GetStats(c.Context(), period)
	if err != nil {
		log.Error().Err(err).Str("period", period).Msg("Failed to get stats")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetTopThemes récupère les thèmes CV les plus consultés
// GET /api/v1/analytics/themes?limit=5
func (h *AnalyticsHandler) GetTopThemes(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 5)
	if limit < 1 || limit > 20 {
		limit = 5
	}

	themes, err := h.service.GetTopThemes(c.Context(), int64(limit))
	if err != nil {
		log.Error().Err(err).Msg("Failed to get top themes")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    themes,
	})
}

// GetLettersStats récupère stats lettres générées
// GET /api/v1/analytics/letters?period=day|week|month
func (h *AnalyticsHandler) GetLettersStats(c *fiber.Ctx) error {
	period := c.Query("period", "day")

	// Validation du paramètre period
	if period != "day" && period != "week" && period != "month" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid period. Must be: day, week, or month",
		})
	}

	stats, err := h.service.GetLettersStats(c.Context(), period)
	if err != nil {
		log.Error().Err(err).Str("period", period).Msg("Failed to get letters stats")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve letters stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetTimeline récupère les événements récents
// GET /api/v1/analytics/timeline?limit=50&offset=0
func (h *AnalyticsHandler) GetTimeline(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	// Validation
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	events, err := h.service.GetTimeline(c.Context(), limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get timeline")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve timeline",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    events,
		"meta": fiber.Map{
			"limit":  limit,
			"offset": offset,
			"count":  len(events),
		},
	})
}

// GetHeatmap récupère les données de heatmap
// GET /api/v1/analytics/heatmap?page_url=/cv&hours=24
func (h *AnalyticsHandler) GetHeatmap(c *fiber.Ctx) error {
	pageURL := c.Query("page_url", "")
	hours := c.QueryInt("hours", 24)

	// Validation
	if hours < 1 || hours > 168 { // Max 7 jours
		hours = 24
	}

	heatmap, err := h.service.GetHeatmapData(c.Context(), pageURL, hours)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get heatmap data")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to retrieve heatmap data",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    heatmap,
		"meta": fiber.Map{
			"page_url": pageURL,
			"hours":    hours,
			"count":    len(heatmap),
		},
	})
}

// TrackEvent enregistre un événement custom
// POST /api/v1/analytics/event
// Body: { "event_type": "button_click", "event_data": { "button": "cta", "x": 450, "y": 200 } }
func (h *AnalyticsHandler) TrackEvent(c *fiber.Ctx) error {
	// Récupérer visitor_id depuis le contexte (ajouté par middleware tracking)
	visitorID, ok := c.Locals("visitor_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Missing visitor session",
		})
	}

	// Parse request body
	var req struct {
		EventType string                 `json:"event_type"`
		EventData map[string]interface{} `json:"event_data"`
		PageURL   string                 `json:"page_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validation
	if req.EventType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "event_type is required",
		})
	}

	// Convertir event_data en JSON string
	eventDataJSON := "{}"
	if req.EventData != nil {
		if data, err := json.Marshal(req.EventData); err == nil {
			eventDataJSON = string(data)
		}
	}

	// Créer l'événement
	event := &models.AnalyticsEvent{
		VisitorID: visitorID,
		EventType: models.EventType(req.EventType),
		EventData: eventDataJSON,
		PageURL:   req.PageURL,
		Referrer:  c.Get("Referer"),
	}

	// Track event
	if err := h.service.TrackEvent(c.Context(), event); err != nil {
		log.Error().Err(err).Msg("Failed to track event")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to track event",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Event tracked successfully",
	})
}
