package dto

import (
	"github.com/go-playground/validator/v10"
)

// --- REQUESTS ---

// GenerateLetterRequest requête pour générer une lettre
type GenerateLetterRequest struct {
	CompanyName string `json:"company_name" validate:"required,min=2,max=200"`
	JobTitle    string `json:"job_title,omitempty" validate:"omitempty,min=2,max=200"` // Optionnel
	Theme       string `json:"theme,omitempty" validate:"omitempty,oneof=backend frontend fullstack devops data ai"`
}

// Validate valide la requête
func (r *GenerateLetterRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// --- RESPONSES ---

// LetterGenerationResponse réponse après enqueue du job de génération
type LetterGenerationResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"` // "queued"
	Message string `json:"message"`

	// Informations rate limiting
	RateLimitRemaining int `json:"rate_limit_remaining"`
}

// LetterJobStatus status d'un job de génération
type LetterJobStatus struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`  // "queued", "processing", "completed", "failed"
	Progress int   `json:"progress"` // 0-100

	// Si completed
	LetterMotivationID     *string `json:"letter_motivation_id,omitempty"` // UUID as string
	LetterAntiMotivationID *string `json:"letter_anti_motivation_id,omitempty"` // UUID as string

	// Si failed
	Error *string `json:"error,omitempty"`

	// Temps estimé restant (secondes)
	EstimatedTime *int `json:"estimated_time,omitempty"`
}

// LetterDetailResponse détails d'une lettre générée
type LetterDetailResponse struct {
	ID          string `json:"id"` // UUID as string
	CompanyName string `json:"company_name"`
	LetterType  string `json:"letter_type"` // "motivation" ou "anti_motivation"
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`

	// Métadonnées
	AIModel      string  `json:"ai_model"`
	TokensUsed   int     `json:"tokens_used"`
	GenerationMS int     `json:"generation_ms"`
	Cost         float64 `json:"cost"` // Coût estimé en USD

	// URL de téléchargement PDF
	PDFURL string `json:"pdf_url,omitempty"`
}

// LetterPairResponse paire de lettres (motivation + anti-motivation)
type LetterPairResponse struct {
	MotivationLetter     *LetterDetailResponse `json:"motivation_letter"`
	AntiMotivationLetter *LetterDetailResponse `json:"anti_motivation_letter"`
	CompanyName          string                `json:"company_name"`
	CompanyInfo          interface{}           `json:"company_info,omitempty"` // Données scrapées
}

// LetterHistoryResponse historique des lettres générées
type LetterHistoryResponse struct {
	Letters []LetterHistoryItem `json:"letters"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
}

// LetterHistoryItem item d'historique de lettre
type LetterHistoryItem struct {
	ID          string `json:"id"` // UUID as string
	CompanyName string `json:"company_name"`
	LetterType  string `json:"letter_type"`
	CreatedAt   string `json:"created_at"`
	Downloaded  bool   `json:"downloaded"`
}

// AccessStatusResponse status d'accès aux fonctionnalités IA
type AccessStatusResponse struct {
	HasAccess         bool   `json:"has_access"`
	CurrentVisits     int    `json:"current_visits"`
	RequiredVisits    int    `json:"required_visits"`
	VisitsRemaining   int    `json:"visits_remaining"`
	ProfileDetected   string `json:"profile_detected,omitempty"`
	AccessGrantedBy   string `json:"access_granted_by"` // "visits" ou "profile"
	Message           string `json:"message"`
}

// RateLimitStatusResponse status du rate limiting IA
type RateLimitStatusResponse struct {
	DailyLimit        int    `json:"daily_limit"`
	DailyUsed         int    `json:"daily_used"`
	DailyRemaining    int    `json:"daily_remaining"`
	ResetAt           string `json:"reset_at"`
	CooldownActive    bool   `json:"cooldown_active"`
	CooldownRemaining int    `json:"cooldown_remaining"` // en secondes
}
