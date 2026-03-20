package config

import (
	"os"
	"strconv"
	"time"
)

type AIConfig struct {
	// Providers
	AnthropicAPIKey  string
	AnthropicBaseURL string // Proxy URL (ex: https://ai.etheryale.com)
	OpenAIAPIKey     string
	PrimaryProvider  string // "claude" ou "openai"
	OwnerAPIKey      string // Clé secrète owner → tier premium (Opus)

	// Models
	ClaudeModel string
	OpenAIModel string

	// Rate Limiting
	MaxRequestsPerMinute int
	MaxTokensPerRequest  int

	// Retry Strategy
	MaxRetries     int
	RetryBaseDelay time.Duration
	RequestTimeout time.Duration

	// Cost Tracking
	EnableCostTracking bool
}

func LoadAIConfig() *AIConfig {
	return &AIConfig{
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicBaseURL:     os.Getenv("ANTHROPIC_BASE_URL"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		PrimaryProvider:      getEnvOrDefault("AI_PRIMARY_PROVIDER", "claude"),
		OwnerAPIKey:          os.Getenv("MAICIVY_OWNER_API_KEY"),
		ClaudeModel:          getEnvOrDefault("CLAUDE_MODEL", "claude-haiku-4-5-20251001"), // Haiku par défaut pour les visiteurs
		OpenAIModel:          getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
		MaxRequestsPerMinute: getEnvAsIntOrDefault("AI_MAX_REQUESTS_PER_MIN", 10),
		MaxTokensPerRequest:  getEnvAsIntOrDefault("AI_MAX_TOKENS", 4000),
		MaxRetries:           getEnvAsIntOrDefault("AI_MAX_RETRIES", 3),
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

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}
