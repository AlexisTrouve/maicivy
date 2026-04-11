package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// BlogPost représente un article de blog généré depuis des commits
type BlogPost struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Slug               string         `gorm:"uniqueIndex;not null" json:"slug"`
	Title              string         `gorm:"not null" json:"title"`
	Summary            string         `json:"summary"`
	Content            string         `gorm:"type:text" json:"content"`
	ContentHTML        string         `gorm:"type:text" json:"content_html"`
	ProjectName        string         `gorm:"index" json:"project_name"`
	Tags               pq.StringArray `gorm:"type:text[]" json:"tags"`
	GeneratedFromCommits CommitRefList `gorm:"type:jsonb" json:"generated_from_commits"`
	CoverImageURL      string         `json:"cover_image_url,omitempty"`
	ReadingTimeMinutes int            `json:"reading_time_minutes"`
	Published          bool           `gorm:"default:false;index" json:"published"`
	PublishedAt        *time.Time     `json:"published_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// TableName définit le nom de table
func (BlogPost) TableName() string {
	return "blog_posts"
}

// CommitRef référence un commit utilisé pour générer l'article
type CommitRef struct {
	SHA       string `json:"sha"`
	Message   string `json:"message"`
	Date      string `json:"date"`
	Project   string `json:"project"`
}

// CommitRefList est une liste de références de commits pour GORM JSONB
type CommitRefList []CommitRef

// Scan implémente sql.Scanner pour GORM
func (cl *CommitRefList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &cl)
}

// Value implémente driver.Valuer pour GORM
func (cl CommitRefList) Value() (driver.Value, error) {
	return json.Marshal(cl)
}

// BlogPostListResponse représente la réponse liste d'articles
type BlogPostListResponse struct {
	Posts      []BlogPost `json:"posts"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
	TotalPages int        `json:"total_pages"`
}

// BlogGenerateRequest représente une demande de génération d'article
type BlogGenerateRequest struct {
	ProjectName string   `json:"project_name"`
	CommitSHAs  []string `json:"commit_shas,omitempty"`
	AutoSelect  bool     `json:"auto_select"` // Si true, sélectionne automatiquement les commits récents significatifs
}

// BlogCreateRequest représente une demande de création d'article directe (ex: WanMira)
type BlogCreateRequest struct {
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Content       string   `json:"content"`         // Markdown
	ProjectName   string   `json:"project_name"`
	Tags          []string `json:"tags"`
	CoverImageURL string   `json:"cover_image_url,omitempty"`
	Publish       bool     `json:"publish"`          // Si true, publie immédiatement
}

// BlogUpdateRequest représente une demande de mise à jour d'article
type BlogUpdateRequest struct {
	Title         *string  `json:"title,omitempty"`
	Summary       *string  `json:"summary,omitempty"`
	Content       *string  `json:"content,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CoverImageURL *string  `json:"cover_image_url,omitempty"`
}
