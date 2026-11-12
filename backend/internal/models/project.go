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

	// i18n - English translations
	TitleEn                 string `gorm:"type:varchar(255);column:title_en" json:"titleEn,omitempty"`
	DescriptionEn           string `gorm:"type:text;column:description_en" json:"descriptionEn,omitempty"`
	CatchphraseEn           string `gorm:"type:varchar(200);column:catchphrase_en" json:"catchphraseEn,omitempty"`
	FunctionalDescriptionEn string `gorm:"type:text;column:functional_description_en" json:"functionalDescriptionEn,omitempty"`
	TechnicalDescriptionEn  string `gorm:"type:text;column:technical_description_en" json:"technicalDescriptionEn,omitempty"`

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
	// Tags — catégorisation maiProFiles (inclut les flags-concept: "Scraping", "AI / LLM Integration"…).
	// Sert AU MATCHING skill→projet côté frontend (en plus de Technologies), PAS à l'affichage du stack.
	// Non persisté (gorm:"-") : provient de maiProFiles, jamais de la DB.
	Tags pq.StringArray `gorm:"-" json:"tags"`

	// Proof — ligne de PREUVE affichée sous l'accroche sur le CV : « en production · 94 k lignes ·
	// 1,4 k tests ». Déjà dans la langue servie (construite par le content provider, cf.
	// services.ligneDePreuve).
	// POURQUOI : sur Malt, le seul prospect qui a vraiment échangé a demandé des preuves vérifiables,
	// et le CV n'affichait que des pastilles de technos. maiProFiles portait déjà status et stats ;
	// le CV les ignorait.
	// Non persisté (gorm:"-") : provient de maiProFiles, jamais de la DB.
	Proof    string `gorm:"-" json:"proof,omitempty"`
	Category string `gorm:"type:varchar(100);index" json:"category" validate:"required"`

	// Signaux de CLASSEMENT des projets sur le CV (cf. services/cv_project_ranking.go). Tous
	// proviennent de maiProFiles (et de Gitea pour l'activité), jamais de la DB.
	//
	// SourceID : identifiant maiProFiles, STABLE. Les scores du classement sont indexés dessus et
	// non sur un slug du titre — un titre se réécrit, se traduit, et le slug changeait avec lui.
	SourceID string `gorm:"-" json:"sourceId,omitempty"`
	// Status : statut maiProFiles brut (production, active, wip, R&D, retired…).
	Status string `gorm:"-" json:"status,omitempty"`
	// LOC / TestCount : ampleur du projet (0 = inconnu).
	LOC       int `gorm:"-" json:"loc,omitempty"`
	TestCount int `gorm:"-" json:"testCount,omitempty"`
	// Activity : activité git récente, 0..1 (longévité × régularité × récence des commits Gitea).
	// 0 = aucune activité connue (pas de dépôt relié, ou dormant).
	Activity float64 `gorm:"-" json:"activity,omitempty"`

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
