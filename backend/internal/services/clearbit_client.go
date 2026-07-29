// backend/internal/services/clearbit_client.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClearbitClient gère les appels à l'API Clearbit Enrichment
type ClearbitClient struct {
	apiKey     string
	httpClient *http.Client
	redis      *redis.Client
}

// ClearbitResponse structure de la réponse Clearbit
type ClearbitResponse struct {
	IP          string `json:"ip"`
	Company     string `json:"company,omitempty"`
	CompanyName string `json:"name,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Type        string `json:"type,omitempty"`
	Industry    string `json:"industry,omitempty"`
	Size        string `json:"employeesRange,omitempty"`
	Location    struct {
		City    string `json:"city,omitempty"`
		Country string `json:"country,omitempty"`
	} `json:"geo,omitempty"`
	Person struct {
		Role  string `json:"role,omitempty"`
		Title string `json:"title,omitempty"`
	} `json:"person,omitempty"`
}

// NewClearbitClient crée un nouveau client Clearbit
func NewClearbitClient(redis *redis.Client) *ClearbitClient {
	apiKey := os.Getenv("CLEARBIT_API_KEY")
	if apiKey == "" {
		// Log warning mais ne pas crash - graceful degradation
		fmt.Println("WARNING: CLEARBIT_API_KEY not set, profile enrichment will be disabled")
	}

	return &ClearbitClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		redis: redis,
	}
}

// EnrichByIP enrichit les informations via l'IP avec Clearbit API
func (c *ClearbitClient) EnrichByIP(ctx context.Context, hashedIP, realIP string) (map[string]interface{}, error) {
	// Si pas de clé API, retourner nil (graceful degradation)
	if c.apiKey == "" {
		return nil, fmt.Errorf("clearbit api key not configured")
	}

	// 1. Vérifier le cache Redis (TTL 7 jours)
	cacheKey := fmt.Sprintf("clearbit:ip:%s", hashedIP)
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return data, nil
		}
	}

	// 2. Appeler l'API Clearbit
	url := fmt.Sprintf("https://person.clearbit.com/v1/people/find?ip=%s", realIP)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Authentification Bearer
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Exécuter la requête
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("clearbit api request failed: %w", err)
	}
	defer resp.Body.Close()

	// Vérifier le status code
	if resp.StatusCode == 404 {
		// Pas de données pour cette IP - normal, pas une erreur
		return nil, nil
	}

	if resp.StatusCode == 429 {
		// Rate limit atteint
		return nil, fmt.Errorf("clearbit rate limit exceeded")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("clearbit api error: status %d", resp.StatusCode)
	}

	// 3. Parser la réponse
	var clearbitResp ClearbitResponse
	if err := json.NewDecoder(resp.Body).Decode(&clearbitResp); err != nil {
		return nil, fmt.Errorf("failed to decode clearbit response: %w", err)
	}

	// 4. Convertir en map pour flexibilité
	enrichmentData := make(map[string]interface{})

	if clearbitResp.CompanyName != "" {
		enrichmentData["company_name"] = clearbitResp.CompanyName
	}
	if clearbitResp.Domain != "" {
		enrichmentData["company_domain"] = clearbitResp.Domain
	}
	if clearbitResp.Type != "" {
		enrichmentData["company_type"] = clearbitResp.Type
	}
	if clearbitResp.Industry != "" {
		enrichmentData["industry"] = clearbitResp.Industry
	}
	if clearbitResp.Size != "" {
		enrichmentData["company_size"] = clearbitResp.Size
	}
	if clearbitResp.Location.City != "" {
		enrichmentData["city"] = clearbitResp.Location.City
	}
	if clearbitResp.Location.Country != "" {
		enrichmentData["country"] = clearbitResp.Location.Country
	}
	if clearbitResp.Person.Role != "" {
		enrichmentData["job_role"] = clearbitResp.Person.Role
	}
	if clearbitResp.Person.Title != "" {
		enrichmentData["job_title"] = clearbitResp.Person.Title
	}

	// 5. Cacher le résultat (TTL 7 jours)
	if len(enrichmentData) > 0 {
		dataJSON, _ := json.Marshal(enrichmentData)
		c.redis.Set(ctx, cacheKey, string(dataJSON), 7*24*time.Hour)
	}

	return enrichmentData, nil
}

// GetCachedEnrichment récupère uniquement les données en cache (sans appel API)
func (c *ClearbitClient) GetCachedEnrichment(ctx context.Context, hashedIP string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("clearbit:ip:%s", hashedIP)
	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cached), &data); err != nil {
		return nil, err
	}

	return data, nil
}

// ClearCache supprime les données en cache pour une IP
func (c *ClearbitClient) ClearCache(ctx context.Context, hashedIP string) error {
	cacheKey := fmt.Sprintf("clearbit:ip:%s", hashedIP)
	return c.redis.Del(ctx, cacheKey).Err()
}

// GetCacheStats retourne des stats sur le cache Clearbit
func (c *ClearbitClient) GetCacheStats(ctx context.Context) (map[string]int, error) {
	pattern := "clearbit:ip:*"
	keys, err := c.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"cached_ips": len(keys),
	}

	return stats, nil
}
