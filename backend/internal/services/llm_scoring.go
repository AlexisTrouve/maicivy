package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// LLMScoringService score les projets via un appel Claude unique
type LLMScoringService struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	redis      *redis.Client
}

// NewLLMScoringService crée une instance. Retourne nil si baseURL ou apiKey vides.
func NewLLMScoringService(baseURL, apiKey string, redisClient *redis.Client) *LLMScoringService {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	return &LLMScoringService{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		redis: redisClient,
	}
}

// ScoreProjectsForTheme retourne map[slug]score (1-100) pour tous les projets.
// Cache Redis 6h par thème — un seul appel LLM par thème.
func (s *LLMScoringService) ScoreProjectsForTheme(ctx context.Context, projects []models.Project, theme *config.CVTheme) (map[string]int, error) {
	cacheKey := fmt.Sprintf("llm:project-scores:%s", theme.ID)

	// Vérifier cache Redis
	if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var scores map[string]int
		if json.Unmarshal([]byte(cached), &scores) == nil {
			return scores, nil
		}
	}

	// Cache miss — appeler le LLM
	prompt := buildProjectScoringPrompt(projects, theme)
	response, err := s.callClaude(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM scoring failed: %w", err)
	}

	scores, err := parseScoreJSON(response, projects)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM scores: %w", err)
	}

	// Mettre en cache 6h
	if data, err := json.Marshal(scores); err == nil {
		s.redis.Set(ctx, cacheKey, data, 6*time.Hour)
	}

	return scores, nil
}

// buildProjectScoringPrompt construit le prompt pour scorer tous les projets d'un coup
func buildProjectScoringPrompt(projects []models.Project, theme *config.CVTheme) string {
	// Lister les tags prioritaires du thème
	tagList := make([]string, 0, len(theme.TagWeights))
	for tag := range theme.TagWeights {
		tagList = append(tagList, tag)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`Tu es un expert en recrutement tech.

Thème recherché: %s — %s
Tags prioritaires: %s

Score chaque projet de 1 à 100 selon sa pertinence pour ce thème de poste.
- 100 = parfaitement ciblé, le recruteur sera impressionné
- 50 = partiellement pertinent, montre des compétences adjacentes
- 1 = aucun rapport avec le thème

Projets (slug | titre | technologies | description):
`, theme.Name, theme.Description, strings.Join(tagList, ", ")))

	for _, p := range projects {
		slug := toSlug(p.Title)
		techs := strings.Join(p.Technologies, ", ")
		// Tronquer la description pour garder le prompt court
		desc := p.Description
		if len(desc) > 120 {
			desc = desc[:120] + "..."
		}
		sb.WriteString(fmt.Sprintf("- %s | %s | %s | %s\n", slug, p.Title, techs, desc))
	}

	sb.WriteString(`
Réponds UNIQUEMENT avec un objet JSON valide, sans texte autour, sans markdown:
{
  "slug-du-projet": 85,
  ...
}`)

	return sb.String()
}

// callClaude envoie le prompt au proxy et retourne le texte de la réponse
func (s *LLMScoringService) callClaude(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 512,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBytes))
	}

	// Parser la réponse Anthropic : content[0].text
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty API response")
	}

	return apiResp.Content[0].Text, nil
}

// parseScoreJSON parse le JSON retourné par le LLM, avec fallback si le JSON est dans un bloc markdown
func parseScoreJSON(text string, projects []models.Project) (map[string]int, error) {
	// Nettoyer le texte (parfois le LLM met ``` autour)
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
	}
	if idx := strings.LastIndex(text, "}"); idx >= 0 {
		text = text[:idx+1]
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w\ntext: %s", err, text)
	}

	scores := make(map[string]int, len(raw))
	for key, val := range raw {
		switch v := val.(type) {
		case float64:
			score := int(v)
			if score < 1 {
				score = 1
			}
			if score > 100 {
				score = 100
			}
			scores[key] = score
		}
	}

	// S'assurer que tous les projets ont un score (fallback 50 si absent)
	for _, p := range projects {
		slug := toSlug(p.Title)
		if _, ok := scores[slug]; !ok {
			scores[slug] = 50
		}
	}

	return scores, nil
}

// toSlug convertit un titre en identifiant slug (lowercase, tirets)
func toSlug(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Supprimer les caractères non alphanumériques sauf tirets
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.Trim(result.String(), "-")
}
