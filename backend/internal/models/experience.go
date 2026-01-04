package models

import (
	"time"

	"github.com/lib/pq"
)

// Experience représente une expérience professionnelle
type Experience struct {
	BaseModel

	// Informations principales
	Title       string    `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=3,max=255"`
	Company     string    `gorm:"type:varchar(255);not null" json:"company" validate:"required,min=2,max=255"`
	Description string    `gorm:"type:text" json:"description" validate:"max=5000"`
	StartDate   time.Time `gorm:"not null" json:"startDate" validate:"required"`
	EndDate     *time.Time `json:"endDate"` // Nullable pour emploi actuel

	// Catégorisation et filtrage
	Technologies pq.StringArray `gorm:"type:text[]" json:"technologies"` // PostgreSQL array
	Tags         pq.StringArray `gorm:"type:text[]" json:"tags"`
	Category     string         `gorm:"type:varchar(100);index" json:"category" validate:"required,oneof=backend frontend fullstack devops data ai mobile other"`

	// Métadonnées
	Featured bool `gorm:"default:false" json:"featured"` // Pour mise en avant
}

// TableName override le nom de table par défaut
func (Experience) TableName() string {
	return "experiences"
}

// IsCurrentJob vérifie si c'est l'emploi actuel
func (e *Experience) IsCurrentJob() bool {
	return e.EndDate == nil
}

// Duration calcule la durée de l'expérience
func (e *Experience) Duration() time.Duration {
	endDate := time.Now()
	if e.EndDate != nil {
		endDate = *e.EndDate
	}
	return endDate.Sub(e.StartDate)
}
