package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// GitHubSyncService gère la synchronisation des repos GitHub
type GitHubSyncService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewGitHubSyncService crée une nouvelle instance
func NewGitHubSyncService(db *gorm.DB, redis *redis.Client) *GitHubSyncService {
	return &GitHubSyncService{
		db:    db,
		redis: redis,
	}
}

// SyncRepositories récupère tous les repos de l'utilisateur et les sauvegarde
func (s *GitHubSyncService) SyncRepositories(ctx context.Context, username, token string) error {
	// Créer client GitHub authentifié
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Lister tous les repos (publics et privés accessibles)
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "updated",
		Direction:   "desc",
	}

	var allRepos []*github.Repository

	// Pagination
	for {
		repos, resp, err := client.Repositories.List(ctx, "", opts)
		if err != nil {
			return fmt.Errorf("failed to list repos: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Sauvegarder dans PostgreSQL
	for _, ghRepo := range allRepos {
		// Transformation GitHub repo → model Project
		repo := &models.GitHubRepository{
			Username:    username,
			RepoName:    ghRepo.GetName(),
			FullName:    ghRepo.GetFullName(),
			Description: ghRepo.GetDescription(),
			URL:         ghRepo.GetHTMLURL(),
			Stars:       int32(ghRepo.GetStargazersCount()),
			Language:    ghRepo.GetLanguage(),
			Topics:      pq.StringArray(ghRepo.Topics),
			IsPrivate:   ghRepo.GetPrivate(),
			PushedAt:    ghRepo.GetPushedAt().Time,
		}

		// Upsert (créer ou mettre à jour si existe)
		// On utilise FullName comme identifiant unique
		var existing models.GitHubRepository
		result := s.db.Where("username = ? AND full_name = ?", username, repo.FullName).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Créer nouveau
			if err := s.db.Create(repo).Error; err != nil {
				return fmt.Errorf("failed to create repo %s: %w", repo.FullName, err)
			}
		} else if result.Error == nil {
			// Mettre à jour existant
			repo.ID = existing.ID
			if err := s.db.Save(repo).Error; err != nil {
				return fmt.Errorf("failed to update repo %s: %w", repo.FullName, err)
			}
		} else {
			return fmt.Errorf("database error: %w", result.Error)
		}
	}

	// Mettre à jour timestamp de sync dans GitHubProfile
	err := s.db.Model(&models.GitHubProfile{}).
		Where("username = ?", username).
		Update("synced_at", time.Now().Unix()).Error

	if err != nil {
		return fmt.Errorf("failed to update sync timestamp: %w", err)
	}

	// Invalider cache Redis
	s.redis.Del(ctx, "github:repos:"+username)

	return nil
}

// GetPublicRepositories retourne les repos publics à afficher (avec cache)
func (s *GitHubSyncService) GetPublicRepositories(ctx context.Context, username string) ([]models.GitHubRepository, error) {
	// Essayer le cache d'abord
	cacheKey := "github:repos:" + username
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var repos []models.GitHubRepository
		if err := json.Unmarshal([]byte(cached), &repos); err == nil {
			return repos, nil
		}
	}

	// Requête DB si cache miss
	var repos []models.GitHubRepository
	if err := s.db.Where("username = ? AND is_private = false", username).
		Order("stars DESC, pushed_at DESC").
		Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}

	// Mettre en cache (TTL 24h)
	data, _ := json.Marshal(repos)
	s.redis.Set(ctx, cacheKey, string(data), 24*time.Hour)

	return repos, nil
}

// GetAllRepositories retourne tous les repos (publics + privés) pour l'owner
func (s *GitHubSyncService) GetAllRepositories(ctx context.Context, username string) ([]models.GitHubRepository, error) {
	var repos []models.GitHubRepository
	if err := s.db.Where("username = ?", username).
		Order("stars DESC, pushed_at DESC").
		Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}

	return repos, nil
}

// GetSyncStatus retourne le statut de synchronisation
func (s *GitHubSyncService) GetSyncStatus(ctx context.Context, username string) (*SyncStatus, error) {
	var profile models.GitHubProfile
	if err := s.db.Where("username = ?", username).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &SyncStatus{
				Connected:  false,
				LastSync:   0,
				RepoCount:  0,
			}, nil
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Compter les repos
	var count int64
	s.db.Model(&models.GitHubRepository{}).Where("username = ?", username).Count(&count)

	return &SyncStatus{
		Connected:  true,
		Username:   profile.Username,
		LastSync:   profile.SyncedAt,
		RepoCount:  int(count),
	}, nil
}

// SyncStatus représente le statut de sync GitHub
type SyncStatus struct {
	Connected bool   `json:"connected"`
	Username  string `json:"username,omitempty"`
	LastSync  int64  `json:"last_sync"`
	RepoCount int    `json:"repo_count"`
}

// DisconnectGitHub déconnecte un utilisateur GitHub
func (s *GitHubSyncService) DisconnectGitHub(ctx context.Context, username string) error {
	// Supprimer le profil
	if err := s.db.Where("username = ?", username).Delete(&models.GitHubProfile{}).Error; err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Note: On garde les repos importés dans la DB
	// Si vous voulez les supprimer aussi, décommentez:
	// s.db.Where("username = ?", username).Delete(&models.GitHubRepository{})

	// Invalider cache
	s.redis.Del(ctx, "github:repos:"+username)

	return nil
}
