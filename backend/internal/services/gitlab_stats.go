package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// GitLabStatsService — commits d'UN projet GitLab, filtrés par auteur, agrégés en DayStat pour être
// MERGÉS dans les gitstats (Gitea + GitLab).
//
// QUOI : récupère les commits du projet (with_stats), ne garde que ceux dont l'author_name matche un
// des auteurs configurés (ex: "alexis" → "Alexis Trouvé"), agrège commits/additions/suppressions par jour.
// POURQUOI : un repo GitLab (ex: REMO) est PARTAGÉ → on ne compte QUE les commits d'Alexis (pas ceux des
// teammates comme "Yulin27"). Unité = commits (additions/suppr par commit), donc compatible avec les
// DayStat de gitea_stats → merge propre. Le repo n'est PAS listé (confidentiel client) : seuls les
// chiffres alimentent les KPI/courbes.
// COMMENT : pagination /repository/commits, filtre author substring, même seuil anti-commit-massif que
// gitea, cache Redis dédié (gitlabstats:v1, refresh 30 min, stale-on-error).
type GitLabStatsService struct {
	redis     *redis.Client
	baseURL   string   // ex: https://gitlab.com
	token     string   // PAT read_api (jamais en source — vient de l'env)
	projectID string   // ID ou path URL-encodé du projet (env, pas hardcodé)
	authors   []string // noms d'auteur en minuscules à matcher en substring
	client    *http.Client

	// Verrou single-flight du rafraîchissement en fond (stale-while-revalidate), cf. bgRefresher.
	refresher bgRefresher
}

// NewGitLabStatsService crée le service — nil si non configuré (token/projet/auteurs manquants).
func NewGitLabStatsService(redisClient *redis.Client, baseURL, token, projectID, authors string) *GitLabStatsService {
	if token == "" || projectID == "" {
		return nil
	}
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	// "alexis,stillhammer" → ["alexis","stillhammer"] (minuscules, vides ignorés).
	var list []string
	for _, a := range strings.Split(authors, ",") {
		if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
			list = append(list, a)
		}
	}
	if len(list) == 0 {
		// Sans auteur configuré on ne saurait pas filtrer un repo partagé → on désactive (sécurité :
		// ne JAMAIS compter les commits des teammates).
		return nil
	}
	return &GitLabStatsService{
		redis:     redisClient,
		baseURL:   strings.TrimRight(baseURL, "/"),
		token:     token,
		projectID: projectID,
		authors:   list,
		client:    &http.Client{Timeout: 20 * time.Second},
	}
}

const (
	gitlabCacheKey         = "gitlabstats:v1"
	gitlabMinFetchInterval = 30 * time.Minute
)

type gitlabCommit struct {
	ID            string    `json:"id"`
	AuthorName    string    `json:"author_name"`
	CommittedDate time.Time `json:"committed_date"`
	Stats         struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
}

type gitlabStatsCache struct {
	Daily     []DayStat `json:"daily"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// GetDaily retourne les DayStat (commits/add/del par jour) des commits de l'auteur sur 6 mois, cachés.
func (s *GitLabStatsService) GetDaily(ctx context.Context, force bool) ([]DayStat, error) {
	if s.redis == nil {
		return s.fetchDaily(ctx)
	}
	if force {
		s.redis.Del(ctx, gitlabCacheKey)
		return s.fetchAndCache(ctx)
	}
	if data, err := s.redis.Get(ctx, gitlabCacheKey).Result(); err == nil {
		var c gitlabStatsCache
		if json.Unmarshal([]byte(data), &c) == nil {
			if time.Since(c.FetchedAt) < gitlabMinFetchInterval {
				return c.Daily, nil
			}
			// Cache périmé → stale-while-revalidate : on sert le stale tout de suite et on refetch en
			// FOND (single-flight). Les commits GitLab sont mergés dans les courbes gitstats : un trou
			// d'attente ici se répercutait sur toute la page → on ne bloque plus jamais le visiteur.
			staleDaily := c.Daily
			s.refresher.trigger(func(bgCtx context.Context) {
				daily, err := s.fetchDaily(bgCtx)
				if err != nil {
					log.Warn().Err(err).Msg("gitlabstats: background refetch failed (stale kept)")
					return
				}
				s.save(bgCtx, daily)
			})
			return staleDaily, nil
		}
	}
	return s.fetchAndCache(ctx)
}

func (s *GitLabStatsService) fetchAndCache(ctx context.Context) ([]DayStat, error) {
	daily, err := s.fetchDaily(ctx)
	if err != nil {
		return nil, err
	}
	s.save(ctx, daily)
	return daily, nil
}

func (s *GitLabStatsService) save(ctx context.Context, daily []DayStat) {
	if s.redis == nil {
		return
	}
	if data, err := json.Marshal(gitlabStatsCache{Daily: daily, FetchedAt: time.Now()}); err == nil {
		s.redis.Set(ctx, gitlabCacheKey, data, 0) // pas de TTL : refresh sur intervalle (cf. GetDaily)
	}
}

// fetchDaily pagine les commits sur 6 mois, filtre par auteur, agrège par jour.
func (s *GitLabStatsService) fetchDaily(ctx context.Context) ([]DayStat, error) {
	fetchCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	since := time.Now().AddDate(0, -6, 0).Format(time.RFC3339)
	dailyMap := make(map[string]*DayStat)

	for page := 1; page <= 50; page++ { // garde-fou pagination (50*100 = 5000 commits max)
		path := fmt.Sprintf("/api/v4/projects/%s/repository/commits?with_stats=true&per_page=100&since=%s&page=%d",
			url.PathEscape(s.projectID), url.QueryEscape(since), page)
		var commits []gitlabCommit
		if err := s.get(fetchCtx, path, &commits); err != nil {
			return nil, err
		}
		for _, c := range commits {
			if !s.matchesAuthor(c.AuthorName) {
				continue
			}
			// Exclure les commits "tuyauterie" (import/vendoring/migration) — même seuil que gitea.
			if c.Stats.Additions > massiveCommitThreshold || c.Stats.Deletions > massiveCommitThreshold {
				continue
			}
			date := c.CommittedDate.Format("2006-01-02")
			d, ok := dailyMap[date]
			if !ok {
				d = &DayStat{Date: date}
				dailyMap[date] = d
			}
			d.Commits++
			d.Additions += c.Stats.Additions
			d.Deletions += c.Stats.Deletions
		}
		if len(commits) < 100 {
			break
		}
	}

	daily := make([]DayStat, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, *d)
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
	log.Info().Int("days", len(daily)).Msg("gitlabstats: fetch done")
	return daily, nil
}

// matchesAuthor : vrai si le nom d'auteur contient (insensible à la casse) un des auteurs configurés.
func (s *GitLabStatsService) matchesAuthor(name string) bool {
	n := strings.ToLower(name)
	for _, a := range s.authors {
		if strings.Contains(n, a) {
			return true
		}
	}
	return false
}

func (s *GitLabStatsService) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", s.token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab %s: %d — %s", path, resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// MergeGitLabDaily fusionne des DayStat GitLab dans une réponse gitstats (Gitea) : additionne par date,
// ajoute les jours manquants, re-trie et recalcule les totaux. La liste des repos et ActiveRepos restent
// INCHANGÉS — le repo GitLab n'est pas exposé (seuls ses chiffres comptent).
//
// IMPLÉMENTATION : on agrège dans une map de VALEURS (pas de pointeurs dans resp.Daily). POURQUOI :
// mélanger des *DayStat pointant dans resp.Daily ET un append(resp.Daily, …) est un bug — l'append peut
// RÉALLOUER le slice, rendant les pointeurs obsolètes ; les incréments sur les jours partagés écrits
// après la réalloc partent dans l'ancien tableau orphelin et sont perdus (sous-comptage). La map de
// valeurs + reconstruction du slice élimine tout aliasing.
func MergeGitLabDaily(resp *GitStatsResponse, gitlabDaily []DayStat) {
	if resp == nil || len(gitlabDaily) == 0 {
		return
	}
	agg := make(map[string]DayStat, len(resp.Daily)+len(gitlabDaily))
	for _, d := range resp.Daily {
		agg[d.Date] = d
	}
	for _, gd := range gitlabDaily {
		cur := agg[gd.Date] // copie de valeur (ou zéro si jour absent)
		cur.Date = gd.Date
		cur.Commits += gd.Commits
		cur.Additions += gd.Additions
		cur.Deletions += gd.Deletions
		agg[gd.Date] = cur
	}

	merged := make([]DayStat, 0, len(agg))
	for _, d := range agg {
		merged = append(merged, d)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Date < merged[j].Date })
	resp.Daily = merged

	tc, ta, td := 0, 0, 0
	for _, d := range merged {
		tc += d.Commits
		ta += d.Additions
		td += d.Deletions
	}
	resp.TotalCommits, resp.TotalAdded, resp.TotalDeleted = tc, ta, td
}
