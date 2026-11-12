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

// PartnerGiteaStatsService — commits d'Alexis sur UN dépôt hébergé par un Gitea TIERS (pas celui
// d'Alexis), agrégés en DayStat pour être MERGÉS dans les gitstats, exactement comme le GitLab client.
//
// QUOI : pagine les commits d'une branche du dépôt, ne garde que ceux dont l'EMAIL d'auteur est l'un
// des emails configurés, agrège commits/lignes par jour sur 6 mois.
// POURQUOI : le jeu partenaire vit sur le Gitea de son studio, pas sur git.etheryale.com — ses commits
// étaient donc invisibles des stats (mesuré le 2026-10-24 : 104 commits d'Alexis sur main en 18 jours).
// Le dépôt est PARTAGÉ avec deux autres développeurs : on ne compte QUE les commits d'Alexis.
// Le dépôt n'est PAS listé dans les repos publics : son nom porte l'acronyme du titre du jeu, que le
// projet interdit sur ce qui sort. Seuls les chiffres
// alimentent les KPI/courbes — même politique que GitLab.
// COMMENT :
//   - filtre par EMAIL, insensible à la casse : Alexis y commite en « Alexistrouve.pro@… » (majuscule)
//     sous deux noms différents (« Trouve Alexis », « StillHammer ») — le nom ne discrimine pas, l'email
//     oui, et un filtre sensible à la casse ratait 101 commits sur 104 ;
//   - un commit de MERGE compte comme commit, mais SES LIGNES NE COMPTENT PAS : sa diff (contre le
//     premier parent) recopie celle des commits de la branche fusionnée, déjà comptés un par un sur
//     main — les additionner doublerait les lignes (23 merges sur 104, dont un de 844 lignes) ;
//   - même règle anti-commit-massif que Gitea/GitLab (isMassiveLOC) : le commit compte, ses lignes non ;
//   - dédup par SHA (une page qui se recouvre ne recompte pas) ;
//   - cache Redis dédié, stale-while-revalidate 30 min, single-flight (bgRefresher).
type PartnerGiteaStatsService struct {
	redis   *redis.Client
	baseURL string   // ex: https://git.example.com (sans /api/v1)
	token   string   // jeton read:repository (jamais en source — vient de l'env)
	owner   string   // propriétaire du dépôt
	repo    string   // nom du dépôt
	ref     string   // branche comptée ; vide = branche par défaut du dépôt
	emails  []string // emails d'auteur en minuscules, comparés à l'égalité
	client  *http.Client

	refresher bgRefresher
}

// NewPartnerGiteaStatsService crée le service — nil si non configuré.
// repoPath = "owner/repo". emails = liste séparée par virgules. Sans email, le service est désactivé :
// sur un dépôt partagé, ne pas savoir filtrer = compter le travail des autres (jamais).
func NewPartnerGiteaStatsService(redisClient *redis.Client, baseURL, token, repoPath, ref, emails string) *PartnerGiteaStatsService {
	owner, repo, ok := strings.Cut(strings.Trim(strings.TrimSpace(repoPath), "/"), "/")
	if token == "" || baseURL == "" || !ok || owner == "" || repo == "" {
		return nil
	}
	var list []string
	for _, e := range strings.Split(emails, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			list = append(list, e)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return &PartnerGiteaStatsService{
		redis:   redisClient,
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		owner:   owner,
		repo:    repo,
		ref:     strings.TrimSpace(ref),
		emails:  list,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

const (
	partnerGiteaCacheKey         = "partnergitea:v1"
	partnerGiteaMinFetchInterval = 30 * time.Minute
	partnerGiteaPageSize         = 50 // plafond par défaut de l'API Gitea (MAX_RESPONSE_ITEMS)
)

// partnerCommit : la forme Gitea d'un commit, réduite à ce qui sert (email d'auteur, parents, lignes).
type partnerCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
	Stats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
}

type partnerGiteaCache struct {
	Daily     []DayStat `json:"daily"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// GetDaily retourne les DayStat d'Alexis sur 6 mois, cachés. Même contrat que GitLabStatsService.GetDaily :
// on ne refetch JAMAIS en avant-plan sur un cache périmé (stale servi, refresh en fond) ; force = synchrone.
func (s *PartnerGiteaStatsService) GetDaily(ctx context.Context, force bool) ([]DayStat, error) {
	if s.redis == nil {
		return s.fetchDaily(ctx)
	}
	if force {
		s.redis.Del(ctx, partnerGiteaCacheKey)
		return s.fetchAndCache(ctx)
	}
	if data, err := s.redis.Get(ctx, partnerGiteaCacheKey).Result(); err == nil {
		var c partnerGiteaCache
		if json.Unmarshal([]byte(data), &c) == nil {
			if time.Since(c.FetchedAt) < partnerGiteaMinFetchInterval {
				return c.Daily, nil
			}
			stale := c.Daily
			s.refresher.trigger(func(bgCtx context.Context) {
				daily, err := s.fetchDaily(bgCtx)
				if err != nil {
					log.Warn().Err(err).Msg("partnergitea: background refetch failed (stale kept)")
					return
				}
				s.save(bgCtx, daily)
			})
			return stale, nil
		}
	}
	return s.fetchAndCache(ctx)
}

func (s *PartnerGiteaStatsService) fetchAndCache(ctx context.Context) ([]DayStat, error) {
	daily, err := s.fetchDaily(ctx)
	if err != nil {
		return nil, err
	}
	s.save(ctx, daily)
	return daily, nil
}

func (s *PartnerGiteaStatsService) save(ctx context.Context, daily []DayStat) {
	if s.redis == nil {
		return
	}
	if data, err := json.Marshal(partnerGiteaCache{Daily: daily, FetchedAt: time.Now()}); err == nil {
		s.redis.Set(ctx, partnerGiteaCacheKey, data, 0) // pas de TTL : refresh sur intervalle (cf. GetDaily)
	}
}

// fetchDaily pagine la branche sur 6 mois (stat=true : les lignes alimentent les totaux) et agrège par
// jour les commits dont l'email d'auteur est configuré. Le jour est celui de la date d'auteur telle que
// Gitea la rend (fuseau de l'auteur) — c'est le jour où Alexis a travaillé.
func (s *PartnerGiteaStatsService) fetchDaily(ctx context.Context) ([]DayStat, error) {
	// Contexte détaché borné : la pagination ne doit pas mourir avec la requête HTTP du visiteur.
	fetchCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	since := url.QueryEscape(time.Now().AddDate(0, -6, 0).UTC().Format(time.RFC3339))
	ref := ""
	if s.ref != "" {
		ref = "sha=" + url.QueryEscape(s.ref) + "&"
	}
	dailyMap := make(map[string]DayStat)
	seen := make(map[string]struct{})
	mine := 0

	for page := 1; page <= 100; page++ { // garde-fou : 100×50 = 5000 commits (346 mesurés sur main)
		path := fmt.Sprintf("/api/v1/repos/%s/%s/commits?%ssince=%s&stat=true&files=false&verification=false&limit=%d&page=%d",
			url.PathEscape(s.owner), url.PathEscape(s.repo), ref, since, partnerGiteaPageSize, page)
		var commits []partnerCommit
		if err := s.get(fetchCtx, path, &commits); err != nil {
			return nil, err
		}
		for _, c := range commits {
			if _, dup := seen[c.SHA]; dup {
				continue
			}
			seen[c.SHA] = struct{}{}
			if !s.matchesEmail(c.Commit.Author.Email) {
				continue
			}
			mine++
			date := c.Commit.Author.Date.Format("2006-01-02")
			d := dailyMap[date]
			d.Date = date
			d.Commits++
			// Merge : commit compté, lignes ignorées (déjà comptées via les commits de la branche).
			// Massif : même règle que Gitea/GitLab.
			if len(c.Parents) < 2 && !isMassiveLOC(c.Stats.Additions, c.Stats.Deletions) {
				d.Additions += c.Stats.Additions
				d.Deletions += c.Stats.Deletions
			}
			dailyMap[date] = d
		}
		if len(commits) < partnerGiteaPageSize {
			break
		}
	}

	daily := make([]DayStat, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, d)
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
	log.Info().Int("commits", mine).Int("days", len(daily)).Msg("partnergitea: fetch done")
	return daily, nil
}

// matchesEmail : égalité stricte, insensible à la casse (un email n'est pas une sous-chaîne à chercher).
func (s *PartnerGiteaStatsService) matchesEmail(email string) bool {
	e := strings.ToLower(strings.TrimSpace(email))
	for _, want := range s.emails {
		if e == want {
			return true
		}
	}
	return false
}

func (s *PartnerGiteaStatsService) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+s.token)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		// Le corps est tronqué : une page d'erreur HTML n'a rien à faire en entier dans les logs.
		if len(body) > 300 {
			body = body[:300]
		}
		return fmt.Errorf("partner gitea %d — %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
