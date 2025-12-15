// backend/internal/services/profile_detector.go
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ProfileType représente les types de profils détectables
type ProfileType string

const (
	ProfileTypeRecruiter ProfileType = "recruiter"
	ProfileTypeCTO       ProfileType = "cto"
	ProfileTypeTechLead  ProfileType = "tech_lead"
	ProfileTypeCEO       ProfileType = "ceo"
	ProfileTypeDeveloper ProfileType = "developer"
	ProfileTypeOther     ProfileType = "other"
)

// DetectedProfile contient les informations de détection d'un visiteur
type DetectedProfile struct {
	ProfileType      ProfileType             `json:"profile_type"`
	Confidence       int                     `json:"confidence"` // 0-100%
	EnrichmentData   map[string]interface{}  `json:"enrichment_data,omitempty"`
	DetectionSources []string                `json:"detection_sources"`
	DeviceInfo       DeviceInfo              `json:"device_info"`
}

// DeviceInfo contient les informations du device du visiteur
type DeviceInfo struct {
	Browser    string `json:"browser"`
	OS         string `json:"os"`
	DeviceType string `json:"device_type"` // mobile, tablet, desktop
	IsBot      bool   `json:"is_bot"`
}

// ClearbitClientInterface defines the interface for Clearbit enrichment
type ClearbitClientInterface interface {
	EnrichByIP(ctx context.Context, hashedIP, realIP string) (map[string]interface{}, error)
}

// UserAgentParserInterface defines the interface for User-Agent parsing
type UserAgentParserInterface interface {
	Parse(userAgent string) (DeviceInfo, bool)
}

// ProfileDetectorService gère la détection de profils visiteurs
type ProfileDetectorService struct {
	db             *gorm.DB
	redis          *redis.Client
	clearbitClient ClearbitClientInterface
	uaParser       UserAgentParserInterface
}

// NewProfileDetectorService crée une nouvelle instance du service
func NewProfileDetectorService(db *gorm.DB, redis *redis.Client, clearbitClient ClearbitClientInterface, uaParser UserAgentParserInterface) *ProfileDetectorService {
	return &ProfileDetectorService{
		db:             db,
		redis:          redis,
		clearbitClient: clearbitClient,
		uaParser:       uaParser,
	}
}

// DetectProfile analyse les informations du visiteur et retourne un profil détecté
func (s *ProfileDetectorService) DetectProfile(ctx context.Context, ip, userAgent, referer string) (*DetectedProfile, error) {
	// Initialiser le profil avec des valeurs par défaut
	profile := &DetectedProfile{
		ProfileType:      ProfileTypeOther,
		Confidence:       0,
		EnrichmentData:   make(map[string]interface{}),
		DetectionSources: []string{},
	}

	// 1. Parse User-Agent
	deviceInfo, isBot := s.uaParser.Parse(userAgent)
	profile.DeviceInfo = deviceInfo

	// Vérifier si c'est un bot recruiter connu
	if isBot && isRecruiterBot(userAgent) {
		profile.ProfileType = ProfileTypeRecruiter
		profile.Confidence = 80
		profile.DetectionSources = append(profile.DetectionSources, "bot_recruiter_user_agent")
		return profile, nil
	}

	// 2. Détecter via User-Agent patterns
	userAgentScore, userAgentType := s.detectFromUserAgent(userAgent)
	if userAgentScore > 0 {
		profile.Confidence += userAgentScore
		profile.ProfileType = userAgentType
		profile.DetectionSources = append(profile.DetectionSources, "user_agent_pattern")
	}

	// 3. Analyser le referer
	refererScore, refererType := s.detectFromReferer(referer)
	if refererScore > 0 {
		// Ajuster le profil type si le referer a plus de poids
		if refererScore > userAgentScore && refererType != ProfileTypeOther {
			profile.ProfileType = refererType
		}
		profile.Confidence += refererScore
		profile.DetectionSources = append(profile.DetectionSources, "referer_analysis")
	}

	// 4. Enrichissement via Clearbit (score le plus élevé)
	hashedIP := hashIP(ip)
	enrichmentData, err := s.clearbitClient.EnrichByIP(ctx, hashedIP, ip)
	if err == nil && enrichmentData != nil {
		ipScore, ipType := s.analyzeEnrichmentData(enrichmentData)

		// L'enrichissement IP a le poids le plus fort (50%)
		if ipScore > 0 {
			profile.ProfileType = ipType
			profile.Confidence = calculateFinalConfidence(userAgentScore, refererScore, ipScore)
			profile.EnrichmentData = enrichmentData
			profile.DetectionSources = append(profile.DetectionSources, "clearbit_enrichment")
		}
	}

	// 5. Normaliser le confidence score (max 100)
	if profile.Confidence > 100 {
		profile.Confidence = 100
	}

	return profile, nil
}

// detectFromUserAgent analyse le User-Agent pour détecter des patterns
func (s *ProfileDetectorService) detectFromUserAgent(userAgent string) (int, ProfileType) {
	userAgentLower := strings.ToLower(userAgent)

	// Patterns de recruteurs
	recruiterPatterns := []string{
		"linkedinapp",
		"linkedin",
		"greenhouse",
		"lever",
		"workday",
		"applicantstack",
		"jobvite",
		"recruiting",
	}
	for _, pattern := range recruiterPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return 30, ProfileTypeRecruiter
		}
	}

	// Patterns de développeurs
	developerPatterns := []string{
		"postman",
		"curl",
		"wget",
		"httpie",
		"insomnia",
	}
	for _, pattern := range developerPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return 20, ProfileTypeDeveloper
		}
	}

	return 0, ProfileTypeOther
}

// detectFromReferer analyse le referer pour détecter la source
func (s *ProfileDetectorService) detectFromReferer(referer string) (int, ProfileType) {
	if referer == "" {
		return 0, ProfileTypeOther
	}

	refererLower := strings.ToLower(referer)

	// Détection LinkedIn
	if strings.Contains(refererLower, "linkedin.com/jobs") ||
	   strings.Contains(refererLower, "linkedin.com/recruiter") {
		return 20, ProfileTypeRecruiter
	}

	if strings.Contains(refererLower, "linkedin.com") {
		return 10, ProfileTypeRecruiter
	}

	// Détection job boards
	jobBoards := []string{
		"indeed.com",
		"glassdoor.com",
		"monster.com",
		"welcometothejungle.com",
	}
	for _, board := range jobBoards {
		if strings.Contains(refererLower, board) {
			return 15, ProfileTypeRecruiter
		}
	}

	// Détection GitHub
	if strings.Contains(refererLower, "github.com") {
		return 10, ProfileTypeDeveloper
	}

	return 0, ProfileTypeOther
}

// analyzeEnrichmentData analyse les données Clearbit pour déterminer le profil
func (s *ProfileDetectorService) analyzeEnrichmentData(data map[string]interface{}) (int, ProfileType) {
	// Extraire company type
	companyType, _ := data["company_type"].(string)
	jobTitle, _ := data["job_title"].(string)
	industry, _ := data["industry"].(string)

	score := 0
	profileType := ProfileTypeOther

	// Analyse du company type
	if strings.Contains(strings.ToLower(companyType), "recruiting") ||
	   strings.Contains(strings.ToLower(industry), "recruiting") {
		score += 40
		profileType = ProfileTypeRecruiter
	}

	// Analyse du job title
	if jobTitle != "" {
		jobTitleLower := strings.ToLower(jobTitle)

		if strings.Contains(jobTitleLower, "cto") ||
		   strings.Contains(jobTitleLower, "chief technology") {
			score += 50
			profileType = ProfileTypeCTO
		} else if strings.Contains(jobTitleLower, "ceo") ||
		          strings.Contains(jobTitleLower, "chief executive") {
			score += 50
			profileType = ProfileTypeCEO
		} else if strings.Contains(jobTitleLower, "tech lead") ||
		          strings.Contains(jobTitleLower, "engineering manager") ||
		          strings.Contains(jobTitleLower, "vp eng") {
			score += 40
			profileType = ProfileTypeTechLead
		} else if strings.Contains(jobTitleLower, "recruiter") ||
		          strings.Contains(jobTitleLower, "talent") ||
		          strings.Contains(jobTitleLower, "hr") {
			score += 40
			profileType = ProfileTypeRecruiter
		}
	}

	return score, profileType
}

// isRecruiterBot vérifie si le User-Agent correspond à un bot recruteur
func isRecruiterBot(userAgent string) bool {
	botPatterns := []string{
		"LinkedInBot",
		"HubSpot",
		"Workable",
		"LeverBot",
		"SmashFly",
		"PeopleClick",
		"JobviteBot",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(userAgent, pattern) {
			return true
		}
	}
	return false
}

// calculateFinalConfidence calcule le score de confiance final
// Formule: (userAgentScore * 0.3) + (refererScore * 0.2) + (ipScore * 0.5)
func calculateFinalConfidence(userAgentScore, refererScore, ipScore int) int {
	confidence := int(float64(userAgentScore)*0.3 + float64(refererScore)*0.2 + float64(ipScore)*0.5)
	if confidence > 100 {
		return 100
	}
	return confidence
}

// hashIP hash l'IP avec SHA-256 pour la privacy (RGPD)
func hashIP(ip string) string {
	// Ajouter un salt pour plus de sécurité (devrait être en env variable)
	salt := "maicivy_ip_salt_2025"
	hash := sha256.Sum256([]byte(ip + salt))
	return hex.EncodeToString(hash[:])
}

// ShouldBypassAccessGate détermine si un profil devrait bypasser l'access gate
func (s *ProfileDetectorService) ShouldBypassAccessGate(profile *DetectedProfile) bool {
	// Bypass si profil détecté avec confiance >= 60%
	if profile.Confidence >= 60 {
		bypassProfiles := []ProfileType{
			ProfileTypeRecruiter,
			ProfileTypeCTO,
			ProfileTypeTechLead,
			ProfileTypeCEO,
		}

		for _, bp := range bypassProfiles {
			if profile.ProfileType == bp {
				return true
			}
		}
	}
	return false
}

// StoreBypassInRedis stocke le bypass en Redis pour la session
func (s *ProfileDetectorService) StoreBypassInRedis(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("access:bypass:%s", sessionID)
	// TTL 30 jours
	return s.redis.Set(ctx, key, "1", 30*24*time.Hour).Err()
}

// CheckBypassExists vérifie si un bypass existe pour la session
func (s *ProfileDetectorService) CheckBypassExists(ctx context.Context, sessionID string) (bool, error) {
	key := fmt.Sprintf("access:bypass:%s", sessionID)
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}
