package models

import "time"

// CompanyInfo : informations sur l'entreprise cible
type CompanyInfo struct {
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	Description  string   `json:"description"`
	Industry     string   `json:"industry"`
	Size         string   `json:"size"`
	Technologies []string `json:"technologies,omitempty"`
	Culture      string   `json:"culture,omitempty"`
	Values       []string `json:"values,omitempty"`
	RecentNews   string   `json:"recent_news,omitempty"`
}

// LetterRequest : requête de génération de lettre
type LetterRequest struct {
	CompanyName string      `json:"company_name" validate:"required,min=2"`
	LetterType  LetterType  `json:"letter_type" validate:"required,oneof=motivation anti_motivation"`
	UserProfile UserProfile `json:"user_profile,omitempty"`
}

// ExperienceDetail : détail d'une expérience professionnelle pour les prompts
type ExperienceDetail struct {
	Title       string   `json:"title"`
	Company     string   `json:"company"`
	Duration    string   `json:"duration"`    // ex: "2021-2023" ou "2023-présent"
	Description string   `json:"description"` // Description complète
	Highlights  []string `json:"highlights"`  // Points clés / achievements
}

// UserProfile : profil utilisateur pour personnalisation
type UserProfile struct {
	Name        string             `json:"name"`
	Address     string             `json:"address"`      // Adresse complète
	PostalCode  string             `json:"postal_code"`  // Code postal
	City        string             `json:"city"`         // Ville
	Email       string             `json:"email"`        // Email
	Phone       string             `json:"phone"`        // Téléphone
	CurrentRole string             `json:"current_role"`
	Skills      []string           `json:"skills"`
	Experience  int                `json:"experience_years"`
	Experiences []ExperienceDetail `json:"experiences"` // Expériences détaillées
	Summary     string             `json:"summary"`     // Résumé professionnel
}

// LetterResponse : lettre générée
type LetterResponse struct {
	Content       string      `json:"content"`
	Type          LetterType  `json:"type"`
	CompanyInfo   CompanyInfo `json:"company_info"`
	GeneratedAt   time.Time   `json:"generated_at"`
	Provider      string      `json:"provider"` // "claude" ou "openai"
	TokensUsed    int         `json:"tokens_used"`
	EstimatedCost float64     `json:"estimated_cost"`
}

// AIMetrics : métriques de coût et usage
type AIMetrics struct {
	Provider       string
	Model          string
	TokensInput    int
	TokensOutput   int
	TotalTokens    int
	EstimatedCost  float64
	ResponseTimeMs int64
	Success        bool
	ErrorMessage   string
}
