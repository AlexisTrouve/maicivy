package config

import (
	"os"
	"strconv"
	"time"
)

type AIConfig struct {
	// Providers
	//
	// Il n'y en a qu'UN : la passerelle Etheryale (AnthropicBaseURL). Tout modèle, quel que soit son
	// fournisseur d'origine, est atteint à travers elle en protocole Anthropic — elle traduit vers
	// OpenAI quand l'identifiant de modèle le désigne. Voir FallbackModel.
	AnthropicAPIKey  string
	AnthropicBaseURL string // Passerelle (ex: https://ai.etheryale.com)
	ChatAPIKey       string // Clé dédiée à l'agent chat (indépendante du reste)
	OwnerAPIKey      string // Clé secrète owner → tier premium (Opus)

	// Models
	ClaudeModel string
	// OwnerModel : modèle utilisé pour les requêtes owner (chat admin, lettres, messages, compose).
	// Voir Load() pour le choix d'une valeur épinglée plutôt que de l'alias `claude-opus-latest`.
	OwnerModel string
	// FallbackModel : modèle de repli si le modèle demandé échoue. Vide = aucun repli (échec franc).
	// Voir LoadAIConfig() pour le pourquoi de la valeur par défaut.
	FallbackModel string
	// CVModel : modèle du CV — classement des projets par pertinence ET CV taillé sur une offre.
	// Voir LoadAIConfig() pour le choix de la valeur.
	CVModel string
	// DossierModel : modèle des cinq agents du dossier vivant (orchestrateur + quatre sous-agents).
	// Voir LoadAIConfig() pour le choix de la valeur.
	DossierModel string
	// DossierSchemaModel : modèle du SEUL Dessinateur du dossier vivant, l'agent que l'Architecte
	// lance pour dessiner le schéma d'architecture. Voir LoadAIConfig() pour le choix de la valeur.
	DossierSchemaModel string

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
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicBaseURL: os.Getenv("ANTHROPIC_BASE_URL"),
		ChatAPIKey:       os.Getenv("CHAT_API_KEY"), // clé dédiée agent chat
		OwnerAPIKey:      os.Getenv("MAICIVY_OWNER_API_KEY"),
		// ClaudeModel : modèle des requêtes VISITEUR (chat public, lettres, messages) et défaut du
		// service IA. L'alias `claude-haiku-latest` du proxy pointe toujours sur le dernier Haiku.
		//
		// POURQUOI l'alias ici, alors qu'OwnerModel est épinglé : ce chemin sert une vitrine et une
		// démo. Personne n'y reconnaît la plume d'Alexis, donc une montée de version est un gain
		// gratuit. La règle du projet : on ÉPINGLE ce qui parle au nom d'Alexis, on laisse FLOTTER ce
		// qui démontre.
		ClaudeModel: getEnvOrDefault("CLAUDE_MODEL", "claude-haiku-latest"),
		// OwnerModel : modèle des requêtes owner (chat admin, lettres, messages, compose).
		//
		// POURQUOI configurable plutôt qu'en dur : il était répété dans trois handlers, ce qui rendait
		// une montée de version invisible et partielle. Une variable d'env permet aussi de basculer
		// sans rebuild sur l'alias `claude-opus-latest` du proxy, qui pointe toujours sur le dernier
		// Opus.
		//
		// POURQUOI la valeur ÉPINGLÉE par défaut, et pas l'alias : cet agent écrit des messages
		// clients calibrés sur la voix d'Alexis. Avec l'alias, une nouvelle version d'Opus changerait
		// le rendu du jour au lendemain sans qu'une ligne de code ait bougé — exactement le genre de
		// régression de voix qu'on vient de corriger, mais sans coupable identifiable. On épingle, et
		// on monte sciemment.
		OwnerModel: getEnvOrDefault("OWNER_MODEL", "claude-opus-5"),
		// FallbackModel : modèle joué quand le modèle demandé échoue.
		//
		// POURQUOI un MODÈLE et non un second fournisseur : le repli tapait auparavant api.openai.com
		// en direct, avec sa propre clé et son propre client. Trois angles morts d'un coup — un
		// message signé Alexis pouvait sortir d'ailleurs sans trace, le coût échappait à la
		// comptabilité de la passerelle, et une seconde clé vivait dans le conteneur. La passerelle
		// sait traduire une requête `/v1/messages` vers un fournisseur OpenAI (translator/
		// messages_to_openai.go, streaming compris) : il suffit donc de CHANGER L'IDENTIFIANT DE
		// MODÈLE. Un transport, une clé, une comptabilité.
		//
		// POURQUOI `gpt-5.6-terra` par défaut : c'est un modèle d'un AUTRE fournisseur, seul moyen que
		// le repli serve à quelque chose — un second modèle Anthropic tomberait avec le premier si la
		// panne est côté Anthropic. Il est exposé par la passerelle (cf. son `/v1/models`).
		//
		// ⚠️ Réglable sans rebuild via AI_FALLBACK_MODEL, et c'est délibéré : l'upstream OpenAI de la
		// passerelle dépend d'un compte dont la session peut expirer. Mettre la variable à vide
		// DÉSACTIVE le repli (échec franc) — c'est le bon geste tant que cet upstream est en panne,
		// plutôt que de payer un aller-retour pour un échec certain.
		FallbackModel: getEnvOrDefault("AI_FALLBACK_MODEL", "gpt-5.6-terra"),
		// CVModel : décide quels projets sortent sur le CV et réécrit les accroches d'un CV taillé.
		// ÉPINGLÉ (règle du projet : ce qui parle au nom d'Alexis ne bouge pas sous lui).
		// Haiku 4.5 jusqu'au 2026-10-11 : il classait par mots-clés (cf. llm_scoring.go). Sonnet
		// 5.5 : jugement nettement meilleur, coût négligeable — un appel par thème et par 6 h pour
		// le classement, un par offre pour le CV taillé. Réglable sans rebuild (CV_MODEL).
		CVModel: getEnvOrDefault("CV_MODEL", "claude-sonnet-5-5"),
		// DossierModel : ÉPINGLÉ et non l'alias `claude-haiku-latest`, parce que l'écran AFFICHE
		// l'identifiant (run.started) et que le coût du run est calculé sur son tarif : un alias qui
		// glisse ferait mentir l'un et l'autre. Haiku : vingt appels courts par run, en parallèle,
		// pour quelques centimes. Réglable sans rebuild (DOSSIER_VIVANT_MODEL).
		DossierModel: getEnvOrDefault("DOSSIER_VIVANT_MODEL", "claude-haiku-5-5"),
		// DossierSchemaModel : le schéma est LA figure du dossier, celle qu'un décideur lit en dix
		// secondes ; dessiné par Haiku il restait générique (« Réception → Extraction → Validation »,
		// jugé « mid » sur le run du 2026-10). Un seul appel par run, sur Sonnet à effort moyen : le
		// jugement là où il se voit, Haiku partout ailleurs. ÉPINGLÉ pour la même raison que
		// DossierModel (identifiant affiché, coût calculé sur son tarif). Réglable sans rebuild.
		DossierSchemaModel:   getEnvOrDefault("DOSSIER_VIVANT_SCHEMA_MODEL", "claude-sonnet-5-5"),
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
