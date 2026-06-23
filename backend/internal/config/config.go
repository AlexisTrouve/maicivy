package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Server
	ServerPort  string
	ServerHost  string
	Environment string

	// CORS
	AllowedOrigins []string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// API Keys
	ClaudeAPIKey string
	OpenAIAPIKey string
	// Proxy Anthropic (vendor proxy interne)
	AnthropicBaseURL string
	AnthropicAPIKey  string

	// Activity Feed (ProjectTracker)
	ActivityFeedURL string

	// Repos Scanner (path to git repos directory)
	ReposDir string

	// Gitea Stats (clé read-only séparée pour les stats Git du CV)
	GiteaStatsURL   string
	GiteaStatsToken string
	GiteaStatsUser  string

	// GitLab Stats (commits d'un projet partagé filtrés par auteur, mergés dans les gitstats).
	// Tout vient de l'ENV (rien en source) → token/projet hors du repo public.
	GitLabStatsURL     string
	GitLabStatsToken   string
	GitLabStatsProject string // ID ou path URL-encodé du projet GitLab
	GitLabStatsAuthors string // noms d'auteur à matcher, séparés par virgule (ex: "alexis,stillhammer")

	// Session : secret HMAC pour signer le cookie maicivy_session (anti-forge + anti-amplification PG)
	SessionSecret string

	// Admin : mot de passe du panneau /admin (login → cookie maicivy_admin signé = privilèges owner).
	// Vide → login admin désactivé (aucun mot de passe ne matche, le panneau est inatteignable).
	AdminPassword string
}

func Load() (*Config, error) {
	// Charger .env en développement (ignore si non présent)
	_ = godotenv.Load()

	cfg := &Config{
		// Server
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// CORS
		AllowedOrigins: getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),

		// PostgreSQL
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "maicivy"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "maicivy"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// API Keys
		ClaudeAPIKey:     getEnv("CLAUDE_API_KEY", ""),
		OpenAIAPIKey:     getEnv("OPENAI_API_KEY", ""),
		AnthropicBaseURL: getEnv("ANTHROPIC_BASE_URL", ""),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", "sk-internal-dev"),

		// Activity Feed
		ActivityFeedURL: getEnv("ACTIVITY_FEED_URL", ""),

		// Repos Scanner
		ReposDir: getEnv("REPOS_DIR", "/repos"),

		// Gitea Stats
		GiteaStatsURL:   getEnv("GITEA_STATS_URL", "https://git.etheryale.com"),
		GiteaStatsToken: getEnv("GITEA_STATS_TOKEN", ""),
		GiteaStatsUser:  getEnv("GITEA_STATS_USER", "StillHammer"),

		// GitLab Stats (vide par défaut = désactivé ; projet/token jamais en source, uniquement env VPS)
		GitLabStatsURL:     getEnv("GITLAB_STATS_URL", "https://gitlab.com"),
		GitLabStatsToken:   getEnv("GITLAB_STATS_TOKEN", ""),
		GitLabStatsProject: getEnv("GITLAB_STATS_PROJECT", ""),
		GitLabStatsAuthors: getEnv("GITLAB_STATS_AUTHORS", "alexis,stillhammer"),

		// Session signing
		SessionSecret: getEnv("SESSION_SECRET", ""),
		// Admin panel
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	// Validation minimale (Phase 1)
	if c.DBPassword == "" {
		log.Warn().Msg("DB_PASSWORD is empty (not recommended for production)")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	// Split by comma (ex: "http://localhost:3000,https://maicivy.com")
	result := []string{}
	for _, v := range splitString(valueStr, ",") {
		trimmed := trimString(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	current := ""
	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
