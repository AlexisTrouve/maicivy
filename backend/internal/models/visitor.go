package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ProfileType représente le type de profil détecté
type ProfileType string

const (
	ProfileTypeUnknown   ProfileType = "unknown"
	ProfileTypeRecruiter ProfileType = "recruiter"
	ProfileTypeTechLead  ProfileType = "tech_lead"
	ProfileTypeCTO       ProfileType = "cto"
	ProfileTypeCEO       ProfileType = "ceo"
	ProfileTypeDeveloper ProfileType = "developer"
	ProfileTypeOther     ProfileType = "other"
)

// EnrichmentData contient les données enrichies via Clearbit
type EnrichmentData map[string]interface{}

// Scan implémente sql.Scanner pour JSONB
func (e *EnrichmentData) Scan(value interface{}) error {
	if value == nil {
		*e = make(EnrichmentData)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, e)
}

// Value implémente driver.Valuer pour JSONB
func (e EnrichmentData) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}
	return json.Marshal(e)
}

// Visitor représente un visiteur unique du site
type Visitor struct {
	BaseModel

	// Identification
	SessionID string `gorm:"type:varchar(255);uniqueIndex;not null" json:"session_id" validate:"required"`
	IPHash    string `gorm:"type:varchar(64);index" json:"ip_hash"` // Hash SHA256 de l'IP (RGPD)

	// User Agent parsing
	UserAgent string `gorm:"type:text" json:"user_agent"`
	Browser   string `gorm:"type:varchar(100)" json:"browser"`
	OS        string `gorm:"type:varchar(100)" json:"os"`
	Device    string `gorm:"type:varchar(50)" json:"device"` // mobile, tablet, desktop

	// Tracking visites
	VisitCount int       `gorm:"default:1" json:"visit_count"`
	FirstVisit time.Time `gorm:"not null" json:"first_visit"`
	LastVisit  time.Time `gorm:"not null" json:"last_visit"`

	// Détection profil (NOUVEAU - Feature 3)
	ProfileDetected      ProfileType    `gorm:"type:varchar(50);default:'unknown'" json:"profile_detected"`
	ProfileType          string         `gorm:"type:varchar(50);index" json:"profile_type"` // recruiter, cto, tech_lead, ceo, other
	// EnrichmentData field removed - column doesn't exist in database and Clearbit integration not configured
	// EnrichmentData       EnrichmentData `gorm:"type:jsonb" json:"enrichment_data,omitempty"` // Données Clearbit
	DetectionConfidence  int            `gorm:"default:0" json:"detection_confidence"` // 0-100%
	CompanyName          string         `gorm:"type:varchar(255)" json:"company_name"` // Via IP lookup ou LinkedIn
	LinkedInURL          string         `gorm:"type:varchar(500)" json:"linkedin_url"` // Si détecté via referrer

	// Géolocalisation (via IP)
	Country     string `gorm:"type:varchar(100)" json:"country"`
	City        string `gorm:"type:varchar(100)" json:"city"`
	Timezone    string `gorm:"type:varchar(50)" json:"timezone"`

	// Relations
	GeneratedLetters []GeneratedLetter `gorm:"foreignKey:VisitorID" json:"generated_letters,omitempty"`
	AnalyticsEvents  []AnalyticsEvent  `gorm:"foreignKey:VisitorID" json:"analytics_events,omitempty"`
}

// TableName override le nom de table par défaut
func (Visitor) TableName() string {
	return "visitors"
}

// HasAccessToAI vérifie si le visiteur a accès aux fonctionnalités IA
func (v *Visitor) HasAccessToAI() bool {
	// Règle: 3+ visites OU profil cible détecté
	if v.VisitCount >= 3 {
		return true
	}

	targetProfiles := []ProfileType{
		ProfileTypeRecruiter,
		ProfileTypeTechLead,
		ProfileTypeCTO,
		ProfileTypeCEO,
	}

	for _, profile := range targetProfiles {
		if v.ProfileDetected == profile {
			return true
		}
	}

	return false
}

// IsTargetProfile vérifie si c'est un profil cible (recruteur, etc.)
func (v *Visitor) IsTargetProfile() bool {
	return v.ProfileDetected != ProfileTypeUnknown && v.ProfileDetected != ProfileTypeDeveloper
}

// IncrementVisit incrémente le compteur de visites
func (v *Visitor) IncrementVisit() {
	v.VisitCount++
	v.LastVisit = time.Now()
}
