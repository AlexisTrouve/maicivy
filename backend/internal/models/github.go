package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// GitHubToken structure OAuth token
type GitHubToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   int64  `json:"expires_at"`
}

// Scan pour GORM - permet de lire depuis PostgreSQL JSONB
func (gt *GitHubToken) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &gt)
}

// Value pour GORM - permet d'écrire en PostgreSQL JSONB
func (gt GitHubToken) Value() (driver.Value, error) {
	return json.Marshal(gt)
}

// GitHubProfile utilisateur GitHub connecté
type GitHubProfile struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	Username    string      `gorm:"uniqueIndex;not null" json:"username"`
	Token       GitHubToken `gorm:"type:jsonb" json:"-"` // Ne pas exposer en JSON
	ConnectedAt int64       `json:"connected_at"`
	SyncedAt    int64       `json:"synced_at"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// TableName définit le nom de table
func (GitHubProfile) TableName() string {
	return "github_profiles"
}

// GitHubRepository représente un repo GitHub importé
type GitHubRepository struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"index;not null" json:"username"`
	RepoName    string         `json:"repo_name"`
	FullName    string         `gorm:"uniqueIndex:idx_username_fullname" json:"full_name"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	Stars       int32          `json:"stars"`
	Language    string         `json:"language"`
	Topics      pq.StringArray `gorm:"type:text[]" json:"topics"`
	IsPrivate   bool           `gorm:"default:false" json:"is_private"`
	PushedAt    time.Time      `json:"pushed_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TableName définit le nom de table
func (GitHubRepository) TableName() string {
	return "github_repositories"
}
