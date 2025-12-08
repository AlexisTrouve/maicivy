# 08. BACKEND AI SERVICES

## 📋 Métadonnées

- **Phase:** 3
- **Priorité:** 🟡 HAUTE
- **Complexité:** ⭐⭐⭐⭐⭐ (5/5)
- **Prérequis:** 04. BACKEND_MIDDLEWARES.md
- **Temps estimé:** 5-7 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter les services backend d'intelligence artificielle pour la génération de lettres de motivation et anti-motivation. Ce module inclut :

1. **Service IA** : clients pour Claude (Anthropic) et GPT-4 (OpenAI) avec fallback automatique
2. **Service Scraper** : extraction automatique d'informations sur les entreprises
3. **Service PDF Lettres** : génération de PDF pour les deux types de lettres
4. **Prompts Engineering** : templates optimisés pour génération de contenu professionnel et humoristique
5. **Error Handling & Retry** : gestion robuste des erreurs API avec stratégie de retry
6. **Cost Tracking** : suivi des tokens utilisés et estimation des coûts

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────┐
│                    Letters API Handler                       │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    AI Service Orchestrator                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Claude     │  │   GPT-4      │  │   Fallback   │      │
│  │   Client     │  │   Client     │  │   Strategy   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└───────────────────────┬─────────────────────────────────────┘
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Scraper  │  │  Prompt  │  │   PDF    │
    │ Service  │  │ Builder  │  │ Service  │
    └──────────┘  └──────────┘  └──────────┘
          │             │             │
          ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │  Redis   │  │Templates │  │ chromedp │
    │  Cache   │  │ Storage  │  │  HTML→PDF│
    └──────────┘  └──────────┘  └──────────┘
```

### Design Decisions

**1. Multi-Provider Strategy (Claude + GPT-4)**
- **Justification** : Résilience et diversité de styles
- Claude Primary : meilleur pour textes créatifs/professionnels
- GPT-4 Fallback : si Claude rate limit ou erreur
- Sélection basée sur disponibilité + coût

**2. Scraping + API Hybride**
- **Justification** : Maximiser données disponibles
- APIs d'abord (Clearbit, Hunter.io) : données structurées
- Scraping en fallback : site web entreprise
- Cache Redis (TTL 7j) : éviter requêtes répétées

**3. chromedp pour PDF**
- **Justification** : Rendu HTML professionnel
- Alternative à gofpdf : plus de flexibilité design
- Templates HTML : facile à designer/modifier
- Headless Chrome : rendu exact du web

**4. Prompt Engineering Dédié**
- **Justification** : Qualité des outputs
- Two-shot examples : guide le style
- Variables dynamiques : personnalisation maximale
- Séparation motivation/anti-motivation : logique distincte

**5. Retry avec Exponential Backoff**
- **Justification** : Gestion rate limits API
- Max 3 retries : éviter attente infinie
- Backoff exponentiel : 1s, 2s, 4s
- Circuit breaker : si trop d'erreurs consécutives

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# IA APIs
go get github.com/anthropics/anthropic-sdk-go
go get github.com/sashabaranov/go-openai

# HTTP Client amélioré
go get github.com/hashicorp/go-retryablehttp

# Web Scraping
go get github.com/gocolly/colly/v2
go get github.com/PuerkitoBio/goquery

# PDF Generation (chromedp)
go get github.com/chromedp/chromedp

# HTML Templating
go get html/template  # stdlib

# Rate Limiting
go get golang.org/x/time/rate

# Cost Tracking / Metrics
go get github.com/prometheus/client_golang/prometheus
```

### Services Externes

**APIs IA (REQUISES) :**
- **Anthropic Claude API** : clé API dans `.env` (`ANTHROPIC_API_KEY`)
  - Endpoint : `https://api.anthropic.com/v1/messages`
  - Model : `claude-3-5-sonnet-20241022` (recommandé)
  - Pricing : ~$3/MTok input, ~$15/MTok output

- **OpenAI GPT-4 API** : clé API dans `.env` (`OPENAI_API_KEY`)
  - Endpoint : `https://api.openai.com/v1/chat/completions`
  - Model : `gpt-4-turbo-preview`
  - Pricing : ~$10/MTok input, ~$30/MTok output

**APIs Enrichissement Entreprises (OPTIONNELLES) :**
- **Clearbit API** : `CLEARBIT_API_KEY` (gratuit tier limité)
  - Enrichissement données entreprise via domaine
- **Hunter.io** : `HUNTER_API_KEY` (recherche emails)
  - Utile pour valider info entreprise

---

## 🔨 Implémentation

### Étape 1 : Configuration et Structures de Base

**Description :** Définir les structures de données, configuration, et interfaces des services.

**Fichier :** `backend/internal/config/ai.go`

```go
package config

import (
	"os"
	"time"
)

type AIConfig struct {
	// Providers
	AnthropicAPIKey string
	OpenAIAPIKey    string
	PrimaryProvider string // "claude" ou "openai"

	// Models
	ClaudeModel string
	OpenAIModel string

	// Rate Limiting
	MaxRequestsPerMinute int
	MaxTokensPerRequest  int

	// Retry Strategy
	MaxRetries      int
	RetryBaseDelay  time.Duration
	RequestTimeout  time.Duration

	// Cost Tracking
	EnableCostTracking bool
}

func LoadAIConfig() *AIConfig {
	return &AIConfig{
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		PrimaryProvider:      getEnvOrDefault("AI_PRIMARY_PROVIDER", "claude"),
		ClaudeModel:          getEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
		OpenAIModel:          getEnvOrDefault("OPENAI_MODEL", "gpt-4-turbo-preview"),
		MaxRequestsPerMinute: getEnvAsInt("AI_MAX_REQUESTS_PER_MIN", 10),
		MaxTokensPerRequest:  getEnvAsInt("AI_MAX_TOKENS", 4000),
		MaxRetries:           getEnvAsInt("AI_MAX_RETRIES", 3),
		RetryBaseDelay:       time.Second,
		RequestTimeout:       30 * time.Second,
		EnableCostTracking:   getEnvAsBool("AI_ENABLE_COST_TRACKING", true),
	}
}

type ScraperConfig struct {
	ClearbitAPIKey string
	HunterAPIKey   string
	UserAgent      string
	Timeout        time.Duration
	CacheTTL       time.Duration // 7 jours par défaut
}

func LoadScraperConfig() *ScraperConfig {
	return &ScraperConfig{
		ClearbitAPIKey: os.Getenv("CLEARBIT_API_KEY"),
		HunterAPIKey:   os.Getenv("HUNTER_API_KEY"),
		UserAgent:      "maicivy-bot/1.0 (+https://maicivy.example.com/bot)",
		Timeout:        15 * time.Second,
		CacheTTL:       7 * 24 * time.Hour, // 7 jours
	}
}
```

**Fichier :** `backend/internal/models/ai.go`

```go
package models

import "time"

// LetterType : type de lettre générée
type LetterType string

const (
	LetterTypeMotivation     LetterType = "motivation"
	LetterTypeAntiMotivation LetterType = "anti_motivation"
)

// CompanyInfo : informations sur l'entreprise cible
type CompanyInfo struct {
	Name        string   `json:"name"`
	Domain      string   `json:"domain"`
	Description string   `json:"description"`
	Industry    string   `json:"industry"`
	Size        string   `json:"size"`
	Technologies []string `json:"technologies,omitempty"`
	Culture      string   `json:"culture,omitempty"`
	Values       []string `json:"values,omitempty"`
	RecentNews   string   `json:"recent_news,omitempty"`
}

// LetterRequest : requête de génération de lettre
type LetterRequest struct {
	CompanyName string     `json:"company_name" validate:"required,min=2"`
	LetterType  LetterType `json:"letter_type" validate:"required,oneof=motivation anti_motivation"`
	UserProfile UserProfile `json:"user_profile,omitempty"`
}

// UserProfile : profil utilisateur pour personnalisation
type UserProfile struct {
	Name        string   `json:"name"`
	CurrentRole string   `json:"current_role"`
	Skills      []string `json:"skills"`
	Experience  int      `json:"experience_years"`
}

// LetterResponse : lettre générée
type LetterResponse struct {
	Content     string      `json:"content"`
	Type        LetterType  `json:"type"`
	CompanyInfo CompanyInfo `json:"company_info"`
	GeneratedAt time.Time   `json:"generated_at"`
	Provider    string      `json:"provider"` // "claude" ou "openai"
	TokensUsed  int         `json:"tokens_used"`
	EstimatedCost float64   `json:"estimated_cost"`
}

// AIMetrics : métriques de coût et usage
type AIMetrics struct {
	Provider       string
	Model          string
	TokensInput    int
	TokensOutput   int
	TotalTokens    int
	EstimatedCost  float64
	ResponseTimeMs int64
	Success        bool
	ErrorMessage   string
}
```

**Explications :**
- `AIConfig` : centralise tous les paramètres configurables (API keys, modèles, retry)
- `CompanyInfo` : structure données entreprise (résultat scraping)
- `LetterRequest/Response` : contrat API pour génération lettres
- `AIMetrics` : tracking coûts et performance

---

### Étape 2 : Service Scraper d'Entreprises

**Description :** Récupérer automatiquement les informations d'une entreprise via APIs et scraping web.

**Fichier :** `backend/internal/services/scraper.go`

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/config"
	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type CompanyScraper struct {
	config      *config.ScraperConfig
	redisClient *redis.Client
	httpClient  *http.Client
	log         *logger.Logger
}

func NewCompanyScraper(cfg *config.ScraperConfig, redis *redis.Client, log *logger.Logger) *CompanyScraper {
	return &CompanyScraper{
		config:      cfg,
		redisClient: redis,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log: log,
	}
}

// GetCompanyInfo : point d'entrée principal
func (s *CompanyScraper) GetCompanyInfo(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	// 1. Check cache Redis
	cacheKey := fmt.Sprintf("company_info:%s", strings.ToLower(companyName))
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var info models.CompanyInfo
		if json.Unmarshal([]byte(cached), &info) == nil {
			s.log.Info("Company info found in cache", "company", companyName)
			return &info, nil
		}
	}

	// 2. Tenter enrichissement via APIs
	info, err := s.enrichViaAPIs(ctx, companyName)
	if err != nil {
		s.log.Warn("API enrichment failed, fallback to scraping", "error", err)
		// 3. Fallback: scraping web
		info, err = s.scrapeCompanyWebsite(ctx, companyName)
		if err != nil {
			return nil, fmt.Errorf("failed to get company info: %w", err)
		}
	}

	// 4. Cache résultat
	data, _ := json.Marshal(info)
	s.redisClient.Set(ctx, cacheKey, data, s.config.CacheTTL)

	return info, nil
}

// enrichViaAPIs : utilise Clearbit ou autres APIs
func (s *CompanyScraper) enrichViaAPIs(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	if s.config.ClearbitAPIKey == "" {
		return nil, fmt.Errorf("no API key configured")
	}

	// Clearbit Company API
	// https://company.clearbit.com/v2/companies/find?domain=example.com
	domain := s.guessDomainFromName(companyName)
	url := fmt.Sprintf("https://company.clearbit.com/v2/companies/find?domain=%s", domain)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.config.ClearbitAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clearbit API returned %d", resp.StatusCode)
	}

	var clearbitData struct {
		Name        string   `json:"name"`
		Domain      string   `json:"domain"`
		Description string   `json:"description"`
		Category    struct {
			Industry string `json:"industry"`
		} `json:"category"`
		Metrics struct {
			Employees string `json:"employees"`
		} `json:"metrics"`
		Tech []string `json:"tech"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&clearbitData); err != nil {
		return nil, err
	}

	return &models.CompanyInfo{
		Name:         clearbitData.Name,
		Domain:       clearbitData.Domain,
		Description:  clearbitData.Description,
		Industry:     clearbitData.Category.Industry,
		Size:         clearbitData.Metrics.Employees,
		Technologies: clearbitData.Tech,
	}, nil
}

// scrapeCompanyWebsite : scraping fallback
func (s *CompanyScraper) scrapeCompanyWebsite(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	domain := s.guessDomainFromName(companyName)
	url := fmt.Sprintf("https://%s", domain)

	info := &models.CompanyInfo{
		Name:   companyName,
		Domain: domain,
	}

	c := colly.NewCollector(
		colly.UserAgent(s.config.UserAgent),
		colly.AllowedDomains(domain),
	)

	// Extract meta description
	c.OnHTML("meta[name=description]", func(e *colly.HTMLElement) {
		info.Description = e.Attr("content")
	})

	// Extract about text (heuristique simple)
	c.OnHTML("section:contains('About'), div:contains('À propos')", func(e *colly.HTMLElement) {
		if info.Description == "" {
			info.Description = strings.TrimSpace(e.Text)
			if len(info.Description) > 500 {
				info.Description = info.Description[:500] + "..."
			}
		}
	})

	// Timeout context
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- c.Visit(url)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, fmt.Errorf("scraping failed: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("scraping timeout")
	}

	// Validation minimum
	if info.Description == "" {
		info.Description = fmt.Sprintf("Entreprise située à %s", domain)
	}

	return info, nil
}

// guessDomainFromName : devine le domaine depuis le nom
func (s *CompanyScraper) guessDomainFromName(name string) string {
	// Logique simple : lowercase, remove spaces, add .com
	// Peut être amélioré avec recherche Google ou API
	domain := strings.ToLower(name)
	domain = strings.ReplaceAll(domain, " ", "")
	domain = strings.ReplaceAll(domain, ".", "")

	// Cas spéciaux connus
	knownDomains := map[string]string{
		"google":    "google.com",
		"microsoft": "microsoft.com",
		"apple":     "apple.com",
		"amazon":    "amazon.com",
		// ... ajouter plus selon besoin
	}

	if known, ok := knownDomains[domain]; ok {
		return known
	}

	return domain + ".com"
}
```

**Explications :**
- **Cache-first strategy** : évite requêtes répétées (TTL 7j)
- **API puis scraping** : maximise qualité données
- **Timeout protection** : évite blocage sur sites lents
- **Heuristiques domain** : améliorer avec vraie recherche DNS/Google si nécessaire

---

### Étape 3 : Prompts Engineering

**Description :** Templates de prompts optimisés pour les deux types de lettres.

**Fichier :** `backend/internal/services/prompts.go`

```go
package services

import (
	"fmt"
	"strings"

	"maicivy/internal/models"
)

type PromptBuilder struct {
	userProfile models.UserProfile
}

func NewPromptBuilder(profile models.UserProfile) *PromptBuilder {
	return &PromptBuilder{userProfile: profile}
}

// BuildMotivationPrompt : prompt pour lettre de motivation professionnelle
func (pb *PromptBuilder) BuildMotivationPrompt(company models.CompanyInfo) string {
	template := `Tu es un expert en rédaction de lettres de motivation professionnelles.

CONTEXTE CANDIDAT:
- Nom: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

CONTEXTE ENTREPRISE:
- Nom: %s
- Secteur: %s
- Description: %s
- Technologies utilisées: %s
- Taille: %s

TÂCHE:
Rédige une lettre de motivation professionnelle, convaincante et authentique pour postuler chez %s.

INSTRUCTIONS:
1. Structure classique (introduction, corps, conclusion)
2. Ton professionnel mais pas rigide
3. Mets en avant l'alignement entre les compétences du candidat et les besoins probables de l'entreprise
4. Montre un intérêt sincère pour l'entreprise (culture, projets, technologies)
5. Sois spécifique et concret (évite les généralités)
6. Longueur: 250-350 mots
7. Format: paragraphes bien structurés (pas de bullet points)

EXEMPLES DE BON STYLE:
- "Votre engagement dans [technologie/projet spécifique] résonne particulièrement avec mon expérience en..."
- "Ayant travaillé X années sur [compétence], je serais ravi de contribuer à [objectif entreprise]..."

N'invente PAS de faits sur l'entreprise. Utilise uniquement les informations fournies.

Génère la lettre maintenant (sans formule de politesse finale "Cordialement", etc.) :`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		company.Name,
		company.Industry,
		company.Description,
		strings.Join(company.Technologies, ", "),
		company.Size,
		company.Name,
	)
}

// BuildAntiMotivationPrompt : prompt pour lettre d'anti-motivation humoristique
func (pb *PromptBuilder) BuildAntiMotivationPrompt(company models.CompanyInfo) string {
	template := `Tu es un humoriste spécialisé en rédaction de lettres d'anti-motivation créatives et absurdes.

CONTEXTE CANDIDAT:
- Nom: %s
- Poste actuel: %s
- Années d'expérience: %d ans
- Compétences clés: %s

CONTEXTE ENTREPRISE:
- Nom: %s
- Secteur: %s
- Description: %s

TÂCHE:
Rédige une lettre d'ANTI-MOTIVATION humoristique expliquant pourquoi le candidat ne devrait SURTOUT PAS être embauché chez %s.

STYLE ET TON:
- Humour absurde et auto-dérision
- Deuxième degré évident (personne ne doit prendre ça au sérieux)
- Références pop culture, jeux de mots, exagérations comiques
- Ton léger, jamais méchant ou offensant envers l'entreprise
- Créatif et original

INSTRUCTIONS:
1. Structure libre (sois créatif !)
2. Liste de "défauts" hilarants et absurdes
3. Fausses compétences inutiles ("Expert en procrastination", "Champion de café froid", etc.)
4. Anecdotes inventées ridicules
5. Conclusion ironique inversée
6. Longueur: 200-300 mots
7. Évite l'humour vulgaire ou offensant

EXEMPLES DE STYLE:
- "Mes 10 ans d'expérience en débogage de code m'ont surtout appris à créer des bugs encore plus créatifs..."
- "Je maîtrise l'art ancestral de transformer un projet de 2 semaines en 6 mois..."
- "Mon CV ressemble à un README.md mal formaté, ce qui est ironique vu que je suis développeur..."

RAPPEL: C'est de l'humour ! Le but est de faire sourire tout en montrant créativité et auto-dérision.

Génère la lettre maintenant:`

	return fmt.Sprintf(
		template,
		pb.userProfile.Name,
		pb.userProfile.CurrentRole,
		pb.userProfile.Experience,
		strings.Join(pb.userProfile.Skills, ", "),
		company.Name,
		company.Industry,
		company.Description,
		company.Name,
	)
}
```

**Explications :**
- **Prompts détaillés** : instructions claires pour guider l'IA
- **Few-shot examples** : guide le style attendu
- **Variables dynamiques** : personnalisation maximale
- **Contraintes claires** : longueur, ton, format
- **Anti-motivation safe** : humour sans offense

---

### Étape 4 : Service IA (Claude + GPT-4)

**Description :** Client unifié pour Claude et GPT-4 avec fallback automatique.

**Fichier :** `backend/internal/services/ai.go`

```go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/time/rate"

	"maicivy/internal/config"
	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type AIService struct {
	config          *config.AIConfig
	claudeClient    *anthropic.Client
	openaiClient    *openai.Client
	rateLimiter     *rate.Limiter
	log             *logger.Logger
	metricsRecorder MetricsRecorder
}

type MetricsRecorder interface {
	RecordAIMetrics(metrics models.AIMetrics)
}

func NewAIService(cfg *config.AIConfig, log *logger.Logger, metrics MetricsRecorder) (*AIService, error) {
	// Validate API keys
	if cfg.AnthropicAPIKey == "" && cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("at least one AI provider API key required")
	}

	svc := &AIService{
		config:          cfg,
		rateLimiter:     rate.NewLimiter(rate.Limit(cfg.MaxRequestsPerMinute)/60, 1),
		log:             log,
		metricsRecorder: metrics,
	}

	// Initialize Claude client
	if cfg.AnthropicAPIKey != "" {
		svc.claudeClient = anthropic.NewClient(cfg.AnthropicAPIKey)
	}

	// Initialize OpenAI client
	if cfg.OpenAIAPIKey != "" {
		svc.openaiClient = openai.NewClient(cfg.OpenAIAPIKey)
	}

	return svc, nil
}

// GenerateText : génère du texte avec fallback automatique
func (s *AIService) GenerateText(ctx context.Context, prompt string) (string, *models.AIMetrics, error) {
	// Rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return "", nil, fmt.Errorf("rate limit: %w", err)
	}

	var (
		text    string
		metrics *models.AIMetrics
		err     error
	)

	// Tenter provider primaire
	if s.config.PrimaryProvider == "claude" && s.claudeClient != nil {
		text, metrics, err = s.generateWithClaude(ctx, prompt)
		if err == nil {
			return text, metrics, nil
		}
		s.log.Warn("Claude generation failed, trying OpenAI fallback", "error", err)
	}

	// Fallback OpenAI (ou primaire si PrimaryProvider == "openai")
	if s.openaiClient != nil {
		text, metrics, err = s.generateWithOpenAI(ctx, prompt)
		if err == nil {
			return text, metrics, nil
		}
	}

	return "", nil, fmt.Errorf("all AI providers failed")
}

// generateWithClaude : génération via Claude
func (s *AIService) generateWithClaude(ctx context.Context, prompt string) (string, *models.AIMetrics, error) {
	start := time.Now()
	metrics := &models.AIMetrics{
		Provider: "claude",
		Model:    s.config.ClaudeModel,
	}

	// Context avec timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	// Appel API avec retry
	var resp *anthropic.MessageResponse
	var err error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			s.log.Info("Retrying Claude request", "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		resp, err = s.claudeClient.Messages.Create(ctx, &anthropic.MessageCreateParams{
			Model: s.config.ClaudeModel,
			Messages: []anthropic.MessageParam{
				{
					Role:    "user",
					Content: prompt,
				},
			},
			MaxTokens: s.config.MaxTokensPerRequest,
		})

		if err == nil {
			break
		}

		// Si erreur non-retryable, arrêter
		if !isRetryableError(err) {
			break
		}
	}

	metrics.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		metrics.Success = false
		metrics.ErrorMessage = err.Error()
		s.recordMetrics(metrics)
		return "", metrics, fmt.Errorf("claude API error: %w", err)
	}

	// Parse response
	if len(resp.Content) == 0 {
		return "", metrics, fmt.Errorf("empty response from Claude")
	}

	text := resp.Content[0].Text

	// Metrics
	metrics.TokensInput = resp.Usage.InputTokens
	metrics.TokensOutput = resp.Usage.OutputTokens
	metrics.TotalTokens = metrics.TokensInput + metrics.TokensOutput
	metrics.EstimatedCost = s.calculateClaudeCost(metrics.TokensInput, metrics.TokensOutput)
	metrics.Success = true

	s.recordMetrics(metrics)

	return text, metrics, nil
}

// generateWithOpenAI : génération via GPT-4
func (s *AIService) generateWithOpenAI(ctx context.Context, prompt string) (string, *models.AIMetrics, error) {
	start := time.Now()
	metrics := &models.AIMetrics{
		Provider: "openai",
		Model:    s.config.OpenAIModel,
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	var resp openai.ChatCompletionResponse
	var err error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		resp, err = s.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: s.config.OpenAIModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: s.config.MaxTokensPerRequest,
		})

		if err == nil {
			break
		}

		if !isRetryableError(err) {
			break
		}
	}

	metrics.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		metrics.Success = false
		metrics.ErrorMessage = err.Error()
		s.recordMetrics(metrics)
		return "", metrics, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", metrics, fmt.Errorf("empty response from OpenAI")
	}

	text := resp.Choices[0].Message.Content

	metrics.TokensInput = resp.Usage.PromptTokens
	metrics.TokensOutput = resp.Usage.CompletionTokens
	metrics.TotalTokens = resp.Usage.TotalTokens
	metrics.EstimatedCost = s.calculateOpenAICost(metrics.TokensInput, metrics.TokensOutput)
	metrics.Success = true

	s.recordMetrics(metrics)

	return text, metrics, nil
}

// calculateClaudeCost : estime coût Claude
func (s *AIService) calculateClaudeCost(inputTokens, outputTokens int) float64 {
	// Claude 3.5 Sonnet pricing (Dec 2024)
	// Input: $3/MTok, Output: $15/MTok
	inputCost := float64(inputTokens) / 1_000_000 * 3.0
	outputCost := float64(outputTokens) / 1_000_000 * 15.0
	return inputCost + outputCost
}

// calculateOpenAICost : estime coût OpenAI
func (s *AIService) calculateOpenAICost(inputTokens, outputTokens int) float64 {
	// GPT-4 Turbo pricing
	// Input: $10/MTok, Output: $30/MTok
	inputCost := float64(inputTokens) / 1_000_000 * 10.0
	outputCost := float64(outputTokens) / 1_000_000 * 30.0
	return inputCost + outputCost
}

// recordMetrics : enregistre métriques
func (s *AIService) recordMetrics(m *models.AIMetrics) {
	if s.config.EnableCostTracking && s.metricsRecorder != nil {
		s.metricsRecorder.RecordAIMetrics(*m)
	}

	s.log.Info("AI generation completed",
		"provider", m.Provider,
		"tokens", m.TotalTokens,
		"cost", m.EstimatedCost,
		"duration_ms", m.ResponseTimeMs,
		"success", m.Success,
	)
}

// isRetryableError : détermine si erreur est retryable
func isRetryableError(err error) bool {
	// Rate limits, timeouts, 5xx errors → retry
	// 4xx (sauf 429) → ne pas retry
	errStr := err.Error()

	retryablePatterns := []string{
		"rate_limit",
		"timeout",
		"503",
		"502",
		"500",
		"429",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Explications :**
- **Fallback automatique** : Claude primary → OpenAI si erreur
- **Retry avec backoff** : 1s, 2s, 4s (exponential)
- **Rate limiting** : protection débit
- **Cost tracking** : calcul coûts en temps réel
- **Timeout protection** : 30s max par requête
- **Metrics complètes** : tokens, coût, durée, succès

---

### Étape 5 : Service PDF Lettres (chromedp)

**Description :** Génération de PDFs professionnels à partir de templates HTML.

**Fichier :** `backend/internal/services/pdf_letters.go`

```go
package services

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type PDFLetterService struct {
	templates *template.Template
	log       *logger.Logger
}

func NewPDFLetterService(templatesPath string, log *logger.Logger) (*PDFLetterService, error) {
	// Load templates
	tmpl, err := template.ParseGlob(templatesPath + "/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &PDFLetterService{
		templates: tmpl,
		log:       log,
	}, nil
}

// GeneratePDF : génère PDF d'une lettre
func (s *PDFLetterService) GeneratePDF(ctx context.Context, letter models.LetterResponse, writer io.Writer) error {
	// 1. Render HTML from template
	html, err := s.renderHTML(letter)
	if err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}

	// 2. Convert HTML to PDF via chromedp
	return s.htmlToPDF(ctx, html, writer)
}

// renderHTML : génère HTML depuis template
func (s *PDFLetterService) renderHTML(letter models.LetterResponse) (string, error) {
	templateName := "letter_motivation.html"
	if letter.Type == models.LetterTypeAntiMotivation {
		templateName = "letter_anti_motivation.html"
	}

	var buf strings.Builder
	data := struct {
		Content     string
		CompanyName string
		Date        string
		Type        string
	}{
		Content:     letter.Content,
		CompanyName: letter.CompanyInfo.Name,
		Date:        letter.GeneratedAt.Format("02 January 2006"),
		Type:        string(letter.Type),
	}

	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// htmlToPDF : convertit HTML en PDF via Chrome headless
func (s *PDFLetterService) htmlToPDF(ctx context.Context, html string, writer io.Writer) error {
	// Create chromedp context
	allocCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Timeout protection
	allocCtx, cancel = context.WithTimeout(allocCtx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte

	// Execute chromedp tasks
	err := chromedp.Run(allocCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Set HTML content
			return chromedp.Run(ctx,
				chromedp.Evaluate(fmt.Sprintf(`
					document.open();
					document.write(%s);
					document.close();
				`, escapeJSString(html)), nil),
			)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Wait for page load
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Print to PDF
			return chromedp.PrintToPDF(&pdfBuf,
				chromedp.PrintToPDFParams{
					PrintBackground:     true,
					Scale:               1.0,
					PaperWidth:          8.27,  // A4 width in inches
					PaperHeight:         11.69, // A4 height in inches
					MarginTop:           0.4,
					MarginBottom:        0.4,
					MarginLeft:          0.4,
					MarginRight:         0.4,
					PreferCSSPageSize:   false,
					DisplayHeaderFooter: false,
				},
			).Do(ctx)
		}),
	)

	if err != nil {
		return fmt.Errorf("chromedp error: %w", err)
	}

	// Write PDF to output
	_, err = writer.Write(pdfBuf)
	return err
}

// escapeJSString : escape string pour JS
func escapeJSString(s string) string {
	// Simple JSON encoding is safe for JS
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return "`" + s + "`"
}
```

**Templates HTML :**

**Fichier :** `backend/templates/letter_motivation.html`

```html
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lettre de Motivation - {{.CompanyName}}</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            font-size: 11pt;
            line-height: 1.6;
            color: #1a1a1a;
            padding: 40px 60px;
            max-width: 800px;
            margin: 0 auto;
        }

        .header {
            margin-bottom: 40px;
            padding-bottom: 20px;
            border-bottom: 2px solid #3b82f6;
        }

        .header h1 {
            font-size: 18pt;
            font-weight: 700;
            color: #3b82f6;
            margin-bottom: 8px;
        }

        .header .date {
            font-size: 10pt;
            color: #6b7280;
        }

        .content {
            white-space: pre-wrap;
            text-align: justify;
        }

        .content p {
            margin-bottom: 16px;
        }

        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #e5e7eb;
            text-align: center;
            font-size: 9pt;
            color: #9ca3af;
        }

        @media print {
            body {
                padding: 0;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Lettre de Motivation</h1>
        <div class="date">{{.Date}}</div>
        <div class="company">À l'attention de {{.CompanyName}}</div>
    </div>

    <div class="content">
        {{.Content}}
    </div>

    <div class="footer">
        Générée automatiquement par maicivy - CV Intelligent
    </div>
</body>
</html>
```

**Fichier :** `backend/templates/letter_anti_motivation.html`

```html
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lettre d'Anti-Motivation - {{.CompanyName}}</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Comic+Neue:wght@400;700&display=swap');

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Comic Neue', cursive, sans-serif;
            font-size: 11pt;
            line-height: 1.7;
            color: #1a1a1a;
            padding: 40px 60px;
            max-width: 800px;
            margin: 0 auto;
            background: linear-gradient(135deg, #fef3c7 0%, #fca5a5 100%);
        }

        .header {
            margin-bottom: 30px;
            padding: 20px;
            background: white;
            border-radius: 12px;
            border: 3px dashed #ef4444;
            text-align: center;
        }

        .header h1 {
            font-size: 20pt;
            font-weight: 700;
            color: #dc2626;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 2px;
        }

        .header .warning {
            font-size: 12pt;
            color: #991b1b;
            font-weight: 600;
        }

        .content {
            background: white;
            padding: 30px;
            border-radius: 12px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            white-space: pre-wrap;
        }

        .content p {
            margin-bottom: 16px;
        }

        .footer {
            margin-top: 30px;
            padding: 15px;
            background: white;
            border-radius: 8px;
            text-align: center;
            font-size: 9pt;
            color: #6b7280;
            border: 2px solid #fbbf24;
        }

        @media print {
            body {
                padding: 0;
                background: white;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>⚠️ Lettre d'Anti-Motivation ⚠️</h1>
        <div class="warning">Humour et second degré - Ne pas prendre au sérieux !</div>
        <div class="date" style="margin-top: 10px; font-size: 10pt;">{{.Date}}</div>
    </div>

    <div class="content">
        {{.Content}}
    </div>

    <div class="footer">
        Générée par l'IA avec beaucoup d'humour 🤖<br>
        maicivy - Parce que l'auto-dérision, c'est la vie
    </div>
</body>
</html>
```

**Explications :**
- **chromedp** : Chrome headless pour rendu professionnel
- **Templates HTML** : designs distincts pour chaque type
- **A4 dimensions** : format standard impression
- **Fonts web** : typographie soignée (Inter pro, Comic Neue fun)
- **Styles CSS** : motivation pro (bleu), anti-motivation fun (rouge/jaune)

---

### Étape 6 : Orchestrateur de Génération de Lettres

**Description :** Service high-level qui orchestre scraping + IA + PDF.

**Fichier :** `backend/internal/services/letter_generator.go`

```go
package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type LetterGenerator struct {
	aiService     *AIService
	scraper       *CompanyScraper
	pdfService    *PDFLetterService
	promptBuilder *PromptBuilder
	log           *logger.Logger
}

func NewLetterGenerator(
	ai *AIService,
	scraper *CompanyScraper,
	pdf *PDFLetterService,
	profile models.UserProfile,
	log *logger.Logger,
) *LetterGenerator {
	return &LetterGenerator{
		aiService:     ai,
		scraper:       scraper,
		pdfService:    pdf,
		promptBuilder: NewPromptBuilder(profile),
		log:           log,
	}
}

// GenerateLetter : génère une lettre complète (IA)
func (lg *LetterGenerator) GenerateLetter(ctx context.Context, req models.LetterRequest) (*models.LetterResponse, error) {
	lg.log.Info("Starting letter generation", "company", req.CompanyName, "type", req.LetterType)

	// 1. Get company info via scraper
	companyInfo, err := lg.scraper.GetCompanyInfo(ctx, req.CompanyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get company info: %w", err)
	}

	// 2. Build prompt based on type
	var prompt string
	switch req.LetterType {
	case models.LetterTypeMotivation:
		prompt = lg.promptBuilder.BuildMotivationPrompt(*companyInfo)
	case models.LetterTypeAntiMotivation:
		prompt = lg.promptBuilder.BuildAntiMotivationPrompt(*companyInfo)
	default:
		return nil, fmt.Errorf("unknown letter type: %s", req.LetterType)
	}

	// 3. Generate text via AI
	content, metrics, err := lg.aiService.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 4. Build response
	response := &models.LetterResponse{
		Content:       content,
		Type:          req.LetterType,
		CompanyInfo:   *companyInfo,
		GeneratedAt:   time.Now(),
		Provider:      metrics.Provider,
		TokensUsed:    metrics.TotalTokens,
		EstimatedCost: metrics.EstimatedCost,
	}

	lg.log.Info("Letter generated successfully",
		"company", req.CompanyName,
		"type", req.LetterType,
		"tokens", metrics.TotalTokens,
		"cost", metrics.EstimatedCost,
	)

	return response, nil
}

// GenerateDualLetters : génère les 2 lettres en parallèle
func (lg *LetterGenerator) GenerateDualLetters(ctx context.Context, companyName string) (*models.LetterResponse, *models.LetterResponse, error) {
	type result struct {
		letter *models.LetterResponse
		err    error
	}

	motivationChan := make(chan result, 1)
	antiMotivationChan := make(chan result, 1)

	// Generate motivation letter
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeMotivation,
		})
		motivationChan <- result{letter, err}
	}()

	// Generate anti-motivation letter
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeAntiMotivation,
		})
		antiMotivationChan <- result{letter, err}
	}()

	// Wait for both
	motivationResult := <-motivationChan
	antiMotivationResult := <-antiMotivationChan

	if motivationResult.err != nil {
		return nil, nil, fmt.Errorf("motivation letter failed: %w", motivationResult.err)
	}
	if antiMotivationResult.err != nil {
		return nil, nil, fmt.Errorf("anti-motivation letter failed: %w", antiMotivationResult.err)
	}

	return motivationResult.letter, antiMotivationResult.letter, nil
}

// GenerateLetterPDF : génère le PDF d'une lettre
func (lg *LetterGenerator) GenerateLetterPDF(ctx context.Context, letter models.LetterResponse, writer io.Writer) error {
	return lg.pdfService.GeneratePDF(ctx, letter, writer)
}
```

**Explications :**
- **Orchestration complète** : scraping → prompt → IA → réponse
- **Dual generation** : parallélisation des 2 lettres
- **Gestion erreurs** : propagation claire des erreurs
- **Logging détaillé** : tracking coûts et durée

---

## 🧪 Tests

### Tests Unitaires

**Fichier :** `backend/internal/services/ai_test.go`

```go
package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"maicivy/internal/config"
	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type MockMetricsRecorder struct {
	mock.Mock
}

func (m *MockMetricsRecorder) RecordAIMetrics(metrics models.AIMetrics) {
	m.Called(metrics)
}

func TestAIService_CalculateCosts(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey: "test-key",
		ClaudeModel:     "claude-3-5-sonnet-20241022",
	}

	mockMetrics := new(MockMetricsRecorder)
	log := logger.NewLogger("test")

	svc, err := NewAIService(cfg, log, mockMetrics)
	assert.NoError(t, err)

	// Test Claude cost calculation
	cost := svc.calculateClaudeCost(1000, 2000) // 1k input, 2k output
	expectedCost := (1000.0/1_000_000)*3.0 + (2000.0/1_000_000)*15.0
	assert.InDelta(t, expectedCost, cost, 0.0001)

	// Test OpenAI cost calculation
	costGPT := svc.calculateOpenAICost(1000, 2000)
	expectedCostGPT := (1000.0/1_000_000)*10.0 + (2000.0/1_000_000)*30.0
	assert.InDelta(t, expectedCostGPT, costGPT, 0.0001)
}

func TestPromptBuilder_BuildMotivationPrompt(t *testing.T) {
	profile := models.UserProfile{
		Name:        "John Doe",
		CurrentRole: "Senior Go Developer",
		Skills:      []string{"Go", "PostgreSQL", "Docker"},
		Experience:  5,
	}

	company := models.CompanyInfo{
		Name:        "TechCorp",
		Industry:    "Software",
		Description: "Leading tech company",
		Technologies: []string{"Go", "Kubernetes"},
	}

	pb := NewPromptBuilder(profile)
	prompt := pb.BuildMotivationPrompt(company)

	// Verify all data is included
	assert.Contains(t, prompt, "John Doe")
	assert.Contains(t, prompt, "Senior Go Developer")
	assert.Contains(t, prompt, "TechCorp")
	assert.Contains(t, prompt, "Go, PostgreSQL, Docker")
}
```

### Tests Integration

**Fichier :** `backend/internal/services/scraper_test.go`

```go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"maicivy/internal/config"
	"maicivy/pkg/logger"
)

func TestCompanyScraper_CacheHit(t *testing.T) {
	// Setup mini redis
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := &config.ScraperConfig{
		CacheTTL: 1 * time.Hour,
		Timeout:  10 * time.Second,
	}

	log := logger.NewLogger("test")
	scraper := NewCompanyScraper(cfg, redisClient, log)

	// Pre-populate cache
	ctx := context.Background()
	cacheKey := "company_info:google"
	cacheData := `{"name":"Google","domain":"google.com","description":"Search engine"}`
	redisClient.Set(ctx, cacheKey, cacheData, cfg.CacheTTL)

	// Test cache hit
	info, err := scraper.GetCompanyInfo(ctx, "Google")
	assert.NoError(t, err)
	assert.Equal(t, "Google", info.Name)
	assert.Equal(t, "google.com", info.Domain)
}
```

### Commandes

```bash
# Run all tests
go test -v ./internal/services/...

# Run with coverage
go test -v -cover ./internal/services/...

# Run specific test
go test -v -run TestAIService_CalculateCosts ./internal/services/

# Benchmark AI service
go test -bench=. -benchmem ./internal/services/
```

---

## ⚠️ Points d'Attention

### Sécurité

- **API Keys** : ⚠️ JAMAIS commiter les clés dans Git
  - Utiliser `.env` avec `.gitignore`
  - Variables d'environnement en production

- **Input Validation** : ⚠️ Valider nom entreprise
  - Limiter longueur (max 100 chars)
  - Bloquer caractères suspects (scripts)
  - Rate limiting strict (5 req/jour)

- **Scraping Legal** : ⚠️ Respecter robots.txt
  - User-Agent identifiable
  - Rate limiting du scraping
  - Préférer APIs officielles

### Performance

- **Coûts IA** : 💡 Optimiser pour réduire coûts
  - Cache Redis (24h) : éviter régénération même lettre
  - Limiter max tokens (4000 suffisant)
  - Fallback GPT-4 seulement si Claude échoue (Claude moins cher)

- **Timeout chromedp** : ⚠️ Chrome peut être lent
  - Timeout 30s recommandé
  - Pool de contexts Chrome si volume élevé
  - Considérer alternatives (gofpdf) si PDF simple suffisant

- **Concurrency** : 💡 Génération parallèle des 2 lettres
  - Divise temps par ~2
  - Attention : double coût API si les deux réussissent

### Edge Cases

- **Entreprise inconnue** : ⚠️ Si scraping échoue
  - Description générique : "Entreprise située à [domain]"
  - IA peut toujours générer lettre (moins personnalisée)

- **Langues** : ⚠️ Prompts FR uniquement actuellement
  - Ajouter détection langue + prompts EN si international

- **Rate Limits API** : ⚠️ Claude/OpenAI ont des limites
  - Gérer erreur 429 (Too Many Requests)
  - Backoff exponentiel implémenté
  - Circuit breaker recommandé si abuse

### Monitoring

- **Prometheus Metrics** : 💡 À ajouter
  ```go
  // Example metrics
  aiRequestsTotal := prometheus.NewCounterVec(
      prometheus.CounterOpts{
          Name: "ai_requests_total",
          Help: "Total AI requests by provider and status",
      },
      []string{"provider", "status"},
  )

  aiCostTotal := prometheus.NewCounter(
      prometheus.CounterOpts{
          Name: "ai_cost_total_usd",
          Help: "Total estimated AI cost in USD",
      },
  )
  ```

---

## 📚 Ressources

### Documentation Officielle

- **Anthropic Claude API** : https://docs.anthropic.com/claude/reference/messages_post
  - Guide prompts : https://docs.anthropic.com/claude/docs/prompt-engineering

- **OpenAI GPT-4 API** : https://platform.openai.com/docs/api-reference/chat
  - Best practices : https://platform.openai.com/docs/guides/prompt-engineering

- **chromedp** : https://github.com/chromedp/chromedp
  - Examples : https://github.com/chromedp/examples

- **colly (scraping)** : https://go-colly.org/docs/

### Articles Techniques

- **Prompt Engineering Guide** : https://www.promptingguide.ai/
- **Cost Optimization for LLMs** : https://huggingface.co/blog/optimize-llm-cost
- **Go Retry Patterns** : https://encore.dev/blog/retries

### Librairies Alternatives

- **gofpdf** (si chromedp trop lourd) : https://github.com/jung-kurt/gofpdf
- **rod** (alternative chromedp) : https://github.com/go-rod/rod

---

## ✅ Checklist de Complétion

### Implémentation
- [ ] Configuration AI + Scraper (`config/ai.go`, `config/scraper.go`)
- [ ] Models définis (`models/ai.go`)
- [ ] Service Scraper (`services/scraper.go`)
  - [ ] Cache Redis
  - [ ] API Clearbit
  - [ ] Fallback scraping web
- [ ] Prompts Engineering (`services/prompts.go`)
  - [ ] Prompt motivation
  - [ ] Prompt anti-motivation
- [ ] Service IA (`services/ai.go`)
  - [ ] Client Claude
  - [ ] Client OpenAI
  - [ ] Fallback strategy
  - [ ] Retry logic
  - [ ] Cost tracking
- [ ] Service PDF (`services/pdf_letters.go`)
  - [ ] chromedp setup
  - [ ] Templates HTML (motivation + anti-motivation)
- [ ] Orchestrateur (`services/letter_generator.go`)
  - [ ] Génération simple
  - [ ] Génération dual parallèle

### Tests
- [ ] Tests unitaires (`*_test.go`)
  - [ ] AIService (cost calculation)
  - [ ] PromptBuilder (template rendering)
  - [ ] Scraper (cache, fallback)
- [ ] Tests integration
  - [ ] End-to-end génération lettre (mock API)
  - [ ] PDF generation
- [ ] Coverage > 70%

### Documentation
- [ ] Commentaires GoDoc sur fonctions publiques
- [ ] README usage examples
- [ ] `.env.example` avec toutes les variables

### Sécurité
- [ ] Validation inputs (company name)
- [ ] Secrets dans `.env` (pas hardcodés)
- [ ] Rate limiting configuré
- [ ] Timeout sur toutes requêtes externes

### Performance
- [ ] Cache Redis activé (TTL 24h lettres, 7j entreprises)
- [ ] Métriques Prometheus ajoutées
- [ ] Benchmarks exécutés

### Déploiement
- [ ] Variables d'environnement documentées
- [ ] Health check endpoint intégré
- [ ] Logs structurés (JSON)
- [ ] Docker build testé

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
**Version:** 1.0
