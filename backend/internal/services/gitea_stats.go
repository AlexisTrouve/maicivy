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
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
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

	// fetches — single-flight des fetches Gitea SYNCHRONES ET de fond, par cache (clés fetchKeyStats /
	// fetchKeyLang). QUOI : au plus UN fetch Gitea par cache à un instant donné ; tout appelant qui
	// arrive pendant qu'il tourne attend ce fetch et en partage le résultat au lieu d'en lancer un.
	// POURQUOI : bgRefresher ne couvrait que le refresh de FOND. Le chemin « pas de cache » et le
	// force=true faisaient chacun un fetch complet synchrone, sans verrou. Le 2026-10-09 à 15:35 UTC,
	// un clic sur Refresh a vidé le cache : 30 fetches complets de 6 mois sont partis en parallèle
	// (heartbeats des métriques démo, CV, page gitstats), Gitea s'est effondré et a affamé maiProFiles
	// — blog et portfolio en 502 pendant ~15 minutes.
	fetches singleflight.Group

	// exclus — noms de dépôts (en minuscules) qui n'existent pas pour ce service, cf. SetExcludedRepos.
	exclus map[string]bool
}

// SetExcludedRepos écarte des dépôts de TOUT ce que sert le service (stats et langages).
// QUOI : liste de noms séparés par des virgules (GITEA_STATS_EXCLUDE), comparés sans casse.
// POURQUOI : la page /gitstats est publique et liste chaque dépôt du compte, privés compris. Certains
// ne doivent apparaître nulle part. Leur nom vient de l'env et JAMAIS du code : maicivy est miroité en
// public sur GitHub, l'écrire ici le publierait.
// COMMENT : appliqué dans listRepos, l'unique source des repos des deux caches. Un dépôt exclu sort
// des stats au fetch suivant (le merge incrémental repart de la liste de repos courante).
func (s *GiteaStatsService) SetExcludedRepos(liste string) {
	s.exclus = depotsExclus(liste)
}

// depotsExclus — GITEA_STATS_EXCLUDE en ensemble de noms en minuscules. Partagé par tout ce qui lit
// les dépôts Gitea (stats, langages, icônes des projets) : une seule règle d'exclusion.
func depotsExclus(liste string) map[string]bool {
	exclus := map[string]bool{}
	for _, nom := range strings.Split(liste, ",") {
		if nom = strings.ToLower(strings.TrimSpace(nom)); nom != "" {
			exclus[nom] = true
		}
	}
	return exclus
}

// Clés du single-flight : une par cache, pour que stats et langages restent indépendants.
const (
	fetchKeyStats = "gitstats"
	fetchKeyLang  = "gitlang"
)

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

// commitStat — un commit réduit à ce dont les agrégats ont besoin : son jour et ses lignes.
// QUOI : la valeur associée à un SHA dans l'ensemble par repo (cf. gitStatsCache.RepoCommits).
// POURQUOI : SOURCE UNIQUE de tous les compteurs servis. Avant, la série journalière était agrégée
// séparément puis mergée jour par jour, pendant que les compteurs par repo se dérivaient des SHA :
// deux chemins, deux résultats (8949 vs 8026 en production le 2026-08-11). En stockant les lignes
// À CÔTÉ du SHA, la série journalière se dérive du MÊME ensemble → l'écart devient impossible par
// construction, plus seulement improbable.
// COMMENT : clés JSON courtes (d/a/r) — l'ensemble porte un enregistrement par commit sur 6 mois
// (~9000 en prod), chaque octet de nom de champ est payé 9000 fois dans Redis.
type commitStat struct {
	Day       string `json:"d"` // YYYY-MM-DD (date auteur)
	Additions int    `json:"a"`
	Deletions int    `json:"r"`
	// SansLignes : commit CONNU (il compte partout) dont les lignes n'ont pas encore été lues.
	// POURQUOI : compter et mesurer sont deux coûts très différents côté Gitea (liste sans stats
	// ~0,08 s la page, avec stats ~1,5 s — mesuré le 2026-10-11). Le comptage est complet à chaque
	// passage ; les lignes se complètent ensuite, page par page, et ce qui n'a pas pu l'être est
	// repris au passage suivant (cf. completerLignes). Champ inversé (« sans ») pour que les
	// entrées déjà en cache, toutes mesurées, restent valides sans migration.
	SansLignes bool `json:"n,omitempty"`
}

type RepoStat struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updatedAt"`  // dernière MAJ du repo (push, tag, settings…) — PAS forcément du code
	Commits     int    `json:"commits"`    // commits sur 6 mois — RECALCULÉ depuis l'union SHA, jamais accumulé
	Commits30d  int    `json:"commits30d"` // commits sur 30 jours glissants → badge "Repos chauds en ce moment"
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
	// GitlabDaily : commits GitLab SEULS par jour (sous-ensemble de Daily, déjà mergé dans le total).
	// POURQUOI : le front superpose une 2e courbe « GitLab » sur le total du chart commits/jour pour
	// visualiser l'apport du repo client partagé. COMMENT : injecté par le handler (h.gitlab.GetDaily),
	// PAS pendant fetchStats → n'est JAMAIS écrit dans le cache Redis gitstats (donc pas de bump de clé).
	// omitempty : absent si GitLab non configuré / merge échoué → le front dégrade en courbe total seule.
	GitlabDaily []DayStat `json:"gitlabDaily,omitempty"`
}

// gitStatsCache — structure persistée en Redis, contient les données + le timestamp du dernier fetch
type gitStatsCache struct {
	Response  GitStatsResponse `json:"response"`
	FetchedAt time.Time        `json:"fetchedAt"` // dernier fetch réussi
	// RepoCommits : par repo, ensemble SHA→commitStat des commits sur la fenêtre 6 mois. SOURCE UNIQUE
	// de TOUT ce qui est servi — série journalière, compteurs par repo, totaux, repos actifs — tous
	// recalculés depuis cet ensemble par UNION dédupliquée, JAMAIS accumulés ni écrasés. Stocké HORS
	// Response (cache only) pour ne pas gonfler le JSON public. Merge incrémental = union des SHA +
	// rolloff 6 mois : la clé SHA préserve le COMPTE (un commit re-fetché ne compte qu'une fois) et
	// l'union garantit qu'une fenêtre partielle ne peut RIEN retrancher à ce qui est déjà connu.
	RepoCommits map[string]map[string]commitStat `json:"repoCommits,omitempty"`
	// Retards : repos dont la LISTE de commits a échoué au dernier passage → date depuis laquelle la
	// relire au prochain. POURQUOI : sans lui, la fenêtre d'un passage raté était perdue pour de bon
	// (le suivant ne relit que depuis son propre fetchedAt − 1h). Effacé dès que la liste réussit.
	Retards map[string]time.Time `json:"retards,omitempty"`
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
	commits map[string]commitStat
	retard  time.Time // non nul : la liste a échoué, à relire depuis cette date
}

const (
	// v8 : SOURCE UNIQUE. Daily, Repos[], les totaux et ActiveRepos se DÉRIVENT tous du même ensemble
	// SHA→commitStat (cf. deriveStats) ; l'ensemble stocke désormais les lignes à côté du jour, d'où le
	// changement de format du cache (SHA→string devient SHA→objet) et donc ce bump. Corrige deux écarts
	// mesurés en production le 2026-08-11 (8949 commits par repo contre 8026 en journalier) :
	//   1. le merge incrémental ÉCRASAIT le jour courant avec une fenêtre partielle (fetchedAt-1h), ce
	//      qui effaçait définitivement les commits antérieurs de ce jour ;
	//   2. isMassiveCommit retirait le commit du COMPTAGE journalier alors qu'il ne doit exclure que ses
	//      LIGNES — l'ensemble par repo, lui, le comptait.
	// Corrige au passage ActiveRepos, qui était repris de la fenêtre incrémentale (un refresh sans
	// nouveau commit affichait 0 repo actif sur le CV).
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
	// v9 (2026-10-11) : comptage séparé de la mesure des lignes (commitStat.SansLignes) + retards. Le
	// cache v8 de prod avait été écrasé par un fetch complet sous charge (la moitié des repos à 0
	// commit, Melodicode à 0 depuis toujours) : un merge ne retire rien, il ne pouvait pas le guérir.
	cacheKey = "gitstats:v9"
	// Intervalle minimum entre deux fetches incrémentaux
	minFetchInterval = 30 * time.Minute

	// forceRefreshWindow — jusqu'où remonte un Refresh (?force=true) quand un cache existe.
	// POURQUOI : Refresh sert à voir les commits récents, pas à reconstruire 6 mois. Le fetch complet
	// (145 repos × git diff côté Gitea) a fait tomber VPS142 le 2026-10-09 ; il est réservé au cas
	// « aucun cache ». 7 jours couvre une semaine de pushs tardifs ou de commits antidatés, et le merge par SHA ne
	// peut qu'ajouter : relire une fenêtre déjà connue ne double rien.
	forceRefreshWindow = 7 * 24 * time.Hour

	// massiveCommitThreshold — seuil (lignes ajoutées OU supprimées sur un seul commit) au-delà
	// duquel le commit est classé "tuyauterie" (import/vendoring, migration LFS, purge de dossier
	// généré) et exclu des agrégats LOC, pour ne pas fausser les compteurs add/del de la page gitstats.
	massiveCommitThreshold = 50000
)

// GetStats retourne les stats Git agrégées avec cache incrémental.
//   - force=true → relit les derniers jours (forceRefreshWindow) et les fusionne ; le cache n'est
//     jamais supprimé (les autres appelants restent servis). Sans cache → full fetch.
//   - Si le cache a moins de 30 min → retourne tel quel
//   - Si le cache a plus de 30 min → fetch incrémental (seulement les nouveaux commits)
//   - Si pas de cache → full fetch
func (s *GiteaStatsService) GetStats(ctx context.Context, force bool) (*GitStatsResponse, error) {
	if s.redis == nil {
		resp, _, _, err := s.fullFetch(ctx)
		return resp, err
	}

	// Charger le cache existant. Jamais de Del, même en force : le Del rendait le cache absent pour
	// tous les autres appelants pendant le fetch, et chacun lançait alors le sien (panne 2026-10-09).
	cached, err := s.loadCache(ctx)

	// Refresh (force) : relire les DERNIERS JOURS et les fusionner, jamais les 6 mois quand un cache
	// existe. since = le plus ancien entre « maintenant − forceRefreshWindow » et « dernier fetch − 1h » :
	// un cache vieux de 10 jours est rattrapé depuis son dernier fetch (pas de trou), un cache récent
	// est quand même relu sur 7 jours. Synchrone (le clic attend un résultat frais) mais partagé
	// (single-flight) avec tout autre fetch en vol. Sans cache → fetch complet, plus bas.
	if force && err == nil {
		since := time.Now().Add(-forceRefreshWindow)
		if last := cached.FetchedAt.Add(-1 * time.Hour); last.Before(since) {
			since = last
		}
		log.Info().Time("since", since).Msg("gitstats: force refresh requested (recent window)")
		v, ferr, _ := s.fetches.Do(fetchKeyStats, func() (interface{}, error) {
			return s.incrementalFetchSince(context.Background(), cached, since)
		})
		if ferr != nil {
			return nil, ferr
		}
		return v.(*GitStatsResponse), nil
	}
	if err != nil {
		// Pas de cache → full fetch (partagé : une rafale d'appelants n'en déclenche qu'un)
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
		// Passe par le même single-flight que les fetches complets : si un fetch complet (force ou
		// cache vide) est déjà en vol, le refresh de fond s'y joint au lieu de taper Gitea en double.
		_, err, _ := s.fetches.Do(fetchKeyStats, func() (interface{}, error) {
			return s.incrementalFetch(bgCtx, cached)
		})
		if err != nil {
			log.Warn().Err(err).Msg("gitstats: background incremental fetch failed (stale cache kept)")
		}
	})

	return &cached.Response, nil
}

// fullFetchAndCache fait un full fetch et persiste en cache — UN SEUL à la fois (single-flight
// `fetches`) : les appelants concurrents attendent ce fetch et reçoivent le même résultat.
// Le fetch et l'écriture cache utilisent un contexte détaché (fullFetch le fait déjà) : le fetch est
// partagé, il ne doit pas mourir avec la requête du premier appelant.
func (s *GiteaStatsService) fullFetchAndCache(ctx context.Context) (*GitStatsResponse, error) {
	v, err, _ := s.fetches.Do(fetchKeyStats, func() (interface{}, error) {
		// Double vérification, SOUS le verrou : l'appelant a constaté « pas de cache » avant d'entrer,
		// mais un fetch complet a pu le poser entre-temps. Sans elle, chaque appel arrivé pendant la
		// sauvegarde relançait 6 mois de fetch (mesuré au déploiement v9 : trois fetches de 2 min
		// enchaînés). Le single-flight ne couvre que les appels SIMULTANÉS, pas les retardataires.
		if cached, err := s.loadCache(context.Background()); err == nil {
			return &cached.Response, nil
		}
		resp, repoCommits, retards, err := s.fullFetch(ctx)
		if err != nil {
			return nil, err
		}
		s.saveCache(context.Background(), resp, repoCommits, retards)
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*GitStatsResponse), nil
}

// cutoffs retourne les deux bornes de fenêtre utilisées partout : 6 mois (périmètre des stats) et
// 30 jours glissants (badge « Repos chauds »). Un seul endroit où « now » est lu, pour que la
// dérivation et le rolloff du merge ne puissent pas travailler sur des bornes différentes.
func cutoffs() (sixMonths, thirtyDays string) {
	now := time.Now()
	return now.AddDate(0, -6, 0).Format("2006-01-02"), now.AddDate(0, 0, -30).Format("2006-01-02")
}

// deriveStats construit TOUTE la réponse publique depuis l'ensemble SHA→commitStat par repo.
//
// QUOI : à partir des métadonnées de repos et de leurs commits (déjà dédupliqués par SHA et bornés à
// 6 mois), produit la série journalière, les compteurs par repo, les totaux et le nombre de repos actifs.
// POURQUOI : c'est LE point qui garantit que ces compteurs disent la même chose. Ils étaient calculés
// par des chemins séparés qui divergeaient en silence ; ici ils sont trois lectures du même ensemble.
// La règle est unique et explicite : **un commit = un SHA distinct**. Il compte dans le journalier,
// dans son repo et dans le total — toujours. Seules ses LIGNES peuvent être écartées (commit de
// tuyauterie, cf. isMassiveLOC), parce que fausser des LOC et fausser un décompte d'activité sont
// deux problèmes différents.
// COMMENT : une seule passe par repo ; le rolloff est appliqué défensivement (le merge l'a déjà fait,
// mais un fetch complet remplit l'ensemble sans passer par le merge).
func deriveStats(repos []RepoStat, commits map[string]map[string]commitStat, cutoff6m, cutoff30d string) *GitStatsResponse {
	dailyMap := make(map[string]*DayStat)
	out := make([]RepoStat, 0, len(repos))
	activeRepos := 0

	for _, r := range repos {
		set := commits[r.Name]
		days := make(map[string]struct{}, len(set))
		r.Commits, r.Commits30d = 0, 0

		for _, c := range set {
			if c.Day < cutoff6m {
				continue // hors fenêtre : ni compté, ni affiché
			}
			// --- comptage : identique pour les trois compteurs ---
			r.Commits++
			if c.Day >= cutoff30d {
				r.Commits30d++
			}
			days[c.Day] = struct{}{}
			d, ok := dailyMap[c.Day]
			if !ok {
				d = &DayStat{Date: c.Day}
				dailyMap[c.Day] = d
			}
			d.Commits++
			// --- lignes : seul endroit où un commit peut être écarté ---
			if !isMassiveLOC(c.Additions, c.Deletions) {
				d.Additions += c.Additions
				d.Deletions += c.Deletions
			}
		}

		r.CommitDays = sortedDays(days)
		if r.Commits > 0 {
			activeRepos++ // actif = a commité dans les 6 mois, PAS dans la dernière fenêtre fetchée
		}
		out = append(out, r)
	}

	daily := make([]DayStat, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, *d)
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })

	totalCommits, totalAdded, totalDeleted := 0, 0, 0
	for _, d := range daily {
		totalCommits += d.Commits
		totalAdded += d.Additions
		totalDeleted += d.Deletions
	}

	return &GitStatsResponse{
		Daily:        daily,
		Repos:        out,
		TotalCommits: totalCommits,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		ActiveRepos:  activeRepos,
		Period:       "6months",
	}
}

// sortedDays convertit un ensemble de jours en slice triée croissant (nil si vide) — forme attendue
// par RepoStat.CommitDays, dont le scoring vedette lit [0] (le plus ancien) et [len-1] (le plus récent).
func sortedDays(set map[string]struct{}) []string {
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

// fullFetch récupère toutes les stats depuis 6 mois (premier appel ou reset). Retourne aussi l'ensemble
// SHA→commitStat par repo, persisté en cache pour le recalcul dédupliqué des compteurs, et les repos
// dont la liste a échoué (à relire au prochain passage).
func (s *GiteaStatsService) fullFetch(ctx context.Context) (*GitStatsResponse, map[string]map[string]commitStat, map[string]time.Time, error) {
	fetchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	depuis := time.Now().AddDate(0, -6, 0)
	repos, commits, retards, err := s.fetchStats(fetchCtx, func(string) time.Time { return depuis }, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	cutoff6m, cutoff30d := cutoffs()
	return deriveStats(repos, commits, cutoff6m, cutoff30d), commits, retards, nil
}

// incrementalFetch ne récupère que les commits depuis le dernier fetch,
// puis merge avec les données cachées
func (s *GiteaStatsService) incrementalFetch(ctx context.Context, cached *gitStatsCache) (*GitStatsResponse, error) {
	// Fetch seulement depuis le dernier fetch (avec 1h de marge pour les commits en retard)
	return s.incrementalFetchSince(ctx, cached, cached.FetchedAt.Add(-1*time.Hour))
}

// depuisPourRepo : d'où relire un repo lors d'un passage incrémental. Le plus ancien de :
//   - la fenêtre du passage (`since`) ;
//   - son retard, si sa liste a échoué au passage précédent ;
//   - le jour du plus ancien commit dont les lignes restent à lire (moins un jour : le filtre `since`
//     de Gitea porte sur la date du committer, toujours postérieure ou égale à celle de l'auteur que
//     l'on stocke — relire un jour de trop ne coûte rien, le merge déduplique par SHA).
func depuisPourRepo(since time.Time, retard time.Time, connus map[string]commitStat) time.Time {
	d := since
	if !retard.IsZero() && retard.Before(d) {
		d = retard
	}
	for _, c := range connus {
		if !c.SansLignes {
			continue
		}
		if jour, err := time.Parse("2006-01-02", c.Day); err == nil {
			if j := jour.Add(-24 * time.Hour); j.Before(d) {
				d = j
			}
		}
	}
	return d
}

// incrementalFetchSince relit les commits depuis `since` et les FUSIONNE au cache (union par SHA).
// Partagé par le refresh de fond (since = dernier fetch − 1h) et le Refresh manuel (derniers jours).
// Un repo en retard ou aux lignes incomplètes est relu plus loin (cf. depuisPourRepo).
func (s *GiteaStatsService) incrementalFetchSince(ctx context.Context, cached *gitStatsCache, since time.Time) (*GitStatsResponse, error) {
	fetchCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	depuis := func(repo string) time.Time {
		return depuisPourRepo(since, cached.Retards[repo], cached.RepoCommits[repo])
	}
	freshRepos, freshCommits, retards, err := s.fetchStats(fetchCtx, depuis, cached.RepoCommits)
	if err != nil {
		return nil, err
	}

	// UNION par SHA, repo par repo : la fenêtre fraîche ne peut qu'AJOUTER, jamais retrancher.
	// C'est LE point qui interdit l'érosion : un jour déjà observé conserve tous ses commits même si
	// la fenêtre étroite (1h30) n'en ramène qu'un seul. L'ancien code écrasait le jour entier avec ce
	// résultat partiel — les commits antérieurs de la journée étaient perdus pour de bon, puisque le
	// lendemain ce jour n'est plus jamais refetché.
	// La liste des repos, elle, vient toujours fraîche et COMPLÈTE (listRepos ne dépend pas de `since`) :
	// un repo absent de cette liste a été supprimé ou renommé côté Gitea, et sort donc légitimement.
	cutoff6m, cutoff30d := cutoffs()
	merged := make(map[string]map[string]commitStat, len(freshRepos))
	for _, r := range freshRepos {
		if set := mergeRepoCommits(cached.RepoCommits[r.Name], freshCommits[r.Name], cutoff6m); len(set) > 0 {
			merged[r.Name] = set
		}
	}

	resp := deriveStats(freshRepos, merged, cutoff6m, cutoff30d)
	s.saveCache(ctx, resp, merged, retards)

	log.Info().
		Int("repos", len(freshRepos)).
		Int("totalDays", len(resp.Daily)).
		Int("commits", resp.TotalCommits).
		Int("retards", len(retards)).
		Msg("gitstats: incremental merge done")

	return resp, nil
}

// fetchStats récupère la MATIÈRE PREMIÈRE : la liste des repos (métadonnées à jour) et, par repo,
// l'ensemble SHA→commitStat de ses commits depuis depuis(repo). Il n'agrège rien.
// POURQUOI : tant que ce niveau produisait aussi des compteurs, il existait deux chemins de calcul
// (ici pour le fetch complet, dans le merge pour l'incrémental) — donc deux résultats possibles.
// Tout le comptage est désormais dans deriveStats, appelé par les deux chemins.
//
// COMMENT, par repo, en deux temps (refonte du 2026-10-11) :
//  1. COMPTER : liste des commits SANS stats (~0,08 s la page). Complète, ou le repo part en retard.
//  2. MESURER : lignes ajoutées/supprimées, lues uniquement pour les commits qui ne les ont pas
//     encore (connus = cache), page par page (cf. completerLignes). Une mesure inachevée laisse des
//     commits SansLignes, repris au passage suivant ; elle ne retire JAMAIS un commit du compte.
//
// Mesuré avant : tout passait par la liste AVEC stats (~1,5 s la page) sous un délai global de
// 1-2 min partagé par 145 repos. Melodicode (~70 pages, ~110 s à lui seul) n'aboutissait jamais et
// valait 0 commit ; sous charge (14:15 UTC, 1 690 délais dépassés) c'est la moitié des repos qui
// tombait à 0 — et un repo en échec était enregistré VIDE.
//
// retards : repos dont la liste a échoué → depuis(repo), à relire au prochain passage.
func (s *GiteaStatsService) fetchStats(ctx context.Context, depuis func(repo string) time.Time, connus map[string]map[string]commitStat) ([]RepoStat, map[string]map[string]commitStat, map[string]time.Time, error) {
	repos, err := s.listRepos(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list repos: %w", err)
	}

	log.Info().Int("repos", len(repos)).Msg("gitstats: fetching commits")

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
			since := depuis(r.Name)

			// 1. Compter.
			pages, err := s.listCommits(ctx, r.Name, r.DefaultBranch, since, false, 0)
			if err != nil {
				log.Debug().Err(err).Str("repo", r.Name).Msg("liste des commits en échec — repo relu au prochain passage")
				res.retard = since
				results[idx] = res
				return
			}
			set := make(map[string]commitStat)
			for _, page := range pages {
				for _, c := range page {
					cs := commitStat{Day: c.Commit.Author.Date.Format("2006-01-02"), SansLignes: true}
					if k, ok := connus[r.Name][c.SHA]; ok && !k.SansLignes {
						cs = k // déjà mesuré : rien à relire
					}
					set[c.SHA] = cs
				}
			}

			// 2. Mesurer ce qui ne l'est pas encore.
			if err := s.completerLignes(ctx, r.Name, r.DefaultBranch, since, pages, set); err != nil {
				log.Debug().Err(err).Str("repo", r.Name).Msg("lignes incomplètes — reprises au prochain passage")
			}
			res.commits = set
			results[idx] = res
		}(i, repo)
	}
	wg.Wait()

	var repoStats []RepoStat
	repoCommits := make(map[string]map[string]commitStat)
	retards := make(map[string]time.Time)
	fetched, sansLignes := 0, 0

	for _, res := range results {
		if res.repo.Name == "" {
			continue // repo vide/fork : ignoré en amont, la case du tableau est restée à zéro
		}
		if !res.retard.IsZero() {
			retards[res.repo.Name] = res.retard
		}
		if len(res.commits) > 0 {
			repoCommits[res.repo.Name] = res.commits
			fetched += len(res.commits)
			for _, c := range res.commits {
				if c.SansLignes {
					sansLignes++
				}
			}
		}
		repoStats = append(repoStats, res.repo)
	}

	log.Info().
		Int("repos", len(repoStats)).
		Int("commitsInWindow", fetched).
		Int("sansLignes", sansLignes).
		Int("retards", len(retards)).
		Msg("gitstats: fetch done")

	return repoStats, repoCommits, retards, nil
}

// completerLignes lit les lignes ajoutées/supprimées des commits de `set` encore SansLignes.
// COMMENT : la liste avec stats suit le MÊME ordre et la même pagination que la liste sans stats
// (même requête, même `since`, même `limit`). On ne relit donc que les pages qui contiennent un
// commit à mesurer. Si des commits arrivent entre-temps, les pages glissent : un SHA attendu peut
// manquer à sa page — il reste SansLignes et sera repris, il n'est jamais mal attribué (on remplit
// par SHA, pas par position). S'arrête à la première erreur (délai du passage épuisé, typiquement).
func (s *GiteaStatsService) completerLignes(ctx context.Context, repo, branch string, since time.Time, pages [][]giteaCommit, set map[string]commitStat) error {
	for i, page := range pages {
		aMesurer := false
		for _, c := range page {
			if set[c.SHA].SansLignes {
				aMesurer = true
				break
			}
		}
		if !aMesurer {
			continue
		}
		mesurees, err := s.listCommits(ctx, repo, branch, since, true, i+1)
		if err != nil {
			return err
		}
		for _, c := range mesurees[0] {
			if k, ok := set[c.SHA]; ok && k.SansLignes {
				k.Additions, k.Deletions, k.SansLignes = c.Stats.Additions, c.Stats.Deletions, false
				set[c.SHA] = k
			}
		}
	}
	return nil
}

// isMassiveCommit signale un commit "tuyauterie" à exclure des agrégats LOC (additions/suppressions).
// POURQUOI symétrique add/del : un import/vendoring gonfle les additions, tandis qu'une migration LFS
// ou une purge de dossier généré gonfle les suppressions. Cas réel verrouillé par TestIsMassiveCommit :
// la migration LFS de ChineseClass (1599 ajouts, 1 187 366 suppressions) passait l'ancien filtre
// additions-seul et faussait TotalDeleted. COMMENT : seuil unique massiveCommitThreshold, comparaison
// stricte (> ) sur chaque sens indépendamment.
func isMassiveCommit(c giteaCommit) bool {
	return isMassiveLOC(c.Stats.Additions, c.Stats.Deletions)
}

// isMassiveLOC porte la règle elle-même, indépendamment de la forme du commit : c'est elle
// qu'applique deriveStats sur un commitStat, pour qu'il n'existe qu'UN seuil dans le service.
// ⚠️ Cette règle n'écarte que des LIGNES. Un commit de tuyauterie reste un commit et continue de
// compter partout où l'on compte des commits — les confondre était l'une des deux causes de l'écart
// entre la série journalière et les compteurs par repo.
func isMassiveLOC(additions, deletions int) bool {
	return additions > massiveCommitThreshold || deletions > massiveCommitThreshold
}

// mergeRepoCommits fusionne deux ensembles de commits (SHA→jour) d'un repo — cache + frais — en UNION
// dédupliquée par SHA, puis élague ceux dont le jour est < cutoff (rolloff fenêtre 6 mois). C'est le
// pendant du rolloff appliqué par deriveStats, mais clé par SHA : la clé SHA garantit qu'un commit
// re-fetché (la fenêtre incrémentale a 1h de marge → recouvrement entre deux fetches de 30 min) n'est
// compté QU'UNE fois. C'est ce qui corrige le bug d'empilement de l'ancien `r.Commits += c.Commits`
// (compteur non borné — ex. blender-mcp affiché à 9300 commits pour 0 jour d'activité). Retour : map
// non nil (possiblement vide), prête à être recomptée par repoCommitCounts.
func mergeRepoCommits(cached, fresh map[string]commitStat, cutoff string) map[string]commitStat {
	out := make(map[string]commitStat, len(cached)+len(fresh))
	for sha, c := range cached {
		if c.Day >= cutoff {
			out[sha] = c
		}
	}
	for sha, c := range fresh {
		if c.Day < cutoff {
			continue
		}
		// Même SHA re-fetché → ne s'ajoute pas. Une version MESURÉE ne cède jamais la place à une
		// version sans lignes : un passage dont la mesure a échoué ne retranche pas de lignes connues.
		if prev, ok := out[sha]; ok && !prev.SansLignes && c.SansLignes {
			continue
		}
		out[sha] = c
	}
	return out
}

// repoCommitCounts dérive les compteurs affichés d'un repo depuis son ensemble SHA→jour (déjà borné à
// 6 mois par le merge) : total = nb de commits sur 6 mois, last30 = commits dont le jour >= cutoff30d
// (fenêtre 30 jours glissants — ce qu'affiche le badge "Repos chauds en ce moment"). Recalcul pur depuis
// l'ensemble dédupliqué → jamais accumulé. nil (repo sans commit / fork) → (0, 0).
func repoCommitCounts(set map[string]commitStat, cutoff30d string) (total, last30 int) {
	for _, c := range set {
		total++
		if c.Day >= cutoff30d {
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

func (s *GiteaStatsService) saveCache(ctx context.Context, resp *GitStatsResponse, repoCommits map[string]map[string]commitStat, retards map[string]time.Time) {
	cached := gitStatsCache{
		Response:    *resp,
		FetchedAt:   time.Now(),
		RepoCommits: repoCommits,
		Retards:     retards,
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

		for _, r := range wrapper.Data {
			if !s.exclus[strings.ToLower(r.Name)] {
				allRepos = append(allRepos, r)
			}
		}
		// La pagination se juge sur la page BRUTE : une exclusion ne doit pas faire croire à la fin.
		if len(wrapper.Data) < 50 {
			break
		}
		page++
	}

	return allRepos, nil
}

// listCommits lit les commits de la branche depuis `since`, page par page (50 par page, du plus
// récent au plus ancien) et rend les pages telles quelles : leur position sert à completerLignes.
//   - avecLignes : demande les stats (lignes ajoutées/supprimées) — ~20× plus cher.
//   - seulePage > 0 : ne lit QUE cette page ; sinon toutes, jusqu'à une page incomplète.
func (s *GiteaStatsService) listCommits(ctx context.Context, repoName, branch string, since time.Time, avecLignes bool, seulePage int) ([][]giteaCommit, error) {
	var pages [][]giteaCommit
	page := 1
	if seulePage > 0 {
		page = seulePage
	}

	// stat=false : liste ~20× moins chère (Gitea n'exécute pas de diff par commit). files et
	// verification à false dans les deux cas : jamais utilisés ici, et chacun coûte côté serveur.
	statParam := "stat=false"
	if avecLignes {
		statParam = "stat=true"
	}

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
		path := fmt.Sprintf("/repos/%s/%s/commits?%ssince=%s&%s&files=false&verification=false&page=%d&limit=50",
			// QueryEscape : hors UTC, RFC3339 porte un « + » (ex. +08:00) qui, non encodé, se lit
			// comme un espace côté serveur → date illisible. En prod (conteneur UTC) c'est « Z ».
			s.username, repoName, shaParam, url.QueryEscape(since.Format(time.RFC3339)), statParam, page)

		data, err := s.doGet(ctx, path)
		if err != nil {
			return nil, err
		}

		var commits []giteaCommit
		if err := json.Unmarshal(data, &commits); err != nil {
			return nil, fmt.Errorf("parse commits %s: %w", repoName, err)
		}

		pages = append(pages, commits)
		if seulePage > 0 || len(commits) < 50 {
			break
		}
		page++
	}

	return pages, nil
}
