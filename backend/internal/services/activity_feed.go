package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// ActivityFeedService gère la récupération et le stockage du feed d'activité
type ActivityFeedService struct {
	db       *gorm.DB
	redis    *redis.Client
	feedURL  string
	client   *http.Client
}

// NewActivityFeedService crée une nouvelle instance
func NewActivityFeedService(db *gorm.DB, redis *redis.Client, feedURL string) *ActivityFeedService {
	return &ActivityFeedService{
		db:      db,
		redis:   redis,
		feedURL: feedURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FeedResponse représente la structure JSON du feed ProjectTracker
type FeedResponse struct {
	LastUpdated string        `json:"last_updated"`
	Projects    []FeedProject `json:"projects"`
	Stats       FeedStats     `json:"stats"`
}

// FeedProject représente un projet dans le feed
type FeedProject struct {
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	RepoURL       string       `json:"repo_url"`
	Category      string       `json:"category"`
	Showcase      bool         `json:"showcase"`
	Languages     []string     `json:"languages"`
	Commits7d     int          `json:"commits_7d"`
	Commits30d    int          `json:"commits_30d"`
	RecentCommits []FeedCommit `json:"recent_commits"`
}

// FeedCommit représente un commit dans le feed
type FeedCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date"`
	Author  string `json:"author"`
}

// FeedStats représente les stats globales
type FeedStats struct {
	TotalCommits30d int      `json:"total_commits_30d"`
	ActiveProjects  int      `json:"active_projects"`
	TopLanguages    []string `json:"top_languages"`
}

// FetchAndStore récupère le feed et le stocke en DB
func (s *ActivityFeedService) FetchAndStore(ctx context.Context) error {
	if s.feedURL == "" {
		log.Warn().Msg("Activity feed URL not configured, skipping fetch")
		return nil
	}

	log.Info().Str("url", s.feedURL).Msg("Fetching activity feed")

	// Fetch le feed
	req, err := http.NewRequestWithContext(ctx, "GET", s.feedURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var feed FeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// Stocker les projets
	for _, fp := range feed.Projects {
		project := s.feedProjectToModel(fp)
		if err := s.upsertProject(project); err != nil {
			log.Error().Err(err).Str("project", fp.Name).Msg("Failed to upsert project")
		}
	}

	// Stocker les stats
	stats := s.feedStatsToModel(feed.Stats)
	if err := s.upsertStats(stats); err != nil {
		log.Error().Err(err).Msg("Failed to upsert stats")
	}

	// Invalider cache
	s.redis.Del(ctx, "activity:feed")
	s.redis.Del(ctx, "activity:showcase")

	log.Info().Int("projects", len(feed.Projects)).Msg("Activity feed updated")
	return nil
}

func (s *ActivityFeedService) feedProjectToModel(fp FeedProject) *models.ActivityProject {
	commits := make(models.CommitList, len(fp.RecentCommits))
	for i, c := range fp.RecentCommits {
		commits[i] = models.Commit{
			SHA:     c.SHA,
			Message: c.Message,
			Date:    c.Date,
			Author:  c.Author,
		}
	}

	return &models.ActivityProject{
		Name:          fp.Name,
		Description:   fp.Description,
		RepoURL:       fp.RepoURL,
		Category:      fp.Category,
		Showcase:      fp.Showcase,
		Languages:     fp.Languages,
		Commits7d:     fp.Commits7d,
		Commits30d:    fp.Commits30d,
		RecentCommits: commits,
		LastActivity:  time.Now(),
	}
}

func (s *ActivityFeedService) feedStatsToModel(fs FeedStats) *models.ActivityStats {
	return &models.ActivityStats{
		TotalCommits30d: fs.TotalCommits30d,
		ActiveProjects:  fs.ActiveProjects,
		TopLanguages:    fs.TopLanguages,
		LastUpdated:     time.Now(),
	}
}

func (s *ActivityFeedService) upsertProject(project *models.ActivityProject) error {
	var existing models.ActivityProject
	result := s.db.Where("name = ?", project.Name).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return s.db.Create(project).Error
	} else if result.Error == nil {
		project.ID = existing.ID
		project.CreatedAt = existing.CreatedAt
		return s.db.Save(project).Error
	}
	return result.Error
}

func (s *ActivityFeedService) upsertStats(stats *models.ActivityStats) error {
	var existing models.ActivityStats
	result := s.db.First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return s.db.Create(stats).Error
	} else if result.Error == nil {
		stats.ID = existing.ID
		stats.CreatedAt = existing.CreatedAt
		return s.db.Save(stats).Error
	}
	return result.Error
}

// GetShowcaseProjects retourne les projets à afficher sur le CV
func (s *ActivityFeedService) GetShowcaseProjects(ctx context.Context) ([]models.ActivityProject, error) {
	// Essayer le cache
	cacheKey := "activity:showcase"
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var projects []models.ActivityProject
		if err := json.Unmarshal([]byte(cached), &projects); err == nil {
			return projects, nil
		}
	}

	// Query DB
	var projects []models.ActivityProject
	if err := s.db.Where("showcase = true").
		Order("commits30d DESC, last_activity DESC").
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch showcase projects: %w", err)
	}

	// Cache (1h)
	data, _ := json.Marshal(projects)
	s.redis.Set(ctx, cacheKey, string(data), time.Hour)

	return projects, nil
}

// GetAllProjects retourne tous les projets
func (s *ActivityFeedService) GetAllProjects(ctx context.Context) ([]models.ActivityProject, error) {
	var projects []models.ActivityProject
	if err := s.db.Order("commits30d DESC, last_activity DESC").
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	return projects, nil
}

// GetStats retourne les statistiques globales
func (s *ActivityFeedService) GetStats(ctx context.Context) (*models.ActivityStats, error) {
	var stats models.ActivityStats
	if err := s.db.First(&stats).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &models.ActivityStats{}, nil
		}
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	return &stats, nil
}

// GetFeed retourne le feed complet formaté pour l'API
func (s *ActivityFeedService) GetFeed(ctx context.Context, showcaseOnly bool) (*models.ActivityFeedResponse, error) {
	var projects []models.ActivityProject
	var err error

	if showcaseOnly {
		projects, err = s.GetShowcaseProjects(ctx)
	} else {
		projects, err = s.GetAllProjects(ctx)
	}
	if err != nil {
		return nil, err
	}

	stats, err := s.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &models.ActivityFeedResponse{
		LastUpdated: stats.LastUpdated.Format(time.RFC3339),
		Projects:    projects,
		Stats:       *stats,
	}, nil
}
