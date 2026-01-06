// backend/internal/api/profile.go
package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// ProfileHandler gère les endpoints liés à la détection de profil
type ProfileHandler struct {
	db              *gorm.DB
	redis           *redis.Client
	profileDetector *services.ProfileDetectorService
}

// NewProfileHandler crée une nouvelle instance du handler
func NewProfileHandler(
	db *gorm.DB,
	redis *redis.Client,
	profileDetector *services.ProfileDetectorService,
) *ProfileHandler {
	return &ProfileHandler{
		db:              db,
		redis:           redis,
		profileDetector: profileDetector,
	}
}

// DetectProfileResponse structure de la réponse de détection
type DetectProfileResponse struct {
	Success        bool                       `json:"success"`
	ProfileType    string                     `json:"profile_type"`
	Confidence     int                        `json:"confidence"`
	EnrichmentData map[string]interface{}     `json:"enrichment_data,omitempty"`
	DeviceInfo     services.DeviceInfo        `json:"device_info"`
	Sources        []string                   `json:"detection_sources"`
	BypassEnabled  bool                       `json:"bypass_enabled"`
}

// GetDetect détecte manuellement le profil du visiteur (endpoint de debug)
// GET /api/v1/profile/detect
func (h *ProfileHandler) GetDetect(c *fiber.Ctx) error {
	// Récupérer les informations de la requête
	ip := c.IP()
	userAgent := c.Get("User-Agent")
	referer := c.Get("Referer")

	// Détecter le profil
	profile, err := h.profileDetector.DetectProfile(c.Context(), ip, userAgent, referer)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "profile_detection_failed",
			"message": err.Error(),
		})
	}

	// Vérifier si bypass est activé
	bypassEnabled := h.profileDetector.ShouldBypassAccessGate(profile)

	// Ajouter des headers de debug
	c.Set("X-Profile-Type", string(profile.ProfileType))
	c.Set("X-Confidence", fmt.Sprintf("%d", profile.Confidence))
	c.Set("X-Bypass-Enabled", fmt.Sprintf("%t", bypassEnabled))

	return c.JSON(DetectProfileResponse{
		Success:        true,
		ProfileType:    string(profile.ProfileType),
		Confidence:     profile.Confidence,
		EnrichmentData: profile.EnrichmentData,
		DeviceInfo:     profile.DeviceInfo,
		Sources:        profile.DetectionSources,
		BypassEnabled:  bypassEnabled,
	})
}

// GetCurrentProfile récupère le profil détecté depuis le context (middleware)
// GET /api/v1/profile/current
func (h *ProfileHandler) GetCurrentProfile(c *fiber.Ctx) error {
	// Récupérer le profil depuis le context (mis par le middleware)
	profile := middleware.GetDetectedProfile(c)

	if profile == nil {
		return c.JSON(fiber.Map{
			"success":      true,
			"profile_type": "other",
			"confidence":   0,
			"message":      "no profile detected",
		})
	}

	// Vérifier si bypass est activé
	bypassEnabled, _ := middleware.CheckAccessGateBypass(c, h.redis)

	return c.JSON(DetectProfileResponse{
		Success:        true,
		ProfileType:    string(profile.ProfileType),
		Confidence:     profile.Confidence,
		EnrichmentData: profile.EnrichmentData,
		DeviceInfo:     profile.DeviceInfo,
		Sources:        profile.DetectionSources,
		BypassEnabled:  bypassEnabled,
	})
}

// GetBypassStatus vérifie si la session a un bypass actif
// GET /api/v1/profile/bypass
func (h *ProfileHandler) GetBypassStatus(c *fiber.Ctx) error {
	sessionID := c.Cookies("maicivy_session", "")
	if sessionID == "" {
		return c.JSON(fiber.Map{
			"success": true,
			"bypass":  false,
			"message": "no session id",
		})
	}

	// Vérifier le bypass
	bypassed, err := h.profileDetector.CheckBypassExists(c.Context(), sessionID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "failed_to_check_bypass",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"bypass":  bypassed,
	})
}

// PostEnableBypass active manuellement le bypass pour une session (admin only)
// POST /api/v1/profile/bypass/enable
func (h *ProfileHandler) PostEnableBypass(c *fiber.Ctx) error {
	sessionID := c.Cookies("maicivy_session", "")
	if sessionID == "" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "no_session_id",
		})
	}

	// Activer le bypass
	if err := h.profileDetector.StoreBypassInRedis(c.Context(), sessionID); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "failed_to_enable_bypass",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "bypass enabled",
	})
}

// GetStats retourne des statistiques sur les profils détectés
// GET /api/v1/profile/stats
func (h *ProfileHandler) GetStats(c *fiber.Ctx) error {
	// Requête SQL pour obtenir les stats par type de profil
	type ProfileStats struct {
		ProfileType string `json:"profile_type"`
		Count       int    `json:"count"`
		AvgConfidence float64 `json:"avg_confidence"`
	}

	var stats []ProfileStats

	h.db.Raw(`
		SELECT
			COALESCE(profile_type, 'other') as profile_type,
			COUNT(*) as count,
			AVG(detection_confidence) as avg_confidence
		FROM visitors
		WHERE profile_type IS NOT NULL
		GROUP BY profile_type
		ORDER BY count DESC
	`).Scan(&stats)

	// Total de visiteurs avec profil détecté
	var totalDetected int64
	h.db.Model(&struct{}{}).
		Table("visitors").
		Where("profile_type IS NOT NULL AND profile_type != 'other'").
		Count(&totalDetected)

	// Total de visiteurs
	var totalVisitors int64
	h.db.Model(&struct{}{}).
		Table("visitors").
		Count(&totalVisitors)

	return c.JSON(fiber.Map{
		"success":         true,
		"stats_by_type":   stats,
		"total_detected":  totalDetected,
		"total_visitors":  totalVisitors,
		"detection_rate":  float64(totalDetected) / float64(totalVisitors) * 100,
	})
}
