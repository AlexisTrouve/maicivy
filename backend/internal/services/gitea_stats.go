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
}

// Gitea /stats/contributors retourne un tableau de contributeurs,
// chaque contributeur a un tableau de "weeks" avec timestamp + additions/deletions/commits
type giteaContributor struct {
	Name  string            `json:"name"`
	Weeks []giteaWeeklyData `json:"weeks"`
}

type giteaWeeklyData struct {
	Week      int64 `json:"week"`      // unix timestamp du lundi de la semaine
	Additions int   `json:"additions"`
	Deletions int   `json:"deletions"`
	Commits   int   `json:"commits"`
}

// repoResult agrège les résultats d'un repo (pour le parallélisme)
type repoResult struct {
	repo    RepoStat
	weeks   []giteaWeeklyData
	commits []giteaCommit
	err     error
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

			// 1) Contributor stats pour additions/deletions par semaine
			weeks, err := s.getContributorWeeks(ctx, r.Name)
			if err != nil {
				log.Debug().Err(err).Str("repo", r.Name).Msg("skip contributor stats")
			} else {
				res.weeks = weeks
			}

			// 2) Commits pour le day-by-day commit count
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

	// Agrégation
	weeklyMap := make(map[string]*DayStat) // key = "2026-03-10" (lundi de la semaine)
	dailyCommits := make(map[string]int)   // key = "2026-03-15" (date exacte du commit)
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

		// Commits par jour
		for _, c := range res.commits {
			date := c.Commit.Author.Date.Format("2006-01-02")
			dailyCommits[date]++
		}

		// Additions/deletions par semaine (depuis contributor stats)
		for _, w := range res.weeks {
			weekTime := time.Unix(w.Week, 0)
			if weekTime.Before(sixMonthsAgo) {
				continue
			}
			weekKey := weekTime.Format("2006-01-02")
			if _, ok := weeklyMap[weekKey]; !ok {
				weeklyMap[weekKey] = &DayStat{Date: weekKey}
			}
			weeklyMap[weekKey].Additions += w.Additions
			weeklyMap[weekKey].Deletions += w.Deletions
		}
	}

	// Fusionner : pour chaque jour avec des commits, ajouter les additions/deletions
	// de la semaine correspondante (répartis uniformément)
	dailyMap := make(map[string]*DayStat)

	// D'abord, ajouter tous les jours avec des commits
	for date, count := range dailyCommits {
		dailyMap[date] = &DayStat{Date: date, Commits: count}
	}

	// Ensuite, répartir les additions/deletions par semaine sur les jours de commits de cette semaine
	for weekKey, ws := range weeklyMap {
		weekStart, _ := time.Parse("2006-01-02", weekKey)
		weekEnd := weekStart.AddDate(0, 0, 7)

		// Trouver les jours de commits dans cette semaine
		var daysInWeek []string
		for date := range dailyCommits {
			d, _ := time.Parse("2006-01-02", date)
			if !d.Before(weekStart) && d.Before(weekEnd) {
				daysInWeek = append(daysInWeek, date)
			}
		}

		if len(daysInWeek) == 0 {
			// Pas de commits cette semaine mais des additions ? Mettre sur le lundi
			if ws.Additions > 0 || ws.Deletions > 0 {
				if _, ok := dailyMap[weekKey]; !ok {
					dailyMap[weekKey] = &DayStat{Date: weekKey}
				}
				dailyMap[weekKey].Additions += ws.Additions
				dailyMap[weekKey].Deletions += ws.Deletions
			}
			continue
		}

		// Répartir uniformément
		addPerDay := ws.Additions / len(daysInWeek)
		delPerDay := ws.Deletions / len(daysInWeek)
		for _, date := range daysInWeek {
			dailyMap[date].Additions += addPerDay
			dailyMap[date].Deletions += delPerDay
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

// getContributorWeeks retourne les weekly stats agrégés de tous les contributeurs d'un repo
func (s *GiteaStatsService) getContributorWeeks(ctx context.Context, repoName string) ([]giteaWeeklyData, error) {
	path := fmt.Sprintf("/repos/%s/%s/stats/contributors", s.username, repoName)
	data, err := s.doGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var contributors []giteaContributor
	if err := json.Unmarshal(data, &contributors); err != nil {
		return nil, fmt.Errorf("parse contributors %s: %w", repoName, err)
	}

	// Agréger toutes les semaines de tous les contributeurs
	weekMap := make(map[int64]*giteaWeeklyData)
	for _, c := range contributors {
		for _, w := range c.Weeks {
			if existing, ok := weekMap[w.Week]; ok {
				existing.Additions += w.Additions
				existing.Deletions += w.Deletions
				existing.Commits += w.Commits
			} else {
				wCopy := w
				weekMap[w.Week] = &wCopy
			}
		}
	}

	weeks := make([]giteaWeeklyData, 0, len(weekMap))
	for _, w := range weekMap {
		weeks = append(weeks, *w)
	}
	return weeks, nil
}
