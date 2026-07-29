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
	Lang        string      `json:"lang" validate:"omitempty,oneof=fr en"` // Langue: fr ou en
	JobOffer    string      `json:"job_offer,omitempty"`                   // Texte brut de l'offre (optionnel)
	Model       string      `json:"model,omitempty"`                       // Override du modèle Claude (optionnel, usage interne)
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
	Address     string             `json:"address"`     // Adresse complète
	PostalCode  string             `json:"postal_code"` // Code postal
	City        string             `json:"city"`        // Ville
	Email       string             `json:"email"`       // Email
	Phone       string             `json:"phone"`       // Téléphone
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
	Model         string      `json:"model"`    // modèle effectif, ex: "claude-opus-4-6"
	TokensUsed    int         `json:"tokens_used"`
	EstimatedCost float64     `json:"estimated_cost"`
}

// PlatformMessageRequest : requête de génération de message plateforme (Malt, LinkedIn...)
type PlatformMessageRequest struct {
	Mission  string `json:"mission" validate:"required,min=20"`                       // Description de la mission copiée-collée
	Platform string `json:"platform" validate:"omitempty,oneof=malt linkedin upwork"` // Plateforme cible
	TJM      int    `json:"tjm" validate:"omitempty,min=50,max=5000"`                 // Tarif journalier en euros
	Lang     string `json:"lang" validate:"omitempty,oneof=fr en"`
}

// PlatformMessageResponse : message généré prêt à envoyer
type PlatformMessageResponse struct {
	Content       string  `json:"content"`
	Platform      string  `json:"platform"`
	TokensUsed    int     `json:"tokens_used"`
	EstimatedCost float64 `json:"estimated_cost"`
	Model         string  `json:"model"`
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
