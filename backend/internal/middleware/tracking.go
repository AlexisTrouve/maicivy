package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mileusna/useragent"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

const (
	SessionCookieName = "maicivy_session"
	SessionTTL        = 30 * 24 * time.Hour // 30 jours
	VisitorKeyPrefix  = "visitor:"
)

type TrackingMiddleware struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewTracking(db *gorm.DB, redisClient *redis.Client) *TrackingMiddleware {
	return &TrackingMiddleware{
		db:    db,
		redis: redisClient,
	}
}

func (tm *TrackingMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// 1. Récupérer ou créer session ID
		sessionID := c.Cookies(SessionCookieName)
		if sessionID == "" {
			sessionID = uuid.New().String()

			// Set cookie
			c.Cookie(&fiber.Cookie{
				Name:     SessionCookieName,
				Value:    sessionID,
				Expires:  time.Now().Add(SessionTTL),
				HTTPOnly: true,
				Secure:   true, // HTTPS only en production
				SameSite: "Lax",
			})
		}

		// 2. Incrémenter compteur de visites dans Redis
		visitCountKey := fmt.Sprintf("%s%s:count", VisitorKeyPrefix, sessionID)
		visitCount, err := tm.redis.Incr(ctx, visitCountKey).Result()
		if err != nil {
			log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to increment visit count")
			visitCount = 1
		}

		// Set TTL si première visite
		if visitCount == 1 {
			tm.redis.Expire(ctx, visitCountKey, SessionTTL)
		}

		// 3. Détection de profil
		profileDetected := tm.detectProfile(c)

		// Stocker profil dans Redis (cache)
		if profileDetected != "" {
			profileKey := fmt.Sprintf("%s%s:profile", VisitorKeyPrefix, sessionID)
			tm.redis.Set(ctx, profileKey, profileDetected, SessionTTL)
		}

		// 4. Stocker dans context pour utilisation dans handlers
		c.Locals("session_id", sessionID)
		c.Locals("visit_count", visitCount)
		c.Locals("profile_detected", profileDetected)

		// 5. Enregistrer/update visiteur dans PostgreSQL (async)
		go tm.saveVisitor(sessionID, c, visitCount, profileDetected)

		return c.Next()
	}
}

// detectProfile analyse User-Agent et IP pour détecter recruteurs/profils cibles
func (tm *TrackingMiddleware) detectProfile(c *fiber.Ctx) string {
	userAgentStr := c.Get("User-Agent")
	ip := c.IP()

	// Parse User-Agent
	ua := useragent.Parse(userAgentStr)

	// Patterns LinkedIn
	if strings.Contains(strings.ToLower(userAgentStr), "linkedin") {
		return "linkedin_bot"
	}

	// Patterns recruteurs (LinkedIn Sales Navigator, etc.)
	recruiterPatterns := []string{
		"sales navigator",
		"recruiter",
		"talent",
		"hiring",
	}

	userAgentLower := strings.ToLower(userAgentStr)
	for _, pattern := range recruiterPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return "recruiter"
		}
	}

	// Détection entreprise via IP (optionnel - nécessite API Clearbit ou GeoIP)
	// company := tm.lookupCompany(ip)
	// if company != "" {
	//     return "corporate:" + company
	// }

	// Desktop professionnel (pas mobile)
	if ua.Desktop && !ua.Mobile {
		return "professional"
	}

	return "" // Pas de profil spécifique détecté
}

// saveVisitor enregistre ou met à jour le visiteur dans PostgreSQL
func (tm *TrackingMiddleware) saveVisitor(sessionID string, c *fiber.Ctx, visitCount int64, profile string) {
	// Hash IP pour privacy
	ipHash := hashIP(c.IP())

	visitor := models.Visitor{
		SessionID:       sessionID,
		IPHash:          ipHash,
		UserAgent:       c.Get("User-Agent"),
		VisitCount:      int(visitCount),
		ProfileDetected: models.ProfileType(profile),
		LastVisit:       time.Now(),
	}

	// Upsert (insert ou update si existe)
	result := tm.db.Where("session_id = ?", sessionID).
		Assign(visitor).
		FirstOrCreate(&visitor)

	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("session_id", sessionID).
			Msg("Failed to save visitor to database")
	}
}

// hashIP hash l'IP pour respecter RGPD/privacy
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}
