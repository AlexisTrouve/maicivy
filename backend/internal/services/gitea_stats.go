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

// gitStatsCache — structure persistée en Redis, contient les données + le timestamp du dernier fetch
type gitStatsCache struct {
	Response  GitStatsResponse `json:"response"`
	FetchedAt time.Time        `json:"fetchedAt"` // dernier fetch réussi
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

const (
	cacheKey       = "gitstats:v3"
	// Intervalle minimum entre deux fetches incrémentaux
	minFetchInterval = 30 * time.Minute
)

// GetStats retourne les stats Git agrégées avec cache incrémental.
// - force=true → supprime le cache et fait un full fetch
// - Si le cache a moins de 30 min → retourne tel quel
// - Si le cache a plus de 30 min → fetch incrémental (seulement les nouveaux commits)
// - Si pas de cache → full fetch
func (s *GiteaStatsService) GetStats(ctx context.Context, force bool) (*GitStatsResponse, error) {
	if s.redis == nil {
		return s.fullFetch(ctx)
	}

	// Force refresh → vider le cache et refaire un full fetch
	if force {
		log.Info().Msg("gitstats: force refresh requested, clearing cache")
		s.redis.Del(ctx, cacheKey)
		return s.fullFetchAndCache(ctx)
	}

	// Charger le cache existant
	cached, err := s.loadCache(ctx)
	if err != nil {
		// Pas de cache → full fetch
		log.Info().Msg("gitstats: no cache, doing full fetch")
		return s.fullFetchAndCache(ctx)
	}

	// Cache assez récent → retourner directement
	if time.Since(cached.FetchedAt) < minFetchInterval {
		return &cached.Response, nil
	}

	// Cache périmé → fetch incrémental depuis le dernier fetch
	log.Info().
		Time("lastFetch", cached.FetchedAt).
		Msg("gitstats: incremental fetch")

	resp, err := s.incrementalFetch(ctx, cached)
	if err != nil {
		// En cas d'erreur, retourner le cache stale plutôt que rien
		log.Warn().Err(err).Msg("gitstats: incremental fetch failed, returning stale cache")
		return &cached.Response, nil
	}

	return resp, nil
}

// fullFetchAndCache fait un full fetch et persiste en cache
func (s *GiteaStatsService) fullFetchAndCache(ctx context.Context) (*GitStatsResponse, error) {
	resp, err := s.fullFetch(ctx)
	if err != nil {
		return nil, err
	}
	s.saveCache(ctx, resp)
	return resp, nil
}

// fullFetch récupère toutes les stats depuis 6 mois (premier appel ou reset)
func (s *GiteaStatsService) fullFetch(ctx context.Context) (*GitStatsResponse, error) {
	fetchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	return s.fetchStats(fetchCtx, sixMonthsAgo)
}

// incrementalFetch ne récupère que les commits depuis le dernier fetch,
// puis merge avec les données cachées
func (s *GiteaStatsService) incrementalFetch(ctx context.Context, cached *gitStatsCache) (*GitStatsResponse, error) {
	fetchCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Fetch seulement depuis le dernier fetch (avec 1h de marge pour les commits en retard)
	since := cached.FetchedAt.Add(-1 * time.Hour)
	newData, err := s.fetchStats(fetchCtx, since)
	if err != nil {
		return nil, err
	}

	// Merger : prendre les daily existants du cache, remplacer/ajouter les jours mis à jour
	dailyMap := make(map[string]*DayStat)

	// Cutoff : virer les données > 6 mois du cache
	sixMonthsAgo := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	for i := range cached.Response.Daily {
		d := &cached.Response.Daily[i]
		if d.Date >= sixMonthsAgo {
			dc := *d
			dailyMap[d.Date] = &dc
		}
	}

	// Écraser avec les données fraîches (les jours fetchés depuis `since`)
	for i := range newData.Daily {
		d := &newData.Daily[i]
		dailyMap[d.Date] = d
	}

	// Reconstruire la slice triée
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

	// Repos : prendre la liste fraîche (elle est rapide à fetcher)
	resp := &GitStatsResponse{
		Daily:        daily,
		Repos:        newData.Repos,
		TotalCommits: totalCommits,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		ActiveRepos:  newData.ActiveRepos,
		Period:       "6months",
	}

	s.saveCache(ctx, resp)

	log.Info().
		Int("newDays", len(newData.Daily)).
		Int("totalDays", len(daily)).
		Msg("gitstats: incremental merge done")

	return resp, nil
}

// fetchStats fait le vrai travail : fetch repos + commits depuis `since`, agrège
func (s *GiteaStatsService) fetchStats(ctx context.Context, since time.Time) (*GitStatsResponse, error) {
	repos, err := s.listRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	log.Info().Int("repos", len(repos)).Time("since", since).Msg("gitstats: fetching commits")

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

			commits, err := s.listCommits(ctx, r.Name, since)
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
	dailyMap := make(map[string]*DayStat)
	var repoStats []RepoStat
	activeRepos := 0

	for _, res := range results {
		if res.repo.Name == "" {
			continue
		}
		repoStats = append(repoStats, res.repo)

		if len(res.commits) > 0 {
			activeRepos++
		}

		for _, c := range res.commits {
			// Exclure les commits massifs (imports, vendoring, migrations)
			if c.Stats.Additions > 50000 {
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
		Msg("gitstats: fetch done")

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

// --- Cache Redis ---

func (s *GiteaStatsService) loadCache(ctx context.Context) (*gitStatsCache, error) {
	data, err := s.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}
	var cached gitStatsCache
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func (s *GiteaStatsService) saveCache(ctx context.Context, resp *GitStatsResponse) {
	cached := gitStatsCache{
		Response:  *resp,
		FetchedAt: time.Now(),
	}
	if data, err := json.Marshal(cached); err == nil {
		// Pas de TTL — le cache persiste et se met à jour incrémentalement
		s.redis.Set(ctx, cacheKey, data, 0)
	}
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
