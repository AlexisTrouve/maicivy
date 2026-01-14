// backend/internal/middleware/profile_enrichment.go
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// ProfileEnrichmentMiddleware enrichit automatiquement les visiteurs avec détection de profil
type ProfileEnrichmentMiddleware struct {
	db                *gorm.DB
	redis             *goredis.Client
	profileDetector   *services.ProfileDetectorService
}

// NewProfileEnrichmentMiddleware crée une nouvelle instance du middleware
func NewProfileEnrichmentMiddleware(
	db *gorm.DB,
	redis *goredis.Client,
	profileDetector *services.ProfileDetectorService,
) *ProfileEnrichmentMiddleware {
	return &ProfileEnrichmentMiddleware{
		db:              db,
		redis:           redis,
		profileDetector: profileDetector,
	}
}

// Handle traite la détection et l'enrichissement du profil
func (m *ProfileEnrichmentMiddleware) Handle() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer les informations de la requête
		ip := c.IP()
		userAgent := c.Get("User-Agent")
		referer := c.Get("Referer")
		sessionID := c.Cookies("maicivy_session", "")

		// Si pas de session ID, passer au next (sera géré par visitor_tracking middleware)
		if sessionID == "" {
			return c.Next()
		}

		// Vérifier si le profil a déjà été détecté pour cette session (cache)
		cacheKey := fmt.Sprintf("profile:session:%s", sessionID)
		cachedProfile, err := m.redis.Get(c.Context(), cacheKey).Result()

		var detectedProfile *services.DetectedProfile

		if err == goredis.Nil {
			// Profil pas en cache, détecter maintenant
			detectedProfile, err = m.profileDetector.DetectProfile(c.Context(), ip, userAgent, referer)
			if err != nil {
				// Log l'erreur mais ne pas bloquer la requête (graceful degradation)
				fmt.Printf("Profile detection error: %v\n", err)
				return c.Next()
			}

			// Mettre en cache (TTL 24h)
			profileJSON, _ := json.Marshal(detectedProfile)
			m.redis.Set(c.Context(), cacheKey, string(profileJSON), 24*time.Hour)

			// Enregistrer en base de données de manière asynchrone
			go m.storeProfileInDB(context.Background(), sessionID, ip, detectedProfile)

		} else if err == nil {
			// Profil en cache, le parser
			detectedProfile = &services.DetectedProfile{}
			if err := json.Unmarshal([]byte(cachedProfile), detectedProfile); err != nil {
				// Cache corrompu, ignorer
				return c.Next()
			}
		} else {
			// Erreur Redis, continuer sans bloquer
			return c.Next()
		}

		// Stocker le profil détecté dans le context pour les handlers suivants
		c.Locals("detected_profile", detectedProfile)

		// Vérifier si le profil devrait bypass l'access gate
		if m.profileDetector.ShouldBypassAccessGate(detectedProfile) {
			// Stocker le bypass en Redis
			if err := m.profileDetector.StoreBypassInRedis(c.Context(), sessionID); err != nil {
				fmt.Printf("Failed to store bypass: %v\n", err)
			}

			// Ajouter un header pour debug
			c.Set("X-Access-Gate-Bypass", "true")
			c.Set("X-Profile-Type", string(detectedProfile.ProfileType))
		}

		// Ajouter des headers de debug
		c.Set("X-Profile-Detected", string(detectedProfile.ProfileType))
		c.Set("X-Confidence-Score", fmt.Sprintf("%d", detectedProfile.Confidence))

		return c.Next()
	}
}

// storeProfileInDB enregistre le profil détecté en base de données
func (m *ProfileEnrichmentMiddleware) storeProfileInDB(ctx context.Context, sessionID, ip string, profile *services.DetectedProfile) {
	// Récupérer le visiteur existant via session_id
	var visitor models.Visitor
	result := m.db.Where("session_id = ?", sessionID).First(&visitor)

	if result.Error == gorm.ErrRecordNotFound {
		// Visiteur n'existe pas encore, sera créé par visitor_tracking middleware
		// On ne fait rien ici
		return
	}

	// Mettre à jour le visiteur avec les infos de profil
	enrichmentDataJSON, _ := json.Marshal(profile.EnrichmentData)

	updates := map[string]interface{}{
		"profile_type":         string(profile.ProfileType),
		"enrichment_data":      enrichmentDataJSON,
		"detection_confidence": profile.Confidence,
	}

	m.db.Model(&visitor).Updates(updates)
}

// GetDetectedProfile récupère le profil détecté depuis le context
func GetDetectedProfile(c *fiber.Ctx) *services.DetectedProfile {
	profile, ok := c.Locals("detected_profile").(*services.DetectedProfile)
	if !ok {
		return nil
	}
	return profile
}

// CheckAccessGateBypass vérifie si la session a un bypass actif
func CheckAccessGateBypass(c *fiber.Ctx, redis *goredis.Client) (bool, error) {
	sessionID := c.Cookies("maicivy_session", "")
	if sessionID == "" {
		return false, nil
	}

	key := fmt.Sprintf("access:bypass:%s", sessionID)
	val, err := redis.Get(c.Context(), key).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return val == "1", nil
}
