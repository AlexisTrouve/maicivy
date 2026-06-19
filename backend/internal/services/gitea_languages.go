package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// --- LOC par langage (alimente la fiche détail d'un skill côté frontend) ---
//
// QUOI : agrège les octets de code par langage sur tous les repos Gitea de l'owner, et en dérive
// une approximation du nombre de lignes (LOC). Sert à la pastille skill cliquable : "~12k lignes".
// POURQUOI : Gitea expose `/repos/{owner}/{repo}/languages` → octets par langage (le même calcul que
// la barre de langages d'un repo). C'est la SEULE source de LOC par langage dont on dispose ; le
// reste (gitstats) n'agrège qu'au global par jour, jamais par langage.
// COMMENT : un fetch par repo (parallélisé), somme des octets par langage normalisé, conversion
// octets→LOC via une moyenne, cache Redis dédié à refresh lent (les langages bougent peu).

// LangStat = octets + LOC approximée pour un langage donné.
type LangStat struct {
	Language string `json:"language"` // clé canonique lowercase (ex: "go", "typescript")
	Bytes    int    `json:"bytes"`    // octets de code (somme sur tous les repos)
	LOC      int    `json:"loc"`      // approximation lignes = Bytes / avgBytesPerLine
}

// LangStatsResponse est la réponse de l'endpoint /api/v1/cv/loc.
type LangStatsResponse struct {
	// Languages : clé = langage normalisé lowercase → stat. Le frontend matche le nom du skill
	// (via sa table d'alias) contre ces clés pour afficher les LOC dans la fiche.
	Languages  map[string]LangStat `json:"languages"`
	TotalLOC   int                 `json:"totalLoc"`
	TotalBytes int                 `json:"totalBytes"`
	Period     string              `json:"period"`
}

// langStatsCache — structure persistée en Redis (données + timestamp du dernier fetch).
type langStatsCache struct {
	Response  LangStatsResponse `json:"response"`
	FetchedAt time.Time         `json:"fetchedAt"`
}

const (
	// langCacheKey — clé Redis dédiée (séparée de gitstats:v6). v1 = premier schéma.
	langCacheKey = "gitlang:v1"

	// langRefreshInterval — intervalle mini entre deux recalculs. Long EXPRÈS : la répartition des
	// langages d'un portfolio bouge en jours/semaines, pas en minutes. Garantit qu'on ne re-pull
	// JAMAIS Gitea à chaque requête (contrainte cache : le hit normal lit Redis, point).
	langRefreshInterval = 6 * time.Hour

	// avgBytesPerLine — diviseur octets→LOC. POURQUOI 38 : moyenne empirique tous langages confondus
	// (indentation + code + accolades), assez juste pour un ordre de grandeur "~12k lignes". On assume
	// l'approximation — la fiche labellise "~" côté UI, on ne prétend pas à l'exactitude.
	avgBytesPerLine = 38
)

// GetLanguageStats retourne les LOC par langage, avec cache Redis à refresh lent.
//   - force=true → vide le cache et refait un full fetch
//   - cache < 6h → retourné tel quel (cas normal, zéro appel Gitea)
//   - cache ≥ 6h → refetch ; en cas d'erreur Gitea, on sert le cache stale plutôt que rien
//   - pas de cache → full fetch
func (s *GiteaStatsService) GetLanguageStats(ctx context.Context, force bool) (*LangStatsResponse, error) {
	// Sans Redis (contexte de test) → fetch direct sans cache.
	if s.redis == nil {
		return s.fetchLanguages(ctx)
	}

	if force {
		log.Info().Msg("gitlang: force refresh requested, clearing cache")
		s.redis.Del(ctx, langCacheKey)
		return s.fetchAndCacheLanguages(ctx)
	}

	cached, err := s.loadLangCache(ctx)
	if err != nil {
		// Pas de cache → full fetch
		return s.fetchAndCacheLanguages(ctx)
	}

	// Cache assez frais → on rend directement (chemin nominal, aucun appel réseau).
	if time.Since(cached.FetchedAt) < langRefreshInterval {
		return &cached.Response, nil
	}

	// Cache périmé → refetch, mais fallback sur le stale si Gitea est down (pas de page vide).
	resp, err := s.fetchLanguages(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("gitlang: refetch failed, returning stale cache")
		return &cached.Response, nil
	}
	s.saveLangCache(ctx, resp)
	return resp, nil
}

// fetchAndCacheLanguages fait un fetch complet et persiste le résultat.
func (s *GiteaStatsService) fetchAndCacheLanguages(ctx context.Context) (*LangStatsResponse, error) {
	resp, err := s.fetchLanguages(ctx)
	if err != nil {
		return nil, err
	}
	s.saveLangCache(ctx, resp)
	return resp, nil
}

// fetchLanguages liste les repos puis interroge l'endpoint languages de chacun, en parallèle.
func (s *GiteaStatsService) fetchLanguages(ctx context.Context) (*LangStatsResponse, error) {
	// Timeout propre détaché du contexte requête : le calcul peut dépasser la durée d'une requête HTTP
	// (N repos), et on le veut aboutir pour peupler le cache même si le client a abandonné.
	fetchCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	repos, err := s.listRepos(fetchCtx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	log.Info().Int("repos", len(repos)).Msg("gitlang: fetching languages per repo")

	// Agrégation concurrente protégée par mutex (max 5 goroutines, comme gitstats).
	totals := make(map[string]int) // nom de langage Gitea brut → octets
	var mu sync.Mutex
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, repo := range repos {
		if repo.Empty || repo.Fork {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			langs, err := s.repoLanguages(fetchCtx, name)
			if err != nil {
				log.Debug().Err(err).Str("repo", name).Msg("gitlang: skip repo languages")
				return
			}
			mu.Lock()
			for lang, bytes := range langs {
				totals[lang] += bytes
			}
			mu.Unlock()
		}(repo.Name)
	}
	wg.Wait()

	return buildLangStats(totals), nil
}

// repoLanguages interroge /repos/{owner}/{repo}/languages → map langage→octets.
func (s *GiteaStatsService) repoLanguages(ctx context.Context, repoName string) (map[string]int, error) {
	data, err := s.doGet(ctx, fmt.Sprintf("/repos/%s/%s/languages", s.username, repoName))
	if err != nil {
		return nil, err
	}
	var langs map[string]int
	if err := json.Unmarshal(data, &langs); err != nil {
		return nil, fmt.Errorf("parse languages %s: %w", repoName, err)
	}
	return langs, nil
}

// buildLangStats convertit la map octets brute en réponse finale. PUR (testable sans réseau).
// COMMENT : 1. normalise + fusionne les noms qui collapse vers la même clé (ex: "Go" et "go") en
// sommant leurs octets ; 2. dérive LOC = octets / avgBytesPerLine une fois la fusion faite (et non
// par nom, pour ne pas accumuler les arrondis) ; 3. calcule les totaux.
func buildLangStats(byBytes map[string]int) *LangStatsResponse {
	// 1. Fusion par clé normalisée
	merged := make(map[string]int, len(byBytes))
	for name, b := range byBytes {
		merged[normalizeLangKey(name)] += b
	}

	// 2. Conversion + 3. totaux
	langs := make(map[string]LangStat, len(merged))
	totalBytes, totalLOC := 0, 0
	for key, b := range merged {
		loc := b / avgBytesPerLine
		langs[key] = LangStat{Language: key, Bytes: b, LOC: loc}
		totalBytes += b
		totalLOC += loc
	}

	return &LangStatsResponse{
		Languages:  langs,
		TotalLOC:   totalLOC,
		TotalBytes: totalBytes,
		Period:     "all-time",
	}
}

// normalizeLangKey canonicalise un nom de langage Gitea en clé comparable (lowercase, trim).
// Le frontend applique sa propre table d'alias par-dessus pour matcher les noms de skills.
func normalizeLangKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// --- Cache Redis ---

func (s *GiteaStatsService) loadLangCache(ctx context.Context) (*langStatsCache, error) {
	data, err := s.redis.Get(ctx, langCacheKey).Result()
	if err != nil {
		return nil, err
	}
	var cached langStatsCache
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func (s *GiteaStatsService) saveLangCache(ctx context.Context, resp *LangStatsResponse) {
	cached := langStatsCache{Response: *resp, FetchedAt: time.Now()}
	if data, err := json.Marshal(cached); err == nil {
		// Pas de TTL — persiste et se rafraîchit sur l'intervalle (cf. GetLanguageStats).
		s.redis.Set(ctx, langCacheKey, data, 0)
	}
}
