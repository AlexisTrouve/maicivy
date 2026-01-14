package models

import (
	"github.com/lib/pq"
)

// Project represents a realized project
type Project struct {
	BaseModel

	// Main information
	Title       string `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=3,max=255"`
	Description string `gorm:"type:text" json:"description" validate:"max=5000"`

	// Short catchphrase for card display
	Catchphrase string `gorm:"type:varchar(200)" json:"catchphrase"`

	// Detailed descriptions for modal
	FunctionalDescription string `gorm:"type:text" json:"functionalDescription"`
	TechnicalDescription  string `gorm:"type:text" json:"technicalDescription"`

	// URLs
	GithubURL string `gorm:"type:varchar(500)" json:"githubUrl" validate:"omitempty,url"`
	DemoURL   string `gorm:"type:varchar(500)" json:"demoUrl" validate:"omitempty,url"`
	ImageURL  string `gorm:"type:varchar(500)" json:"imageUrl" validate:"omitempty,url"`

	// Images (array of URLs for gallery)
	Images pq.StringArray `gorm:"type:text[]" json:"images"`

	// Links (JSON array of link objects: [{name, url, icon}])
	Links LinksJSON `gorm:"type:jsonb" json:"links"`

	// Categorization
	Technologies pq.StringArray `gorm:"type:text[]" json:"technologies"`
	Category     string         `gorm:"type:varchar(100);index" json:"category" validate:"required"`

	// GitHub metadata (synced automatically)
	GithubStars    int    `gorm:"default:0" json:"githubStars"`
	GithubForks    int    `gorm:"default:0" json:"githubForks"`
	GithubLanguage string `gorm:"type:varchar(50)" json:"githubLanguage"`

	// Flags
	Featured   bool `gorm:"default:false" json:"featured"`
	InProgress bool `gorm:"default:false" json:"inProgress"`
}

// TableName override le nom de table par défaut
func (Project) TableName() string {
	return "projects"
}

// HasGithub vérifie si le projet a un repo GitHub
func (p *Project) HasGithub() bool {
	return p.GithubURL != ""
}

// HasDemo vérifie si le projet a une démo live
func (p *Project) HasDemo() bool {
	return p.DemoURL != ""
}
