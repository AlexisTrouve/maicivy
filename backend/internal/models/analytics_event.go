package models

import (
	"github.com/google/uuid"
)

// EventType représente les types d'événements trackés
type EventType string

const (
	EventTypePageView       EventType = "page_view"
	EventTypeCVThemeChange  EventType = "cv_theme_change"
	EventTypeLetterGenerate EventType = "letter_generate"
	EventTypePDFDownload    EventType = "pdf_download"
	EventTypeButtonClick    EventType = "button_click"
	EventTypeLinkClick      EventType = "link_click"
	EventTypeFormSubmit     EventType = "form_submit"
)

// AnalyticsEvent représente un événement analytics
type AnalyticsEvent struct {
	BaseModel

	// Relation avec visiteur
	VisitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"visitor_id"`
	Visitor   *Visitor  `gorm:"foreignKey:VisitorID" json:"visitor,omitempty"`

	// Type d'événement
	EventType EventType `gorm:"type:varchar(50);not null;index" json:"event_type" validate:"required"`

	// Données événement (flexible JSONB)
	EventData string `gorm:"type:jsonb" json:"event_data"`

	// Contexte page
	PageURL  string `gorm:"type:varchar(500)" json:"page_url"`
	Referrer string `gorm:"type:varchar(500)" json:"referrer"`

	// Métadonnées
	SessionDuration int `gorm:"default:0" json:"session_duration"` // En secondes
}

// TableName override le nom de table par défaut
func (AnalyticsEvent) TableName() string {
	return "analytics_events"
}

// IsPageView vérifie si c'est un page view
func (ae *AnalyticsEvent) IsPageView() bool {
	return ae.EventType == EventTypePageView
}

// IsConversion vérifie si c'est un événement de conversion
func (ae *AnalyticsEvent) IsConversion() bool {
	conversionEvents := []EventType{
		EventTypeLetterGenerate,
		EventTypePDFDownload,
	}

	for _, event := range conversionEvents {
		if ae.EventType == event {
			return true
		}
	}

	return false
}
