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
	"github.com/rs/zerolog/log"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

type CompanyScraper struct {
	config      *config.ScraperConfig
	redisClient *redis.Client
	httpClient  *http.Client
}

func NewCompanyScraper(cfg *config.ScraperConfig, redis *redis.Client) *CompanyScraper {
	return &CompanyScraper{
		config:      cfg,
		redisClient: redis,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
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
			log.Info().Str("company", companyName).Msg("Company info found in cache")
			return &info, nil
		}
	}

	// 2. Tenter enrichissement via APIs
	info, err := s.enrichViaAPIs(ctx, companyName)
	if err != nil {
		log.Warn().Err(err).Msg("API enrichment failed, fallback to scraping")
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
		Name        string `json:"name"`
		Domain      string `json:"domain"`
		Description string `json:"description"`
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
		"google":     "google.com",
		"microsoft":  "microsoft.com",
		"apple":      "apple.com",
		"amazon":     "amazon.com",
		"meta":       "meta.com",
		"facebook":   "facebook.com",
		"netflix":    "netflix.com",
		"tesla":      "tesla.com",
		"nvidia":     "nvidia.com",
		"intel":      "intel.com",
		"ibm":        "ibm.com",
		"oracle":     "oracle.com",
		"salesforce": "salesforce.com",
		"adobe":      "adobe.com",
		"spotify":    "spotify.com",
		"airbnb":     "airbnb.com",
		"uber":       "uber.com",
		"twitter":    "twitter.com",
		"linkedin":   "linkedin.com",
	}

	if known, ok := knownDomains[domain]; ok {
		return known
	}

	return domain + ".com"
}
