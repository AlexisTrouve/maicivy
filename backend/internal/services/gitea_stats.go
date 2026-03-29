package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// GiteaStatsService récupère les statistiques Git depuis l'API Gitea
type GiteaStatsService struct {
	redis    *redis.Client
	baseURL  string // ex: https://git.etheryale.com
	token    string // token read-only Gitea
	username string // owner des repos
	client   *http.Client
}

// NewGiteaStatsService crée le service — nil si token manquant
func NewGiteaStatsService(redisClient *redis.Client, baseURL, token, username string) *GiteaStatsService {
	if token == "" || baseURL == "" {
		return nil
	}
	if username == "" {
		username = "music"
	}
	return &GiteaStatsService{
		redis:    redisClient,
		baseURL:  baseURL,
		token:    token,
		username: username,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// --- Types réponse API ---

// DayStat représente les stats d'un jour
type DayStat struct {
	Date      string `json:"date"`      // "2026-03-15"
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// RepoStat résumé d'un repo
type RepoStat struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updatedAt"`
}

// GitStatsResponse réponse complète pour le frontend
type GitStatsResponse struct {
	Daily        []DayStat  `json:"daily"`        // stats jour par jour (6 mois)
	Repos        []RepoStat `json:"repos"`        // liste des repos
	TotalCommits int        `json:"totalCommits"`
	TotalAdded   int        `json:"totalAdded"`
	TotalDeleted int        `json:"totalDeleted"`
	ActiveRepos  int        `json:"activeRepos"`  // repos avec commits dans les 6 mois
	Period       string     `json:"period"`       // "6months"
}

// --- Types Gitea API ---

type giteaRepo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	Stars       int       `json:"stars_count"`
	UpdatedAt   time.Time `json:"updated_at"`
	Empty       bool      `json:"empty"`
	Fork        bool      `json:"fork"`
}

type giteaCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type giteaCommitDetail struct {
	Stats struct {
		Total     int `json:"total"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
}

// GetStats retourne les stats Git agrégées, avec cache Redis (2h TTL)
func (s *GiteaStatsService) GetStats(ctx context.Context) (*GitStatsResponse, error) {
	cacheKey := "gitstats:v1"

	// Check cache
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var resp GitStatsResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return &resp, nil
			}
		}
	}

	// Fetch depuis Gitea
	resp, err := s.fetchAllStats(ctx)
	if err != nil {
		return nil, err
	}

	// Cache 2h
	if s.redis != nil {
		if data, err := json.Marshal(resp); err == nil {
			s.redis.Set(ctx, cacheKey, data, 2*time.Hour)
		}
	}

	return resp, nil
}

// fetchAllStats agrège commits + lignes depuis tous les repos
func (s *GiteaStatsService) fetchAllStats(ctx context.Context) (*GitStatsResponse, error) {
	repos, err := s.listRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	// Map date -> stats agrégées
	dailyMap := make(map[string]*DayStat)
	var repoStats []RepoStat
	activeRepos := 0

	for _, repo := range repos {
		if repo.Empty || repo.Fork {
			continue
		}

		repoStats = append(repoStats, RepoStat{
			Name:        repo.Name,
			Description: repo.Description,
			Language:    repo.Language,
			Stars:       repo.Stars,
			UpdatedAt:   repo.UpdatedAt.Format("2006-01-02"),
		})

		// Fetch commits des 6 derniers mois pour ce repo
		commits, err := s.listCommits(ctx, repo.Name, sixMonthsAgo)
		if err != nil {
			log.Warn().Err(err).Str("repo", repo.Name).Msg("skip repo commits")
			continue
		}

		if len(commits) > 0 {
			activeRepos++
		}

		for _, commit := range commits {
			date := commit.Commit.Author.Date.Format("2006-01-02")
			if _, ok := dailyMap[date]; !ok {
				dailyMap[date] = &DayStat{Date: date}
			}
			dailyMap[date].Commits++

			// Fetch détail du commit pour les additions/deletions
			detail, err := s.getCommitDetail(ctx, repo.Name, commit.SHA)
			if err != nil {
				continue // skip si erreur, on a quand même le count de commits
			}
			dailyMap[date].Additions += detail.Stats.Additions
			dailyMap[date].Deletions += detail.Stats.Deletions
		}
	}

	// Convertir map en slice triée par date
	daily := make([]DayStat, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, *d)
	}
	sort.Slice(daily, func(i, j int) bool {
		return daily[i].Date < daily[j].Date
	})

	// Totaux
	totalCommits, totalAdded, totalDeleted := 0, 0, 0
	for _, d := range daily {
		totalCommits += d.Commits
		totalAdded += d.Additions
		totalDeleted += d.Deletions
	}

	return &GitStatsResponse{
		Daily:        daily,
		Repos:        repoStats,
		TotalCommits: totalCommits,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		ActiveRepos:  activeRepos,
		Period:       "6months",
	}, nil
}

// --- Appels API Gitea ---

func (s *GiteaStatsService) doGet(ctx context.Context, path string) ([]byte, error) {
	url := s.baseURL + "/api/v1" + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+s.token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitea %s: %d — %s", path, resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (s *GiteaStatsService) listRepos(ctx context.Context) ([]giteaRepo, error) {
	var allRepos []giteaRepo
	page := 1

	for {
		data, err := s.doGet(ctx, fmt.Sprintf("/repos/search?owner=%s&page=%d&limit=50", s.username, page))
		if err != nil {
			return nil, err
		}

		// Gitea wraps repos in {"data": [...], "ok": true}
		var wrapper struct {
			Data []giteaRepo `json:"data"`
			OK   bool        `json:"ok"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("parse repos: %w", err)
		}

		allRepos = append(allRepos, wrapper.Data...)
		if len(wrapper.Data) < 50 {
			break
		}
		page++
	}

	return allRepos, nil
}

func (s *GiteaStatsService) listCommits(ctx context.Context, repoName string, since time.Time) ([]giteaCommit, error) {
	var allCommits []giteaCommit
	page := 1

	for {
		path := fmt.Sprintf("/repos/%s/%s/commits?since=%s&page=%d&limit=50",
			s.username, repoName, since.Format(time.RFC3339), page)

		data, err := s.doGet(ctx, path)
		if err != nil {
			return nil, err
		}

		var commits []giteaCommit
		if err := json.Unmarshal(data, &commits); err != nil {
			return nil, fmt.Errorf("parse commits %s: %w", repoName, err)
		}

		allCommits = append(allCommits, commits...)
		if len(commits) < 50 {
			break
		}
		page++
	}

	return allCommits, nil
}

func (s *GiteaStatsService) getCommitDetail(ctx context.Context, repoName, sha string) (*giteaCommitDetail, error) {
	path := fmt.Sprintf("/repos/%s/%s/git/commits/%s", s.username, repoName, sha)
	data, err := s.doGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var detail giteaCommitDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}
