package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// maiProFilesBaseURL — API de profil live
const maiProFilesBaseURL = "https://maiprofiles.etheryale.com"

// cacheTTL — durée de vie d'une entrée en cache (5 minutes)
const cacheTTL = 5 * time.Minute

// --- Types API ---

// MPFProfile représente la réponse de GET /profile
type MPFProfile struct {
	Name            string         `json:"name"`
	Headline        string         `json:"headline"`
	Location        string         `json:"location"`
	ExperienceYears int            `json:"experience_years"`
	Contact         MPFContact     `json:"contact"`
	Bio             MPFBio         `json:"bio"`
	Skills          MPFSkills      `json:"skills"`
	Domains         []string       `json:"domains"`
	Links           MPFLinks       `json:"links"`
}

type MPFContact struct {
	Email    string `json:"email"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Gitea    string `json:"gitea"`
}

type MPFBio struct {
	Short string `json:"short"`
	Full  string `json:"full"`
}

type MPFSkills struct {
	Strong   []string `json:"strong"`
	Familiar []string `json:"familiar"`
}

type MPFLinks struct {
	Portfolio string `json:"portfolio"`
	GitHub    string `json:"github"`
	Gitea     string `json:"gitea"`
}

// MPFProject représente un projet retourné par GET /projects ou GET /projects/{id}
type MPFProject struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Status      string            `json:"status"`
	Stack       []string          `json:"stack"`
	Stats       MPFStats          `json:"stats"`
	Description MPFDescription    `json:"description"`
	Tags        []string          `json:"tags"`
	Screenshots []MPFScreenshot   `json:"screenshots"`
	Links       map[string]string `json:"links"`
}

type MPFStats struct {
	LOC     int `json:"loc"`
	Tests   int `json:"tests"`
	Modules int `json:"modules"`
}

type MPFDescription struct {
	Short     string `json:"short"`
	Portfolio string `json:"portfolio"`
}

type MPFScreenshot struct {
	Filename string `json:"filename"`
	Section  string `json:"section"`
}

// MPFGlobalStats représente GET /stats
type MPFGlobalStats struct {
	Projects  int            `json:"projects"`
	TotalLOC  int            `json:"total_loc"`
	TotalTests int           `json:"total_tests"`
	Stack     map[string]int `json:"stack"`
}

// --- Cache générique ---

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// MaiProFilesClient — client HTTP avec cache in-memory TTL 5min
type MaiProFilesClient struct {
	http    *http.Client
	baseURL string
	cache   sync.Map // clé string → *cacheEntry
}

func NewMaiProFilesClient() *MaiProFilesClient {
	return &MaiProFilesClient{
		http:    &http.Client{Timeout: 8 * time.Second},
		baseURL: maiProFilesBaseURL,
	}
}

// get fait un GET sur path et décode le JSON dans out.
// Cherche d'abord dans le cache, sinon fetch et met en cache.
func (c *MaiProFilesClient) get(ctx context.Context, path string, out interface{}) error {
	// Vérifier le cache
	if cached, ok := c.cache.Load(path); ok {
		entry := cached.(*cacheEntry)
		if !entry.isExpired() {
			// Ré-encoder/décoder pour copier dans out (types différents selon l'appelant)
			b, err := json.Marshal(entry.value)
			if err == nil {
				if err = json.Unmarshal(b, out); err == nil {
					return nil
				}
			}
		}
	}

	// Fetch l'API
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("maiprofiles fetch %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", path)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("maiprofiles %s: HTTP %d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Décoder dans une interface{} générique pour le cache
	var raw interface{}
	if err = json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("json decode %s: %w", path, err)
	}

	// Mettre en cache
	c.cache.Store(path, &cacheEntry{value: raw, expiresAt: time.Now().Add(cacheTTL)})

	// Décoder dans le type cible
	return json.Unmarshal(body, out)
}

// GetProfile retourne le profil complet
func (c *MaiProFilesClient) GetProfile(ctx context.Context) (*MPFProfile, error) {
	var p MPFProfile
	err := c.get(ctx, "/profile", &p)
	return &p, err
}

// ListProjects retourne tous les projets (sans description.portfolio)
func (c *MaiProFilesClient) ListProjects(ctx context.Context) ([]MPFProject, error) {
	var projects []MPFProject
	err := c.get(ctx, "/projects", &projects)
	return projects, err
}

// GetProject retourne les détails complets d'un projet par ID
func (c *MaiProFilesClient) GetProject(ctx context.Context, id string) (*MPFProject, error) {
	var p MPFProject
	err := c.get(ctx, "/projects/"+id, &p)
	return &p, err
}

// GetStats retourne les stats globales
func (c *MaiProFilesClient) GetStats(ctx context.Context) (*MPFGlobalStats, error) {
	var s MPFGlobalStats
	err := c.get(ctx, "/stats", &s)
	return &s, err
}

// Search recherche des projets par mot-clé
func (c *MaiProFilesClient) Search(ctx context.Context, query string) ([]MPFProject, error) {
	var results []MPFProject
	path := "/search?q=" + url.QueryEscape(query)
	err := c.get(ctx, path, &results)
	return results, err
}

// InvalidateCache vide l'entrée en cache pour un path donné (utile après PATCH)
func (c *MaiProFilesClient) InvalidateCache(path string) {
	c.cache.Delete(path)
}
