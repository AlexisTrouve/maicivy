package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

// GetCompanyInfo : point d'entrée principal - multi-sources pour résilience
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

	// 2. Lancer toutes les sources en parallèle
	info := &models.CompanyInfo{
		Name:   companyName,
		Domain: s.guessDomainFromName(companyName),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Source 1: Wikipedia
	wg.Add(1)
	go func() {
		defer wg.Done()
		wikiInfo, err := s.fetchFromWikipedia(ctx, companyName)
		if err != nil {
			log.Debug().Err(err).Str("company", companyName).Msg("Wikipedia fetch failed")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if info.Description == "" && wikiInfo.Description != "" {
			info.Description = wikiInfo.Description
			log.Info().Str("company", companyName).Str("source", "wikipedia").Msg("Got description")
		}
		if info.Industry == "" && wikiInfo.Industry != "" {
			info.Industry = wikiInfo.Industry
		}
		if info.Size == "" && wikiInfo.Size != "" {
			info.Size = wikiInfo.Size
		}
	}()

	// Source 2: DuckDuckGo
	wg.Add(1)
	go func() {
		defer wg.Done()
		ddgDesc, err := s.fetchFromDuckDuckGo(ctx, companyName)
		if err != nil {
			log.Debug().Err(err).Str("company", companyName).Msg("DuckDuckGo fetch failed")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if info.Description == "" && ddgDesc != "" {
			info.Description = ddgDesc
			log.Info().Str("company", companyName).Str("source", "duckduckgo").Msg("Got description")
		}
	}()

	// Source 3: Scraping website
	wg.Add(1)
	go func() {
		defer wg.Done()
		scrapedInfo, err := s.scrapeCompanyWebsite(ctx, companyName)
		if err != nil {
			log.Debug().Err(err).Str("company", companyName).Msg("Website scraping failed")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if info.Description == "" && scrapedInfo.Description != "" {
			info.Description = scrapedInfo.Description
			log.Info().Str("company", companyName).Str("source", "scraping").Msg("Got description")
		}
		if len(info.Technologies) == 0 && len(scrapedInfo.Technologies) > 0 {
			info.Technologies = scrapedInfo.Technologies
		}
	}()

	// Source 4: GitHub - projets open-source
	wg.Add(1)
	go func() {
		defer wg.Done()
		repos, err := s.fetchFromGitHub(ctx, companyName)
		if err != nil {
			log.Debug().Err(err).Str("company", companyName).Msg("GitHub fetch failed")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if repos != "" {
			info.RecentNews = repos // On utilise RecentNews pour les projets GitHub
			log.Info().Str("company", companyName).Str("source", "github").Msg("Got open-source projects")
		}
	}()

	// Source 5: Blog/Newsroom RSS
	wg.Add(1)
	go func() {
		defer wg.Done()
		news, err := s.fetchRecentNews(ctx, companyName)
		if err != nil {
			log.Debug().Err(err).Str("company", companyName).Msg("News fetch failed")
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if news != "" {
			if info.RecentNews != "" {
				info.RecentNews += "\n\n" + news
			} else {
				info.RecentNews = news
			}
			log.Info().Str("company", companyName).Str("source", "news").Msg("Got recent news")
		}
	}()

	// Source 6: Clearbit API (si clé dispo)
	if s.config.ClearbitAPIKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clearbitInfo, err := s.fetchFromClearbit(ctx, companyName)
			if err != nil {
				log.Debug().Err(err).Str("company", companyName).Msg("Clearbit fetch failed")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if len(clearbitInfo.Technologies) > 0 {
				info.Technologies = clearbitInfo.Technologies
			}
			if info.Size == "" && clearbitInfo.Size != "" {
				info.Size = clearbitInfo.Size
			}
			if info.Industry == "" && clearbitInfo.Industry != "" {
				info.Industry = clearbitInfo.Industry
			}
			log.Info().Str("company", companyName).Str("source", "clearbit").Msg("Got enrichment data")
		}()
	}

	// Attendre toutes les sources (avec timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Toutes les sources ont répondu
	case <-time.After(12 * time.Second):
		log.Warn().Str("company", companyName).Msg("Timeout waiting for all sources")
	}

	// 3. Fallback minimal si tout échoue
	if info.Description == "" {
		info.Description = fmt.Sprintf("%s est une entreprise.", companyName)
	}

	// 4. Cache résultat (7 jours)
	data, _ := json.Marshal(info)
	s.redisClient.Set(ctx, cacheKey, data, s.config.CacheTTL)

	log.Info().
		Str("company", companyName).
		Bool("has_description", info.Description != "").
		Bool("has_industry", info.Industry != "").
		Bool("has_size", info.Size != "").
		Bool("has_news", info.RecentNews != "").
		Int("tech_count", len(info.Technologies)).
		Msg("Company info aggregated from multiple sources")

	return info, nil
}

// fetchFromGitHub récupère les repos open-source populaires
func (s *CompanyScraper) fetchFromGitHub(ctx context.Context, companyName string) (string, error) {
	// GitHub API - recherche des repos de l'organisation
	orgName := strings.ToLower(strings.ReplaceAll(companyName, " ", ""))

	// Mapping des noms d'entreprises vers leurs orgs GitHub
	githubOrgs := map[string]string{
		"stripe": "stripe", "microsoft": "microsoft", "google": "google",
		"facebook": "facebook", "meta": "facebook", "netflix": "netflix",
		"uber": "uber", "airbnb": "airbnb", "twitter": "twitter",
		"spotify": "spotify", "slack": "slackhq", "datadog": "datadog",
		"mongodb": "mongodb", "elastic": "elastic", "cloudflare": "cloudflare",
		"vercel": "vercel", "supabase": "supabase", "hashicorp": "hashicorp",
	}

	if org, ok := githubOrgs[orgName]; ok {
		orgName = org
	}

	apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos?sort=stars&per_page=5", orgName)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %d", resp.StatusCode)
	}

	var repos []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Stars       int    `json:"stargazers_count"`
		UpdatedAt   string `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return "", err
	}

	if len(repos) == 0 {
		return "", fmt.Errorf("no repos found")
	}

	// Formater les projets
	var sb strings.Builder
	sb.WriteString("Projets open-source actifs:\n")
	for i, repo := range repos {
		if i >= 5 {
			break
		}
		desc := repo.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("• %s (%s, %d★): %s\n", repo.Name, repo.Language, repo.Stars, desc))
	}

	return sb.String(), nil
}

// fetchRecentNews récupère les actualités récentes via recherche
func (s *CompanyScraper) fetchRecentNews(ctx context.Context, companyName string) (string, error) {
	// Utiliser DuckDuckGo News ou scraper le newsroom
	domain := s.guessDomainFromName(companyName)

	// Essayer de scraper le blog/newsroom
	newsURLs := []string{
		fmt.Sprintf("https://%s/blog", domain),
		fmt.Sprintf("https://blog.%s", domain),
		fmt.Sprintf("https://%s/newsroom", domain),
		fmt.Sprintf("https://news.%s", domain),
	}

	var titles []string

	for _, newsURL := range newsURLs {
		c := colly.NewCollector(
			colly.UserAgent(s.config.UserAgent),
			colly.MaxDepth(1),
		)

		c.OnHTML("article h2, article h3, .post-title, .blog-title, h2.title, h3.title", func(e *colly.HTMLElement) {
			title := strings.TrimSpace(e.Text)
			if len(title) > 10 && len(title) < 200 {
				titles = append(titles, title)
			}
		})

		c.OnHTML("h1 a, h2 a, h3 a", func(e *colly.HTMLElement) {
			title := strings.TrimSpace(e.Text)
			if len(title) > 10 && len(title) < 200 {
				titles = append(titles, title)
			}
		})

		_ = c.Visit(newsURL)

		if len(titles) >= 3 {
			break
		}
	}

	if len(titles) == 0 {
		return "", fmt.Errorf("no news found")
	}

	// Dédupliquer et formater
	uniqueTitles := uniqueStrings(titles)
	if len(uniqueTitles) > 5 {
		uniqueTitles = uniqueTitles[:5]
	}

	var sb strings.Builder
	sb.WriteString("Actualités récentes:\n")
	for _, title := range uniqueTitles {
		sb.WriteString(fmt.Sprintf("• %s\n", title))
	}

	return sb.String(), nil
}

// fetchFromWikipedia récupère les infos depuis l'API Wikipedia
func (s *CompanyScraper) fetchFromWikipedia(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	// Essayer plusieurs variantes du nom
	variants := []string{
		companyName,
		companyName + "_(company)",
		companyName + "_(entreprise)",
		companyName + "_(software)",
	}

	for _, variant := range variants {
		searchURL := fmt.Sprintf(
			"https://en.wikipedia.org/api/rest_v1/page/summary/%s",
			url.PathEscape(variant),
		)

		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", s.config.UserAgent)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var wikiData struct {
				Title       string `json:"title"`
				Extract     string `json:"extract"`
				Description string `json:"description"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&wikiData); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if wikiData.Extract != "" {
				info := &models.CompanyInfo{
					Name:        wikiData.Title,
					Description: wikiData.Extract,
					Industry:    extractIndustryFromText(wikiData.Extract),
					Size:        extractSizeFromText(wikiData.Extract),
				}
				return info, nil
			}
		}
		resp.Body.Close()
	}

	return nil, fmt.Errorf("no wikipedia article found for %s", companyName)
}

// fetchFromDuckDuckGo récupère un résumé via DuckDuckGo Instant Answer
func (s *CompanyScraper) fetchFromDuckDuckGo(ctx context.Context, companyName string) (string, error) {
	ddgURL := fmt.Sprintf(
		"https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(companyName+" company"),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", ddgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", s.config.UserAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ddgData struct {
		Abstract     string `json:"Abstract"`
		AbstractText string `json:"AbstractText"`
		Heading      string `json:"Heading"`
	}

	if err := json.Unmarshal(body, &ddgData); err != nil {
		return "", err
	}

	if ddgData.AbstractText != "" {
		return ddgData.AbstractText, nil
	}
	if ddgData.Abstract != "" {
		return ddgData.Abstract, nil
	}
	return "", fmt.Errorf("no duckduckgo result")
}

// scrapeCompanyWebsite : scraping du site officiel
func (s *CompanyScraper) scrapeCompanyWebsite(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	domain := s.guessDomainFromName(companyName)

	info := &models.CompanyInfo{
		Name:   companyName,
		Domain: domain,
	}

	c := colly.NewCollector(
		colly.UserAgent(s.config.UserAgent),
		colly.AllowedDomains(domain, "www."+domain),
		colly.MaxDepth(1),
	)

	// Meta description
	c.OnHTML("meta[name=description]", func(e *colly.HTMLElement) {
		if info.Description == "" {
			content := e.Attr("content")
			if len(content) > 50 {
				info.Description = content
			}
		}
	})

	// OG description (souvent meilleure)
	c.OnHTML("meta[property='og:description']", func(e *colly.HTMLElement) {
		if info.Description == "" {
			content := e.Attr("content")
			if len(content) > 50 {
				info.Description = content
			}
		}
	})

	// Twitter description
	c.OnHTML("meta[name='twitter:description']", func(e *colly.HTMLElement) {
		if info.Description == "" {
			content := e.Attr("content")
			if len(content) > 50 {
				info.Description = content
			}
		}
	})

	// Détecter les technos via scripts et meta
	var detectedTechs []string
	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		src := strings.ToLower(e.Attr("src"))
		if strings.Contains(src, "react") {
			detectedTechs = append(detectedTechs, "React")
		}
		if strings.Contains(src, "angular") {
			detectedTechs = append(detectedTechs, "Angular")
		}
		if strings.Contains(src, "vue") {
			detectedTechs = append(detectedTechs, "Vue.js")
		}
		if strings.Contains(src, "jquery") {
			detectedTechs = append(detectedTechs, "jQuery")
		}
	})

	c.OnScraped(func(r *colly.Response) {
		if len(detectedTechs) > 0 {
			info.Technologies = uniqueStrings(detectedTechs)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Debug().Err(err).Str("url", r.Request.URL.String()).Msg("Scraping error")
	})

	// Visiter le site
	urls := []string{
		fmt.Sprintf("https://www.%s", domain),
		fmt.Sprintf("https://%s", domain),
	}

	for _, siteURL := range urls {
		err := c.Visit(siteURL)
		if err == nil {
			break
		}
	}

	if info.Description == "" && len(info.Technologies) == 0 {
		return nil, fmt.Errorf("no useful data scraped from %s", domain)
	}

	return info, nil
}

// fetchFromClearbit : enrichissement via Clearbit API
func (s *CompanyScraper) fetchFromClearbit(ctx context.Context, companyName string) (*models.CompanyInfo, error) {
	if s.config.ClearbitAPIKey == "" {
		return nil, fmt.Errorf("no clearbit API key")
	}

	domain := s.guessDomainFromName(companyName)
	apiURL := fmt.Sprintf("https://company.clearbit.com/v2/companies/find?domain=%s", domain)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
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
		return nil, fmt.Errorf("clearbit returned %d", resp.StatusCode)
	}

	var data struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &models.CompanyInfo{
		Name:         data.Name,
		Domain:       data.Domain,
		Description:  data.Description,
		Industry:     data.Category.Industry,
		Size:         data.Metrics.Employees,
		Technologies: data.Tech,
	}, nil
}

// guessDomainFromName devine le domaine depuis le nom
func (s *CompanyScraper) guessDomainFromName(name string) string {
	domain := strings.ToLower(name)
	domain = strings.ReplaceAll(domain, " ", "")
	domain = strings.ReplaceAll(domain, ".", "")

	// Cas spéciaux connus
	knownDomains := map[string]string{
		"google": "google.com", "microsoft": "microsoft.com", "apple": "apple.com",
		"amazon": "amazon.com", "meta": "meta.com", "facebook": "facebook.com",
		"netflix": "netflix.com", "tesla": "tesla.com", "nvidia": "nvidia.com",
		"intel": "intel.com", "ibm": "ibm.com", "oracle": "oracle.com",
		"salesforce": "salesforce.com", "adobe": "adobe.com", "spotify": "spotify.com",
		"airbnb": "airbnb.com", "uber": "uber.com", "twitter": "twitter.com",
		"linkedin": "linkedin.com", "datadog": "datadoghq.com", "stripe": "stripe.com",
		"slack": "slack.com", "zoom": "zoom.us", "shopify": "shopify.com",
		"twilio": "twilio.com", "snowflake": "snowflake.com", "mongodb": "mongodb.com",
		"elastic": "elastic.co", "cloudflare": "cloudflare.com", "atlassian": "atlassian.com",
		"github": "github.com", "gitlab": "gitlab.com", "docker": "docker.com",
	}

	if known, ok := knownDomains[domain]; ok {
		return known
	}

	return domain + ".com"
}

// extractIndustryFromText extrait l'industrie depuis un texte
func extractIndustryFromText(text string) string {
	text = strings.ToLower(text)

	industries := map[string]string{
		"software":            "Technology / Software",
		"technology company":  "Technology",
		"tech company":        "Technology",
		"e-commerce":          "E-commerce",
		"ecommerce":           "E-commerce",
		"online retail":       "E-commerce",
		"financial":           "Finance",
		"fintech":             "Fintech",
		"bank":                "Banking",
		"automotive":          "Automotive",
		"electric vehicle":    "Automotive / EV",
		"streaming":           "Entertainment / Streaming",
		"social media":        "Social Media",
		"social network":      "Social Media",
		"cloud computing":     "Cloud Computing",
		"cloud infrastructure": "Cloud Computing",
		"semiconductor":       "Semiconductors",
		"chip":                "Semiconductors",
		"pharmaceutical":      "Pharmaceuticals",
		"healthcare":          "Healthcare",
		"gaming":              "Gaming",
		"video game":          "Gaming",
		"music streaming":     "Entertainment / Music",
		"payment":             "Fintech / Payments",
		"consulting":          "Consulting",
		"manufacturing":       "Manufacturing",
		"energy":              "Energy",
		"renewable":           "Energy / Renewables",
		"aerospace":           "Aerospace",
		"defense":             "Defense",
		"telecommunications":  "Telecommunications",
		"advertising":         "Advertising",
		"artificial intelligence": "Technology / AI",
		"machine learning":    "Technology / AI",
		"cybersecurity":       "Technology / Cybersecurity",
		"saas":                "Technology / SaaS",
		"enterprise software": "Technology / Enterprise",
		"database":            "Technology / Data",
		"data platform":       "Technology / Data",
		"observability":       "Technology / DevOps",
		"monitoring":          "Technology / DevOps",
	}

	for keyword, industry := range industries {
		if strings.Contains(text, keyword) {
			return industry
		}
	}
	return ""
}

// extractSizeFromText extrait la taille depuis un texte
func extractSizeFromText(text string) string {
	text = strings.ToLower(text)

	if strings.Contains(text, "million") && strings.Contains(text, "employees") {
		return "1,000,000+ employees"
	}
	if strings.Contains(text, "hundreds of thousands") {
		return "100,000+ employees"
	}
	if strings.Contains(text, "100,000") || strings.Contains(text, "100000") {
		return "100,000+ employees"
	}
	if strings.Contains(text, "50,000") || strings.Contains(text, "50000") {
		return "50,000+ employees"
	}
	if strings.Contains(text, "10,000") || strings.Contains(text, "10000") {
		return "10,000+ employees"
	}
	if strings.Contains(text, "thousands of employees") {
		return "1,000+ employees"
	}
	if strings.Contains(text, "1,000") || strings.Contains(text, "1000") {
		return "1,000+ employees"
	}
	return ""
}

// uniqueStrings retourne une slice sans doublons
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
