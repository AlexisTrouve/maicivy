package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	// Verrous single-flight pour le rafraîchissement en fond (stale-while-revalidate). Un par cache :
	// les stats (gitstats:v6) et les langages (gitlang:v1) ont des intervalles différents et se
	// rafraîchissent indépendamment, donc deux verrous distincts.
	statsRefresher bgRefresher
	langRefresher  bgRefresher
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
	UpdatedAt   string `json:"updatedAt"` // dernière MAJ du repo (push, tag, settings…) — PAS forcément du code
	Commits     int    `json:"commits"`     // commits sur 6 mois — RECALCULÉ depuis l'union SHA, jamais accumulé
	Commits30d  int    `json:"commits30d"`  // commits sur 30 jours glissants → badge "Repos chauds en ce moment"
	// CommitDays = jours calendaires distincts (YYYY-MM-DD, triés) avec ≥1 commit sur la branche par
	// défaut, fenêtre 6 mois. SOURCE UNIQUE du scoring vedette : âge = now-CommitDays[0],
	// récence = now-CommitDays[len-1], régularité = len(CommitDays). Stocké comme ensemble pour que
	// le merge incrémental soit une UNION (pas une addition qui gonflerait) + rolloff des >6 mois.
	CommitDays []string `json:"commitDays"`
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
	// RepoCommits : par repo, ensemble SHA→jour (YYYY-MM-DD) des commits sur la fenêtre 6 mois. SOURCE
	// UNIQUE des compteurs par-repo (Commits + Commits30d), recalculés par UNION dédupliquée — JAMAIS
	// accumulés (corrige le bug `r.Commits += c.Commits` qui empilait sans borne). Stocké HORS Response
	// (cache only) pour ne pas gonfler le JSON public. Merge incrémental = union des SHA + rolloff 6 mois,
	// exactement comme CommitDays mais clé par SHA (préserve le COMPTE, pas seulement les jours distincts).
	RepoCommits map[string]map[string]string `json:"repoCommits,omitempty"`
}

// --- Types Gitea API ---

type giteaRepo struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Language      string    `json:"language"`
	Stars         int       `json:"stars_count"`
	UpdatedAt     time.Time `json:"updated_at"`
	Empty         bool      `json:"empty"`
	Fork          bool      `json:"fork"`
	DefaultBranch string    `json:"default_branch"` // main, master, ou autre — varie selon le repo
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
	// v7 : compteur de commits PAR REPO recalculé depuis un ensemble SHA→jour (union dédupliquée +
	// rolloff 6 mois) au lieu de l'accumulateur cassé `r.Commits += c.Commits` (empilait à chaque fetch
	// de 30 min → blender-mcp à 9300 commits pour 0 jour d'activité). Ajout du champ Commits30d (badge
	// "Repos chauds"). Bump de clé = full fetch propre au déploiement pour peupler RepoCommits.
	// v6 : filtre des commits massifs rendu SYMÉTRIQUE (suppressions aussi, plus seulement additions).
	// La migration LFS de ChineseClass (1599 ajouts, 1,19M suppressions) passait l'ancien filtre
	// additions-seul et avait gonflé TotalDeleted dans le cache v5 ; le merge incrémental ne corrige
	// pas une donnée déjà agrégée → bump de clé pour forcer un refetch complet qui réapplique le filtre.
	// v5 : ajout des signaux de scoring vedette (FirstCommit/LastCommit/ActiveDays) → refetch requis.
	// (v4 invalidait v3 et ses commits=0 erronés du bug sha=main, cf. listCommits.)
	cacheKey       = "gitstats:v7"
	// Intervalle minimum entre deux fetches incrémentaux
	minFetchInterval = 30 * time.Minute

	// massiveCommitThreshold — seuil (lignes ajoutées OU supprimées sur un seul commit) au-delà
	// duquel le commit est classé "tuyauterie" (import/vendoring, migration LFS, purge de dossier
	// généré) et exclu des agrégats LOC, pour ne pas fausser les compteurs add/del de la page gitstats.
	massiveCommitThreshold = 50000
)

// GetStats retourne les stats Git agrégées avec cache incrémental.
// - force=true → supprime le cache et fait un full fetch
// - Si le cache a moins de 30 min → retourne tel quel
// - Si le cache a plus de 30 min → fetch incrémental (seulement les nouveaux commits)
// - Si pas de cache → full fetch
func (s *GiteaStatsService) GetStats(ctx context.Context, force bool) (*GitStatsResponse, error) {
	if s.redis == nil {
		resp, _, err := s.fullFetch(ctx)
		return resp, err
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

	// Cache périmé → STALE-WHILE-REVALIDATE : on rend le cache stale IMMÉDIATEMENT (le visiteur n'attend
	// JAMAIS Gitea — c'était le « repull la planète à chaque fois ») et on rafraîchit en FOND. Le verrou
	// single-flight (statsRefresher) garantit qu'un seul refetch incrémental tourne, même sous trafic.
	// Le ctx passé à incrementalFetch est détaché (Background) par bgRefresher : la requête a déjà rendu
	// le stale, son ctx est mort, mais saveCache doit aboutir pour peupler le cache frais.
	log.Info().
		Time("lastFetch", cached.FetchedAt).
		Msg("gitstats: serving stale, refreshing in background")

	s.statsRefresher.trigger(func(bgCtx context.Context) {
		if _, err := s.incrementalFetch(bgCtx, cached); err != nil {
			log.Warn().Err(err).Msg("gitstats: background incremental fetch failed (stale cache kept)")
		}
	})

	return &cached.Response, nil
}

// fullFetchAndCache fait un full fetch et persiste en cache
func (s *GiteaStatsService) fullFetchAndCache(ctx context.Context) (*GitStatsResponse, error) {
	resp, repoCommits, err := s.fullFetch(ctx)
	if err != nil {
		return nil, err
	}
	s.saveCache(ctx, resp, repoCommits)
	return resp, nil
}

// fullFetch récupère toutes les stats depuis 6 mois (premier appel ou reset). Retourne aussi l'ensemble
// SHA→jour par repo (repoCommits), persisté en cache pour le recalcul dédupliqué des compteurs.
func (s *GiteaStatsService) fullFetch(ctx context.Context) (*GitStatsResponse, map[string]map[string]string, error) {
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
	newData, freshRepoCommits, err := s.fetchStats(fetchCtx, since)
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

	// Repos : liste fraîche (métadonnée à jour) MAIS on fusionne l'activité avec le cache.
	// - Commits / Commits30d : RECALCULÉS depuis l'union dédupliquée des SHA (cache ∪ frais, rolloff
	//   6 mois). Remplace l'ancien `r.Commits += c.Commits` qui empilait sans borne à chaque fetch de
	//   30 min (recouvrement 1h re-compté) → blender-mcp affichait 9300 commits pour 0 jour d'activité.
	//   La clé SHA garantit qu'un commit re-fetché n'est compté qu'une fois.
	// - CommitDays : source du scoring vedette → UNION + rolloff 6 mois (cf. mergeCommitDays), sinon
	//   le jour courant serait recompté à chaque fetch incrémental et la régularité gonflerait à tort.
	cutoffDay := time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	cutoff30d := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	cachedByName := make(map[string]RepoStat, len(cached.Response.Repos))
	for _, r := range cached.Response.Repos {
		cachedByName[r.Name] = r
	}
	mergedRepoCommits := make(map[string]map[string]string, len(newData.Repos))
	mergedRepos := make([]RepoStat, len(newData.Repos))
	for i, r := range newData.Repos {
		c := cachedByName[r.Name]
		merged := mergeRepoCommits(cached.RepoCommits[r.Name], freshRepoCommits[r.Name], cutoffDay)
		if len(merged) > 0 {
			mergedRepoCommits[r.Name] = merged
		}
		r.Commits, r.Commits30d = repoCommitCounts(merged, cutoff30d)
		r.CommitDays = mergeCommitDays(c.CommitDays, r.CommitDays, cutoffDay)
		mergedRepos[i] = r
	}

	resp := &GitStatsResponse{
		Daily:        daily,
		Repos:        mergedRepos,
		TotalCommits: totalCommits,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		ActiveRepos:  newData.ActiveRepos,
		Period:       "6months",
	}

	s.saveCache(ctx, resp, mergedRepoCommits)

	log.Info().
		Int("newDays", len(newData.Daily)).
		Int("totalDays", len(daily)).
		Msg("gitstats: incremental merge done")

	return resp, nil
}

// fetchStats fait le vrai travail : fetch repos + commits depuis `since`, agrège. Retourne aussi
// repoCommits (par repo : SHA→jour) → source dédupliquée des compteurs Commits / Commits30d.
func (s *GiteaStatsService) fetchStats(ctx context.Context, since time.Time) (*GitStatsResponse, map[string]map[string]string, error) {
	repos, err := s.listRepos(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list repos: %w", err)
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

			commits, err := s.listCommits(ctx, r.Name, r.DefaultBranch, since)
			if err != nil {
				log.Debug().Err(err).Str("repo", r.Name).Msg("skip commits")
			} else {
				res.commits = commits
				// QUOI : ensemble des jours calendaires distincts avec commit → source du scoring vedette.
				// POURQUOI : âge/récence/régularité s'en dérivent (cf. RepoStat.CommitDays) ; stocké
				// comme ensemble trié pour que le merge incrémental soit une union sans double-comptage.
				// COMMENT : set des dates formatées, puis tri croissant ([0]=plus ancien, [n-1]=plus récent).
				res.repo.CommitDays = distinctSortedDays(commits)
			}

			results[idx] = res
		}(i, repo)
	}
	wg.Wait()

	// Agrégation
	dailyMap := make(map[string]*DayStat)
	var repoStats []RepoStat
	// repoCommits : par repo, ensemble SHA→jour des commits → source dédupliquée des compteurs affichés
	// (Commits 6 mois + Commits30d). Hors GitStatsResponse : persisté dans le cache, pas servi au public.
	repoCommits := make(map[string]map[string]string)
	activeRepos := 0
	cutoff30d := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	for _, res := range results {
		if res.repo.Name == "" {
			continue
		}

		// SHA→jour dédupliqué de ce repo → compteurs (6 mois = len, 30j = sous-ensemble récent).
		shaDay := make(map[string]string, len(res.commits))
		for _, c := range res.commits {
			shaDay[c.SHA] = c.Commit.Author.Date.Format("2006-01-02")
		}
		if len(shaDay) > 0 {
			repoCommits[res.repo.Name] = shaDay
		}
		res.repo.Commits, res.repo.Commits30d = repoCommitCounts(shaDay, cutoff30d)

		repoStats = append(repoStats, res.repo)

		if len(res.commits) > 0 {
			activeRepos++
		}

		for _, c := range res.commits {
			// Exclure les commits "tuyauterie" (imports, vendoring, migrations LFS, purges) — dans les
			// DEUX sens add/del, cf. isMassiveCommit : un vendoring gonfle les additions, une migration
			// LFS / nettoyage gonfle les suppressions.
			if isMassiveCommit(c) {
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
	}, repoCommits, nil
}

// isMassiveCommit signale un commit "tuyauterie" à exclure des agrégats LOC (additions/suppressions).
// POURQUOI symétrique add/del : un import/vendoring gonfle les additions, tandis qu'une migration LFS
// ou une purge de dossier généré gonfle les suppressions. Cas réel verrouillé par TestIsMassiveCommit :
// la migration LFS de ChineseClass (1599 ajouts, 1 187 366 suppressions) passait l'ancien filtre
// additions-seul et faussait TotalDeleted. COMMENT : seuil unique massiveCommitThreshold, comparaison
// stricte (> ) sur chaque sens indépendamment.
func isMassiveCommit(c giteaCommit) bool {
	return c.Stats.Additions > massiveCommitThreshold || c.Stats.Deletions > massiveCommitThreshold
}

// distinctSortedDays extrait les jours calendaires distincts (YYYY-MM-DD) d'une liste de commits,
// triés croissant. Sert de base au scoring vedette (âge/récence/régularité) et au merge incrémental.
func distinctSortedDays(commits []giteaCommit) []string {
	if len(commits) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(commits))
	for _, c := range commits {
		set[c.Commit.Author.Date.Format("2006-01-02")] = struct{}{}
	}
	days := make([]string, 0, len(set))
	for d := range set {
		days = append(days, d)
	}
	sort.Strings(days)
	return days
}

// mergeCommitDays fusionne deux ensembles de jours de commit (cache + frais) en UNION dédupliquée,
// puis élague ceux antérieurs à `cutoff` (rolloff fenêtre 6 mois). Évite le double-comptage qu'une
// simple addition de compteurs provoquerait à chaque fetch incrémental.
func mergeCommitDays(cached, fresh []string, cutoff string) []string {
	set := make(map[string]struct{}, len(cached)+len(fresh))
	for _, d := range cached {
		if d >= cutoff {
			set[d] = struct{}{}
		}
	}
	for _, d := range fresh {
		if d >= cutoff {
			set[d] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	days := make([]string, 0, len(set))
	for d := range set {
		days = append(days, d)
	}
	sort.Strings(days)
	return days
}

// mergeRepoCommits fusionne deux ensembles de commits (SHA→jour) d'un repo — cache + frais — en UNION
// dédupliquée par SHA, puis élague ceux dont le jour est < cutoff (rolloff fenêtre 6 mois). C'est le
// pendant de mergeCommitDays mais clé par SHA (et non par jour) : la clé SHA garantit qu'un commit
// re-fetché (la fenêtre incrémentale a 1h de marge → recouvrement entre deux fetches de 30 min) n'est
// compté QU'UNE fois. C'est ce qui corrige le bug d'empilement de l'ancien `r.Commits += c.Commits`
// (compteur non borné — ex. blender-mcp affiché à 9300 commits pour 0 jour d'activité). Retour : map
// non nil (possiblement vide), prête à être recomptée par repoCommitCounts.
func mergeRepoCommits(cached, fresh map[string]string, cutoff string) map[string]string {
	out := make(map[string]string, len(cached)+len(fresh))
	for sha, day := range cached {
		if day >= cutoff {
			out[sha] = day
		}
	}
	for sha, day := range fresh {
		if day >= cutoff {
			out[sha] = day
		}
	}
	return out
}

// repoCommitCounts dérive les compteurs affichés d'un repo depuis son ensemble SHA→jour (déjà borné à
// 6 mois par le merge) : total = nb de commits sur 6 mois, last30 = commits dont le jour >= cutoff30d
// (fenêtre 30 jours glissants — ce qu'affiche le badge "Repos chauds en ce moment"). Recalcul pur depuis
// l'ensemble dédupliqué → jamais accumulé. nil (repo sans commit / fork) → (0, 0).
func repoCommitCounts(shaToDay map[string]string, cutoff30d string) (total, last30 int) {
	for _, day := range shaToDay {
		total++
		if day >= cutoff30d {
			last30++
		}
	}
	return total, last30
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

func (s *GiteaStatsService) saveCache(ctx context.Context, resp *GitStatsResponse, repoCommits map[string]map[string]string) {
	cached := gitStatsCache{
		Response:    *resp,
		FetchedAt:   time.Now(),
		RepoCommits: repoCommits,
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

func (s *GiteaStatsService) listCommits(ctx context.Context, repoName, branch string, since time.Time) ([]giteaCommit, error) {
	var allCommits []giteaCommit
	page := 1

	// QUOI : construit le paramètre de branche (sha) pour l'API commits.
	// POURQUOI : la moitié des repos StillHammer ont `master` comme branche par défaut
	// (GroveEngine, grimorium…), l'autre `main` (maicivy, Aurelm…). Coder `sha=main` en dur
	// vidait les repos `master` (erreur branche absente → commits=0 → exclus du scoring vedette) ;
	// et OMETTRE `sha` ne marche pas non plus car Gitea retombe sur un `master` implicite et
	// casse les repos `main` (500 "refs/heads/master object does not exist").
	// COMMENT : on passe EXPLICITEMENT la branche par défaut de chaque repo (fournie par
	// l'API repo-list). Fallback : si vide, on omet `sha` et on laisse Gitea décider.
	shaParam := ""
	if branch != "" {
		shaParam = "sha=" + url.QueryEscape(branch) + "&"
	}

	for {
		path := fmt.Sprintf("/repos/%s/%s/commits?%ssince=%s&stat=true&page=%d&limit=50",
			s.username, repoName, shaParam, since.Format(time.RFC3339), page)

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
