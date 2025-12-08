package models

import (
	"github.com/google/uuid"
)

// LetterType représente le type de lettre générée
type LetterType string

const (
	LetterTypeMotivation     LetterType = "motivation"
	LetterTypeAntiMotivation LetterType = "anti_motivation"
)

// GeneratedLetter représente une lettre générée par IA
type GeneratedLetter struct {
	BaseModel

	// Relation avec visiteur
	VisitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"visitor_id"`
	Visitor   *Visitor  `gorm:"foreignKey:VisitorID" json:"visitor,omitempty"`

	// Informations lettre
	CompanyName string     `gorm:"type:varchar(255);not null" json:"company_name" validate:"required,min=2,max=255"`
	LetterType  LetterType `gorm:"type:varchar(20);not null" json:"letter_type" validate:"required,oneof=motivation anti_motivation"`
	Content     string     `gorm:"type:text;not null" json:"content" validate:"required"`

	// Métadonnées génération
	AIModel      string `gorm:"type:varchar(50)" json:"ai_model"`       // "claude-3" ou "gpt-4"
	TokensUsed   int    `gorm:"default:0" json:"tokens_used"`           // Tracking coûts
	GenerationMS int    `gorm:"default:0" json:"generation_ms"`         // Temps de génération en ms
	CompanyInfo  string `gorm:"type:jsonb" json:"company_info"`         // Données entreprise scrapées (JSON)

	// Flags
	Downloaded bool `gorm:"default:false" json:"downloaded"` // Tracking PDF téléchargé
}

// TableName override le nom de table par défaut
func (GeneratedLetter) TableName() string {
	return "generated_letters"
}

// IsMotivation vérifie si c'est une lettre de motivation
func (gl *GeneratedLetter) IsMotivation() bool {
	return gl.LetterType == LetterTypeMotivation
}

// EstimatedCost calcule le coût estimé en USD
func (gl *GeneratedLetter) EstimatedCost() float64 {
	// Exemple: Claude Sonnet ~$3 / 1M input tokens, ~$15 / 1M output tokens
	// Estimation simplifiée : $0.01 par 1000 tokens
	return float64(gl.TokensUsed) * 0.00001
}
