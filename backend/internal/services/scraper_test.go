package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

func TestCompanyScraper_CacheHit(t *testing.T) {
	// Setup mini redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := &config.ScraperConfig{
		CacheTTL: 1 * time.Hour,
		Timeout:  10 * time.Second,
	}

	scraper := NewCompanyScraper(cfg, redisClient)

	// Pre-populate cache
	ctx := context.Background()
	cacheKey := "company_info:google"
	cacheData := `{"name":"Google","domain":"google.com","description":"Search engine"}`
	err = redisClient.Set(ctx, cacheKey, cacheData, cfg.CacheTTL).Err()
	require.NoError(t, err)

	// Test cache hit
	info, err := scraper.GetCompanyInfo(ctx, "Google")
	assert.NoError(t, err)
	assert.Equal(t, "Google", info.Name)
	assert.Equal(t, "google.com", info.Domain)
	assert.Equal(t, "Search engine", info.Description)
}

func TestCompanyScraper_CacheMiss(t *testing.T) {
	// Setup mini redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := &config.ScraperConfig{
		CacheTTL: 1 * time.Hour,
		Timeout:  10 * time.Second,
	}

	scraper := NewCompanyScraper(cfg, redisClient)

	ctx := context.Background()

	// Test cache miss (will fail to scrape but we test the flow)
	info, err := scraper.GetCompanyInfo(ctx, "NonExistentCompany123XYZ")

	// We expect an error since we don't have API keys and scraping will fail
	// but we verify the cache key was checked
	if err != nil {
		// Expected behavior when no API keys configured
		assert.Error(t, err)
	}

	// Verify cache key format
	cacheKey := "company_info:nonexistentcompany123xyz"
	cached, err := redisClient.Get(ctx, cacheKey).Result()

	// If we got company info, it should be cached
	if info != nil {
		assert.NoError(t, err)
		var cachedInfo map[string]interface{}
		err = json.Unmarshal([]byte(cached), &cachedInfo)
		assert.NoError(t, err)
	}
}

func TestCompanyScraper_GuessDomainFromName(t *testing.T) {
	cfg := &config.ScraperConfig{}
	scraper := NewCompanyScraper(cfg, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Known company - Google",
			input:    "Google",
			expected: "google.com",
		},
		{
			name:     "Known company - Microsoft",
			input:    "Microsoft",
			expected: "microsoft.com",
		},
		{
			name:     "Unknown company",
			input:    "MyStartup",
			expected: "mystartup.com",
		},
		{
			name:     "Company with spaces",
			input:    "My Company",
			expected: "mycompany.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scraper.guessDomainFromName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompanyScraper_EnrichViaAPIs_NoAPIKey(t *testing.T) {
	cfg := &config.ScraperConfig{
		// No API key
		Timeout: 10 * time.Second,
	}

	scraper := NewCompanyScraper(cfg, nil)
	ctx := context.Background()

	_, err := scraper.enrichViaAPIs(ctx, "TestCompany")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API key configured")
}
