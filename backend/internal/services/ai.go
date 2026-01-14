package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sashabaranov/go-openai"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

type AIService struct {
	config          *config.AIConfig
	claudeClient    *anthropic.Client
	openaiClient    *openai.Client
	rateLimiter     *rate.Limiter
	metricsRecorder MetricsRecorder
}

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
	// Validate API keys
	if cfg.AnthropicAPIKey == "" && cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("at least one AI provider API key required")
	}

	if metrics == nil {
		metrics = &DefaultMetricsRecorder{}
	}

	svc := &AIService{
		config:          cfg,
		rateLimiter:     rate.NewLimiter(rate.Limit(cfg.MaxRequestsPerMinute)/60, 1),
		metricsRecorder: metrics,
	}

	// Initialize Claude client
	if cfg.AnthropicAPIKey != "" {
		client := anthropic.NewClient(
			option.WithAPIKey(cfg.AnthropicAPIKey),
		)
		svc.claudeClient = &client
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
		log.Warn().Err(err).Msg("Claude generation failed, trying OpenAI fallback")
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
	var resp *anthropic.Message
	var err error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			log.Info().Int("attempt", attempt).Dur("backoff", backoff).Msg("Retrying Claude request")
			time.Sleep(backoff)
		}

		resp, err = s.claudeClient.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(s.config.ClaudeModel),
			MaxTokens: int64(s.config.MaxTokensPerRequest),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
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
	metrics.TokensInput = int(resp.Usage.InputTokens)
	metrics.TokensOutput = int(resp.Usage.OutputTokens)
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
		log.Error().Err(err).Str("model", s.config.OpenAIModel).Msg("OpenAI API call failed")
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
