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
	db     *gorm.DB
	redis  *redis.Client
	secret string // secret HMAC pour signer/vérifier le cookie de session
}

// NewTracking construit le middleware de tracking. `secret` signe le cookie de session ; s'il est
// vide (dev / non configuré) on retombe sur un défaut NON sécurisé en loggant un avertissement — la
// signature reste active (l'anti-amplification marche), mais le secret est à définir en prod.
func NewTracking(db *gorm.DB, redisClient *redis.Client, secret string) *TrackingMiddleware {
	if secret == "" {
		log.Warn().Msg("SESSION_SECRET non défini — fallback dev non sécurisé ; à définir en prod")
		secret = "maicivy-dev-insecure-session-secret"
	}
	return &TrackingMiddleware{
		db:     db,
		redis:  redisClient,
		secret: secret,
	}
}

func (tm *TrackingMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()

		// Skip tracking pour les endpoints de monitoring/health
		path := c.Path()
		if path == "/health" || path == "/health/deep" || path == "/metrics" {
			return c.Next()
		}

		// 1. Le cookie de session doit être un token QUE NOUS avons émis (signé HMAC). Un cookie
		// absent ou forgé = visiteur non identifié : on lui émet une session signée neuve et on NE
		// persiste RIEN (ni Redis, ni PostgreSQL). POURQUOI : sinon n'importe quel bot envoyant des
		// cookies bidons déclenche un INSERT `visitors` par cookie (amplification d'écriture non
		// authentifiée sur le hot path). La persistance n'arrive que quand un cookie signé NOUS
		// revient (2e requête d'un vrai navigateur) → un bot sans cookie ne fait jamais écrire la base.
		sessionID := c.Cookies(SessionCookieName)
		if !verifySession(sessionID, tm.secret) {
			fresh := newSignedSession(tm.secret)
			c.Cookie(&fiber.Cookie{
				Name:     SessionCookieName,
				Value:    fresh,
				Expires:  time.Now().Add(SessionTTL),
				HTTPOnly: true,
				Secure:   true, // HTTPS only en production
				SameSite: "Lax",
			})
			// Contact anonyme : on expose la session aux handlers mais SANS visitor_id (l'analytics
			// skippe proprement son absence) et sans toucher Redis/PostgreSQL.
			c.Locals("session_id", fresh)
			c.Locals("visit_count", int64(0))
			c.Locals("profile_detected", "")
			return c.Next()
		}

		// --- Cookie valide → visiteur réel qui revient : comptage + profil + persistance. ---

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
		if profileDetected != "" {
			profileKey := fmt.Sprintf("%s%s:profile", VisitorKeyPrefix, sessionID)
			tm.redis.Set(ctx, profileKey, profileDetected, SessionTTL)
		}

		// 4. Extraire les valeurs du contexte
		userAgent := c.Get("User-Agent")
		ip := c.IP()

		// 5. Enregistrer/update visiteur dans PostgreSQL (SYNCHRONE pour avoir visitor_id)
		visitorID := tm.saveVisitorSync(sessionID, ip, userAgent, visitCount, profileDetected)

		// 6. Stocker dans context pour les handlers + analytics middleware
		c.Locals("session_id", sessionID)
		c.Locals("visit_count", visitCount)
		c.Locals("profile_detected", profileDetected)
		c.Locals("visitor_id", visitorID) // UUID pour analytics

		return c.Next()
	}
}

// detectProfile analyse User-Agent et IP pour détecter recruteurs/profils cibles
func (tm *TrackingMiddleware) detectProfile(c *fiber.Ctx) string {
	userAgentStr := c.Get("User-Agent")
	_ = c.IP() // IP could be used for geo-targeting in the future

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

// saveVisitorSync enregistre ou met à jour le visiteur dans PostgreSQL et retourne l'ID
func (tm *TrackingMiddleware) saveVisitorSync(sessionID string, ip string, userAgent string, visitCount int64, profile string) uuid.UUID {
	// Hash IP pour privacy
	ipHash := hashIP(ip)
	now := time.Now()

	// Check if visitor exists
	var existingVisitor models.Visitor
	result := tm.db.Where("session_id = ?", sessionID).First(&existingVisitor)

	if result.Error == gorm.ErrRecordNotFound {
		// New visitor - create with FirstVisit set
		newVisitor := models.Visitor{
			SessionID:       sessionID,
			IPHash:          ipHash,
			UserAgent:       userAgent,
			VisitCount:      int(visitCount),
			ProfileDetected: models.ProfileType(profile),
			FirstVisit:      now,
			LastVisit:       now,
		}

		if err := tm.db.Create(&newVisitor).Error; err != nil {
			log.Error().
				Err(err).
				Str("session_id", sessionID).
				Msg("Failed to create visitor in database")
			return uuid.Nil
		}
		return newVisitor.ID
	} else if result.Error == nil {
		// Existing visitor - update visit count and last visit
		updates := map[string]interface{}{
			"visit_count":      int(visitCount),
			"last_visit":       now,
			"profile_detected": models.ProfileType(profile),
			"user_agent":       userAgent,
			"ip_hash":          ipHash,
		}

		if err := tm.db.Model(&existingVisitor).Updates(updates).Error; err != nil {
			log.Error().
				Err(err).
				Str("session_id", sessionID).
				Msg("Failed to update visitor in database")
		}
		return existingVisitor.ID
	} else {
		// Database error
		log.Error().
			Err(result.Error).
			Str("session_id", sessionID).
			Msg("Failed to query visitor from database")
		return uuid.Nil
	}
}

// hashIP hash l'IP pour respecter RGPD/privacy
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}
