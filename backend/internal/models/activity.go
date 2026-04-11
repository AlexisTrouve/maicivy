package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// ActivityProject représente un projet dans le feed d'activité
type ActivityProject struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"uniqueIndex;not null" json:"name"`
	Description   string         `json:"description"`
	RepoURL       string         `json:"repo_url"`
	Category      string         `json:"category"` // "WIP", "Production", "Archive"
	Showcase      bool           `gorm:"default:false" json:"showcase"`
	Languages     pq.StringArray `gorm:"type:text[]" json:"languages"`
	Commits7d     int            `gorm:"column:commits7d" json:"commits_7d"`
	Commits30d    int            `gorm:"column:commits30d" json:"commits_30d"`
	RecentCommits CommitList     `gorm:"type:jsonb" json:"recent_commits"`
	LastActivity  time.Time      `json:"last_activity"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// TableName définit le nom de table
func (ActivityProject) TableName() string {
	return "activity_projects"
}

// Commit représente un commit récent
type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date"`
	Author  string `json:"author"`
}

// CommitList est une liste de commits pour GORM JSONB
type CommitList []Commit

// Scan implémente sql.Scanner pour GORM
func (cl *CommitList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &cl)
}

// Value implémente driver.Valuer pour GORM
func (cl CommitList) Value() (driver.Value, error) {
	return json.Marshal(cl)
}

// ActivityStats représente les statistiques globales
type ActivityStats struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	TotalCommits30d int            `json:"total_commits_30d"`
	ActiveProjects  int            `json:"active_projects"`
	TopLanguages    pq.StringArray `gorm:"type:text[]" json:"top_languages"`
	LastUpdated     time.Time      `json:"last_updated"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// TableName définit le nom de table
func (ActivityStats) TableName() string {
	return "activity_stats"
}

// ActivityFeedResponse représente la réponse API complète
type ActivityFeedResponse struct {
	LastUpdated string            `json:"last_updated"`
	Projects    []ActivityProject `json:"projects"`
	Stats       ActivityStats     `json:"stats"`
}
