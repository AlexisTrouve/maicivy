package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// GiteaStatsService récupère les statistiques Git depuis l'API Gitea
type GiteaStatsService struct {
	redis    *redis.Client
	baseURL  string
	token    string
	username string
	client   *http.Client
}

// NewGiteaStatsService crée le service — nil si token manquant
func NewGiteaStatsService(redisClient *redis.Client, baseURL, token, username string) *GiteaStatsService {
	if token == "" || baseURL == "" {
		return nil
	}
	if username == "" {
		username = "StillHammer"
	}
	return &GiteaStatsService{
		redis:    redisClient,
		baseURL:  baseURL,
		token:    token,
		username: username,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Types réponse API ---

type DayStat struct {
	Date      string `json:"date"`
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type RepoStat struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updatedAt"`
}

type GitStatsResponse struct {
	Daily        []DayStat  `json:"daily"`
	Repos        []RepoStat `json:"repos"`
	TotalCommits int        `json:"totalCommits"`
	TotalAdded   int        `json:"totalAdded"`
	TotalDeleted int        `json:"totalDeleted"`
	ActiveRepos  int        `json:"activeRepos"`
	Period       string     `json:"period"`
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
	Stats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
}

// repoResult agrège les résultats d'un repo (pour le parallélisme)
type repoResult struct {
	repo    RepoStat
	commits []giteaCommit
}

// GetStats retourne les stats Git agrégées, avec cache Redis (2h)
func (s *GiteaStatsService) GetStats(ctx context.Context) (*GitStatsResponse, error) {
	cacheKey := "gitstats:v2"

	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var resp GitStatsResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return &resp, nil
			}
		}
	}

	// Context avec timeout de 2 minutes pour le fetch complet
	fetchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := s.fetchAllStats(fetchCtx)
	if err != nil {
		return nil, err
	}

	if s.redis != nil {
		if data, err := json.Marshal(resp); err == nil {
			s.redis.Set(ctx, cacheKey, data, 2*time.Hour)
		}
	}

	return resp, nil
}

// fetchAllStats agrège les stats depuis tous les repos en parallèle
// Utilise /stats/contributors (1 appel/repo) au lieu de fetcher chaque commit detail
func (s *GiteaStatsService) fetchAllStats(ctx context.Context) (*GitStatsResponse, error) {
	repos, err := s.listRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	log.Info().Int("repos", len(repos)).Msg("gitea stats: fetching contributor stats")

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	// Fetch en parallèle (max 5 goroutines)
	sem := make(chan struct{}, 5)
	results := make([]repoResult, len(repos))
	var wg sync.WaitGroup

	for i, repo := range repos {
		if repo.Empty || repo.Fork {
			continue
		}
		wg.Add(1)
		go func(idx int, r giteaRepo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := repoResult{
				repo: RepoStat{
					Name:        r.Name,
					Description: r.Description,
					Language:    r.Language,
					Stars:       r.Stars,
					UpdatedAt:   r.UpdatedAt.Format("2006-01-02"),
				},
			}

			// Commits avec stats (additions/deletions) via ?stat=true
			commits, err := s.listCommits(ctx, r.Name, sixMonthsAgo)
			if err != nil {
				log.Debug().Err(err).Str("repo", r.Name).Msg("skip commits")
			} else {
				res.commits = commits
			}

			results[idx] = res
		}(i, repo)
	}
	wg.Wait()

	// Agrégation — additions/deletions directement depuis chaque commit (stat=true)
	dailyMap := make(map[string]*DayStat)
	var repoStats []RepoStat
	activeRepos := 0

	for _, res := range results {
		if res.repo.Name == "" {
			continue // empty/fork skipped
		}
		repoStats = append(repoStats, res.repo)

		if len(res.commits) > 0 {
			activeRepos++
		}

		for _, c := range res.commits {
			// Exclure les commits massifs (imports, vendoring, migrations)
			if c.Stats.Additions > 10000 {
				continue
			}
			date := c.Commit.Author.Date.Format("2006-01-02")
			if _, ok := dailyMap[date]; !ok {
				dailyMap[date] = &DayStat{Date: date}
			}
			dailyMap[date].Commits++
			dailyMap[date].Additions += c.Stats.Additions
			dailyMap[date].Deletions += c.Stats.Deletions
		}
	}

	// Convertir en slice triée
	daily := make([]DayStat, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, *d)
	}
	sort.Slice(daily, func(i, j int) bool {
		return daily[i].Date < daily[j].Date
	})

	totalCommits, totalAdded, totalDeleted := 0, 0, 0
	for _, d := range daily {
		totalCommits += d.Commits
		totalAdded += d.Additions
		totalDeleted += d.Deletions
	}

	log.Info().
		Int("commits", totalCommits).
		Int("added", totalAdded).
		Int("deleted", totalDeleted).
		Int("repos", len(repoStats)).
		Int("active", activeRepos).
		Msg("gitea stats: done")

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
		path := fmt.Sprintf("/repos/%s/%s/commits?sha=main&since=%s&stat=true&page=%d&limit=50",
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

