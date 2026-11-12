package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// AIService : génération de texte pour les lettres, messages, compose, blog et CV tailorés.
//
// UN SEUL CLIENT, à dessein : la passerelle Etheryale parle le protocole Anthropic et traduit vers
// les autres fournisseurs selon l'identifiant de modèle. Changer de fournisseur = changer de modèle,
// pas de client. Voir config.AIConfig.FallbackModel pour le pourquoi complet.
type AIService struct {
	config          *config.AIConfig
	client          *anthropic.Client
	rateLimiter     *rate.Limiter
	metricsRecorder MetricsRecorder
}

// gatewayProvider : valeur du champ Provider des métriques. Tout passe par la passerelle, donc la
// distinction historique "claude" / "openai" ne décrivait plus le transport — elle décrivait le
// modèle, information déjà portée par le champ Model.
const gatewayProvider = "etheryale"

type MetricsRecorder interface {
	RecordAIMetrics(metrics models.AIMetrics)
}

// DefaultMetricsRecorder : implémentation par défaut qui log seulement
type DefaultMetricsRecorder struct{}

func (d *DefaultMetricsRecorder) RecordAIMetrics(metrics models.AIMetrics) {
	log.Info().
		Str("provider", metrics.Provider).
		Str("model", metrics.Model).
		Int("total_tokens", metrics.TotalTokens).
		Float64("cost", metrics.EstimatedCost).
		Int64("duration_ms", metrics.ResponseTimeMs).
		Bool("success", metrics.Success).
		Msg("AI metrics recorded")
}

func NewAIService(cfg *config.AIConfig, metrics MetricsRecorder) (*AIService, error) {
	// Une seule clé compte désormais : celle de la passerelle. Sans elle il n'y a aucun chemin vers
	// un modèle, quel que soit le fournisseur visé — le repli passe par la même porte.
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("gateway API key required (ANTHROPIC_API_KEY)")
	}

	if metrics == nil {
		metrics = &DefaultMetricsRecorder{}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.AnthropicAPIKey),
	}
	if cfg.AnthropicBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.AnthropicBaseURL))
	}
	client := anthropic.NewClient(opts...)

	return &AIService{
		config:          cfg,
		client:          &client,
		rateLimiter:     rate.NewLimiter(rate.Limit(cfg.MaxRequestsPerMinute)/60, 1),
		metricsRecorder: metrics,
	}, nil
}

// GenerateText : génère du texte avec fallback automatique (modèle par défaut)
func (s *AIService) GenerateText(ctx context.Context, prompt string) (string, *models.AIMetrics, error) {
	return s.GenerateTextWithModel(ctx, prompt, "")
}

// GenerateTextWithModel : génère du texte avec override de modèle optionnel
// model="" → utilise s.config.ClaudeModel (défaut Haiku)
// model="claude-opus-4-6" → force Opus pour l'owner par exemple
//
// En cas d'échec, rejoue le prompt sur config.FallbackModel — MÊME client, MÊME passerelle, MÊME
// clé : seul l'identifiant de modèle change. C'est ce qui permet de basculer sur un fournisseur
// différent sans ouvrir une seconde route réseau (cf. config.AIConfig.FallbackModel).
//
// COMMENT : 1. résoudre le modèle effectif (l'override prime sur la config), 2. tenter, 3. si échec
// et qu'un repli DISTINCT est configuré, rejouer une fois. Une seule tentative de repli : au-delà,
// ce n'est plus un incident transient mais une panne, et il vaut mieux la rendre visible que de la
// diluer en latence.
func (s *AIService) GenerateTextWithModel(ctx context.Context, prompt, model string) (string, *models.AIMetrics, error) {
	// Rate limiting interne (par minute)
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return "", nil, fmt.Errorf("rate limit: %w", err)
	}

	primary := model
	if primary == "" {
		primary = s.config.ClaudeModel
	}

	text, metrics, err := s.generateWithModel(ctx, prompt, primary)
	if err == nil {
		return text, metrics, nil
	}

	// Repli : seulement s'il est configuré ET différent du modèle qui vient d'échouer. Rejouer le
	// même identifiant ne ferait que reproduire l'échec en payant un aller-retour de plus.
	fallback := s.config.FallbackModel
	if fallback == "" || fallback == primary {
		return "", nil, fmt.Errorf("génération impossible sur %s (aucun repli applicable): %w", primary, err)
	}

	log.Warn().
		Err(err).
		Str("primary_model", primary).
		Str("fallback_model", fallback).
		Msg("modèle primaire en échec — repli sur un autre modèle de la passerelle")

	text, metrics, err = s.generateWithModel(ctx, prompt, fallback)
	if err != nil {
		return "", nil, fmt.Errorf("génération impossible sur %s puis sur le repli %s: %w", primary, fallback, err)
	}
	return text, metrics, nil
}

// CompletionRequest : un appel unique et entièrement paramétré (cf. Complete).
type CompletionRequest struct {
	Model     string // obligatoire : Complete ne résout aucun défaut
	System    string // vide = pas de prompt système
	Prompt    string
	MaxTokens int
	// Effort — niveau de réflexion demandé (`low` | `medium` | `high`…) ; vide = rien n'est envoyé
	// (la passerelle pose alors la forme par défaut du modèle).
	//
	// POURQUOI la forme `thinking:{type:"adaptive"}` + `output_config:{effort}` et rien d'autre :
	// c'est la convention NATIVE des modèles 5.5 (capturée sur le vrai client par la passerelle,
	// LeCodeurFou translator/default_thinking.go). `budget_tokens` y rend un bloc de réflexion VIDE
	// (panne muette, cf. CLAUDE.md) et `temperature` un 400 : ni l'un ni l'autre ne partent jamais.
	// La réflexion se paie sur MaxTokens : l'appelant dimensionne son plafond en conséquence.
	Effort string
	// Timeout — durée maximale de CET appel ; 0 = config.RequestTimeout (30 s). Un modèle qui
	// réfléchit (effort moyen) peut dépasser le délai réglé pour les appels courts sans être en panne.
	Timeout time.Duration
}

// Completion : le texte (blocs `text` seulement), la raison d'arrêt et les métriques RÉELLES de
// l'appel (tokens rendus par l'API, cache compris ; coût calculé par calculateModelCost).
type Completion struct {
	Text       string
	StopReason string
	Metrics    models.AIMetrics
}

// Complete : UN appel sur un modèle imposé, avec prompt système et plafond de sortie choisis par
// l'appelant. Même client, même passerelle, mêmes retries et même enregistrement des métriques que
// GenerateTextWithModel — deux différences, toutes deux voulues :
//
//   - PAS DE REPLI. L'appelant annonce le modèle à l'écran (dossier vivant : `run.started` porte
//     l'identifiant affiché) ; une bascule silencieuse sur un autre fournisseur ferait mentir
//     l'écran. Un échec remonte tel quel.
//   - PAS DU LIMITEUR GLOBAL `rateLimiter`. Il est réglé pour les chemins visiteur (10 req/min,
//     rafale 1) : des sous-agents qui travaillent en parallèle y seraient sérialisés, un appel
//     toutes les 6 s. Un appelant de Complete DOIT donc porter sa propre borne — le dossier vivant
//     (public, sans quota par décision d'Alexis) borne l'entrée de CHAQUE appel (callsize.go).
//
// La raison d'arrêt est rendue pour que l'appelant traite `max_tokens` (sortie tronquée) comme une
// erreur nommée plutôt que de parser un JSON coupé.
//
// La Completion est rendue MÊME en cas d'erreur : ses métriques disent ce que l'appel a consommé
// (zéro si la passerelle n'a pas répondu), ce qu'un budget doit compter dans tous les cas.
func (s *AIService) Complete(ctx context.Context, req CompletionRequest) (*Completion, error) {
	text, stop, metrics, err := s.call(ctx, req)
	return &Completion{Text: text, StopReason: stop, Metrics: *metrics}, err
}

// generateWithModel : un appel à la passerelle sur un modèle DONNÉ, avec retry sur erreurs
// transientes. Le modèle est toujours explicite ici — la résolution du défaut appartient à
// GenerateTextWithModel, qui doit connaître le modèle effectif pour décider du repli.
//
// L'identifiant peut désigner n'importe quel fournisseur exposé par la passerelle (`claude-*`,
// `gpt-*`) : la requête part en protocole Anthropic dans tous les cas, la traduction est son travail.
func (s *AIService) generateWithModel(ctx context.Context, prompt, effectiveModel string) (string, *models.AIMetrics, error) {
	text, _, metrics, err := s.call(ctx, CompletionRequest{Model: effectiveModel, Prompt: prompt, MaxTokens: s.config.MaxTokensPerRequest})
	return text, metrics, err
}

// call : le seul endroit qui parle à la passerelle — retries, lecture des blocs `text`, métriques.
// Partagé par generateWithModel (lettres, messages…) et Complete (dossier vivant).
func (s *AIService) call(ctx context.Context, req CompletionRequest) (string, string, *models.AIMetrics, error) {
	effectiveModel := req.Model
	start := time.Now()
	metrics := &models.AIMetrics{
		Provider: gatewayProvider,
		Model:    effectiveModel,
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(effectiveModel),
		MaxTokens: int64(req.MaxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	if req.Effort != "" {
		params.Thinking = anthropic.ThinkingConfigParamUnion{OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{}}
		params.OutputConfig = anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffort(req.Effort)}
	}

	// Context avec timeout
	timeout := s.config.RequestTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Appel API avec retry
	var resp *anthropic.Message
	var err error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			log.Info().Int("attempt", attempt).Dur("backoff", backoff).Str("model", effectiveModel).Msg("Retrying gateway request")
			time.Sleep(backoff)
		}

		resp, err = s.client.Messages.New(ctx, params)

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
		return "", "", metrics, fmt.Errorf("gateway API error (model %s): %w", effectiveModel, err)
	}

	// Parse response
	if len(resp.Content) == 0 {
		return "", "", metrics, fmt.Errorf("empty response from model %s", effectiveModel)
	}

	// QUOI : concatène les seuls blocs `text` de la réponse.
	// POURQUOI : les modèles à réflexion toujours active (claude-haiku-5-5, Opus 5) ouvrent la réponse
	// par un bloc `thinking` : lire Content[0].Text rendait "" SANS erreur (lettre ou traduction vide).
	// COMMENT : on ignore thinking/redacted_thinking ; aucun texte du tout = erreur franche, pas de "".
	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	text := sb.String()
	// Metrics — relevées AVANT le contrôle du texte vide : un appel sans texte a quand même consommé
	// des tokens, et un appelant qui tient un budget doit pouvoir les compter.
	//
	// L'entrée compte les TROIS postes facturés : la passerelle pose un cache de prompt (1 h) et son
	// `input_tokens` exclut alors ce qu'elle écrit ou lit en cache (cf. ai_pricing.go). Tout appelant
	// qui tient un budget sur TokensInput (dossier vivant) compte donc l'entrée réellement facturée.
	usage := usageOf(resp.Usage)
	metrics.TokensInput = usage.Prompt()
	metrics.TokensCacheWrite = usage.CacheWrite()
	metrics.TokensCacheRead = usage.CacheRead
	metrics.TokensOutput = usage.Output
	metrics.TotalTokens = metrics.TokensInput + metrics.TokensOutput
	metrics.EstimatedCost = calculateModelCost(usage, effectiveModel)

	if text == "" {
		return "", string(resp.StopReason), metrics, fmt.Errorf("no text block in response from model %s", effectiveModel)
	}

	metrics.Success = true
	s.recordMetrics(metrics)

	return text, string(resp.StopReason), metrics, nil
}

// recordMetrics : enregistre métriques
func (s *AIService) recordMetrics(m *models.AIMetrics) {
	if s.config.EnableCostTracking && s.metricsRecorder != nil {
		s.metricsRecorder.RecordAIMetrics(*m)
	}

	log.Info().
		Str("provider", m.Provider).
		Int("tokens", m.TotalTokens).
		Float64("cost", m.EstimatedCost).
		Int64("duration_ms", m.ResponseTimeMs).
		Bool("success", m.Success).
		Msg("AI generation completed")
}

// isRetryableError : détermine si erreur est retryable
func isRetryableError(err error) bool {
	// Rate limits, timeouts, 5xx errors → retry
	// 4xx (sauf 429) → ne pas retry
	errStr := strings.ToLower(err.Error())

	retryablePatterns := []string{
		"rate_limit",
		"rate limit",
		"timeout",
		"503",
		"502",
		"500",
		"429",
		"too many requests",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
