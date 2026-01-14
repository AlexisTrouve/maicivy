package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// LinkData represents a link object for experiences and projects
type LinkData struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon,omitempty"`
}

// LinksJSON is a custom type for JSON array of links
type LinksJSON []LinkData

// Value implements driver.Valuer for database storage
func (l LinksJSON) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	return json.Marshal(l)
}

// Scan implements sql.Scanner for database retrieval
func (l *LinksJSON) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, l)
}

// GormDataType returns the GORM data type for LinksJSON
func (LinksJSON) GormDataType() string {
	return "jsonb"
}

// Experience represents a professional experience
type Experience struct {
	BaseModel

	// Main information
	Title       string     `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=3,max=255"`
	Company     string     `gorm:"type:varchar(255);not null" json:"company" validate:"required,min=2,max=255"`
	Description string     `gorm:"type:text" json:"description" validate:"max=5000"`
	StartDate   time.Time  `gorm:"not null" json:"startDate" validate:"required"`
	EndDate     *time.Time `json:"endDate"` // Nullable for current job

	// Short catchphrase for card display
	Catchphrase string `gorm:"type:varchar(200)" json:"catchphrase"`

	// Detailed descriptions for modal
	FunctionalDescription string `gorm:"type:text" json:"functionalDescription"`
	TechnicalDescription  string `gorm:"type:text" json:"technicalDescription"`

	// Images (array of URLs for gallery)
	Images pq.StringArray `gorm:"type:text[]" json:"images"`

	// Links (JSON array of link objects: [{name, url, icon}])
	Links LinksJSON `gorm:"type:jsonb" json:"links"`

	// Categorization and filtering
	Technologies pq.StringArray `gorm:"type:text[]" json:"technologies"` // PostgreSQL array
	Tags         pq.StringArray `gorm:"type:text[]" json:"tags"`
	Category     string         `gorm:"type:varchar(100);index" json:"category" validate:"required,oneof=backend frontend fullstack devops data ai mobile other"`

	// Metadata
	Featured bool `gorm:"default:false" json:"featured"` // For highlighting
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
