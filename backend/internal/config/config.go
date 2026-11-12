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
	// Passerelle Etheryale (proxy vendor interne) : seule route vers un modèle, tous fournisseurs
	// confondus. Le champ OpenAIAPIKey a été retiré avec le client OpenAI direct — plus rien ne le
	// lisait une fois le repli passé par la passerelle.
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
	// GitLabStatsRef : branche à compter (ex. "dev"). Vide = branche par défaut du projet.
	// La branche d'intégration n'est pas toujours la branche par défaut : sur le projet client tout arrive par
	// MR sur `dev` pendant que `main` suit les releases, et les deux divergent.
	GitLabStatsRef string

	// GiteaStatsExclude : dépôts du compte à écarter des stats publiques (noms séparés par des
	// virgules, sans casse). ENV seulement : un nom écrit ici serait publié par le miroir GitHub.
	GiteaStatsExclude string

	// Gitea tiers (dépôt partagé hébergé ailleurs — un jeu partenaire), mergé comme le GitLab. Tout vient
	// de l'ENV : le chemin du dépôt porte un nom qui ne doit pas apparaître dans le dépôt public.
	PartnerGiteaURL    string
	PartnerGiteaToken  string // jeton read:repository
	PartnerGiteaRepo   string // "owner/repo"
	PartnerGiteaRef    string // branche comptée ; vide = branche par défaut
	PartnerGiteaEmails string // emails d'auteur d'Alexis, séparés par virgule (égalité, sans casse)

	// Session : secret HMAC pour signer le cookie maicivy_session (anti-forge + anti-amplification PG)
	SessionSecret string

	// Admin : mot de passe du panneau /admin (login → cookie maicivy_admin signé = privilèges owner).
	// Vide → login admin désactivé (aucun mot de passe ne matche, le panneau est inatteignable).
	AdminPassword string

	// Compose : ancrage tarifaire de l'écran /admin/compose. POURQUOI en config et non en dur : sans
	// référence, le modèle invente un montant plausible — un nombre que l'opérateur ne peut ni juger
	// ni défendre face au client. En lui donnant le tarif réellement pratiqué et la cible visée, son
	// estimation devient une interpolation argumentable. Et ces deux valeurs bougent avec le
	// positionnement : les changer ne doit pas demander un rebuild.
	ComposeRateCurrent int // TJM effectivement facturé aujourd'hui, en euros
	ComposeRateTarget  int // TJM visé, en euros — borne haute de la fourchette proposée

	// ComposeRateFloor : plancher DUR du TJM, en euros, appliqué par le tool set_rate du Composer —
	// owner comme prospecteur. POURQUOI un plancher distinct de l'ancrage : l'ancrage oriente le
	// modèle, il ne l'empêche pas de descendre. Un prospecteur négocie librement, mais aucun message ne
	// doit engager Alexis sous ce seuil ; le refus est mécanique (erreur renvoyée au modèle), pas une
	// consigne qu'il pourrait ignorer.
	ComposeRateFloor int

	// ProspectDailyAIMax : nombre de générations IA (tours de Composer + CV) par prospecteur et par
	// jour. Un prospecteur n'est jamais owner : sans plafond, un compte compromis brûlerait le budget.
	// < 0 = levé (choix provisoire du 2026-10-15 : seuls le disjoncteur global et le verrou in-flight jouent).
	ProspectDailyAIMax int
	// ProspectVerifyDailyMax : plafond quotidien des vérifications Composer d'un prospecteur
	// (compteur distinct, cf. middleware.ProspectVerifyQuota). < 0 = levé, 0 = aucune.
	ProspectVerifyDailyMax int
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
		AnthropicBaseURL: getEnv("ANTHROPIC_BASE_URL", ""),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", "sk-internal-dev"),

		// Activity Feed
		ActivityFeedURL: getEnv("ACTIVITY_FEED_URL", ""),

		// Repos Scanner
		ReposDir: getEnv("REPOS_DIR", "/repos"),

		// Gitea Stats
		GiteaStatsURL:     getEnv("GITEA_STATS_URL", "https://git.etheryale.com"),
		GiteaStatsToken:   getEnv("GITEA_STATS_TOKEN", ""),
		GiteaStatsUser:    getEnv("GITEA_STATS_USER", "StillHammer"),
		GiteaStatsExclude: getEnv("GITEA_STATS_EXCLUDE", ""),

		// GitLab Stats (vide par défaut = désactivé ; projet/token jamais en source, uniquement env VPS)
		GitLabStatsURL:     getEnv("GITLAB_STATS_URL", "https://gitlab.com"),
		GitLabStatsToken:   getEnv("GITLAB_STATS_TOKEN", ""),
		GitLabStatsProject: getEnv("GITLAB_STATS_PROJECT", ""),
		GitLabStatsAuthors: getEnv("GITLAB_STATS_AUTHORS", "alexis,stillhammer"),
		GitLabStatsRef:     getEnv("GITLAB_STATS_REF", ""),

		// Gitea tiers (vide par défaut = désactivé)
		PartnerGiteaURL:    getEnv("PARTNER_GITEA_URL", ""),
		PartnerGiteaToken:  getEnv("PARTNER_GITEA_TOKEN", ""),
		PartnerGiteaRepo:   getEnv("PARTNER_GITEA_REPO", ""),
		PartnerGiteaRef:    getEnv("PARTNER_GITEA_REF", ""),
		PartnerGiteaEmails: getEnv("PARTNER_GITEA_EMAILS", "alexistrouve.pro@gmail.com"),

		// Session signing
		SessionSecret: getEnv("SESSION_SECRET", ""),
		// Admin panel
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),

		// Défauts = positionnement actuel : TJM pratiqué, TJM visé.
		ComposeRateCurrent: getEnvAsInt("COMPOSE_RATE_CURRENT", 200),
		ComposeRateTarget:  getEnvAsInt("COMPOSE_RATE_TARGET", 300),
		ComposeRateFloor:   getEnvAsInt("COMPOSE_RATE_FLOOR", 260),
		// -1 = plafond des générations LEVÉ (décision d'Alexi du 2026-10-15, provisoire).
		// Valeur de référence pour le rétablir : 40 (une journée active — une dizaine d'annonces,
		// 2-3 tours chacune, quelques CV — sans ouvrir la porte à un usage en masse).
		ProspectDailyAIMax: getEnvAsInt("PROSPECT_DAILY_AI_MAX", -1),
		// -1 = plafond des vérifications Composer LEVÉ (décision d'Alexi du 2026-10-15, provisoire).
		// Pour le rétablir : poser une valeur positive (ex. 120 = 3 × PROSPECT_DAILY_AI_MAX).
		ProspectVerifyDailyMax: getEnvAsInt("PROSPECT_VERIFY_DAILY_MAX", -1),
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
