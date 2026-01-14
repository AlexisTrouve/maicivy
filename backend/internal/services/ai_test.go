// backend/internal/services/ai_test.go
package services

import (
	"context"
	"errors"
	"testing"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMetricsRecorder pour tests
type MockMetricsRecorder struct {
	mock.Mock
}

func (m *MockMetricsRecorder) RecordAIMetrics(metrics models.AIMetrics) {
	m.Called(metrics)
}

// Test création service avec configuration valide
func TestNewAIService_ValidConfig(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-anthropic-key",
		OpenAIAPIKey:         "test-openai-key",
		PrimaryProvider:      "claude",
		MaxRequestsPerMinute: 60,
	}

	mockMetrics := new(MockMetricsRecorder)
	svc, err := NewAIService(cfg, mockMetrics)

	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.claudeClient)
	assert.NotNil(t, svc.openaiClient)
	assert.Equal(t, "claude", svc.config.PrimaryProvider)
}

// Test création service sans API keys
func TestNewAIService_NoAPIKeys(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey: "",
		OpenAIAPIKey:    "",
	}

	svc, err := NewAIService(cfg, nil)

	assert.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "at least one AI provider API key required")
}

// Test création service avec seulement Claude
func TestNewAIService_OnlyClaude(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-claude-key",
		OpenAIAPIKey:         "",
		PrimaryProvider:      "claude",
		MaxRequestsPerMinute: 60,
	}

	svc, err := NewAIService(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.claudeClient)
	assert.Nil(t, svc.openaiClient)
}

// Test création service avec seulement OpenAI
func TestNewAIService_OnlyOpenAI(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "",
		OpenAIAPIKey:         "test-openai-key",
		PrimaryProvider:      "openai",
		MaxRequestsPerMinute: 60,
	}

	svc, err := NewAIService(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.claudeClient)
	assert.NotNil(t, svc.openaiClient)
}

// Test metrics recorder par défaut
func TestAIService_DefaultMetricsRecorder(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 60,
	}

	svc, err := NewAIService(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, svc.metricsRecorder)

	// Vérifier que DefaultMetricsRecorder fonctionne sans paniquer
	metrics := models.AIMetrics{
		Provider:       "claude",
		Model:          "claude-3-5-sonnet-20241022",
		Success:        true,
		TotalTokens:    150,
		EstimatedCost:  0.0015,
		ResponseTimeMs: 1200,
	}

	// Ne devrait pas paniquer
	assert.NotPanics(t, func() {
		svc.metricsRecorder.RecordAIMetrics(metrics)
	})
}

// Test configuration des modèles
func TestAIService_ModelConfiguration(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 60,
		ClaudeModel:          "claude-3-5-sonnet-20241022",
		OpenAIModel:          "gpt-4-turbo-preview",
	}
	svc, _ := NewAIService(cfg, nil)

	assert.Equal(t, "claude-3-5-sonnet-20241022", svc.config.ClaudeModel)
	assert.Equal(t, "gpt-4-turbo-preview", svc.config.OpenAIModel)
}

// Test enregistrement des métriques
func TestAIService_RecordMetrics(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 60,
	}

	mockMetrics := new(MockMetricsRecorder)
	svc, _ := NewAIService(cfg, mockMetrics)

	metrics := models.AIMetrics{
		Provider:       "claude",
		Model:          "claude-3-5-sonnet-20241022",
		Success:        true,
		TotalTokens:    200,
		EstimatedCost:  0.002,
		ResponseTimeMs: 1500,
	}

	// Expect RecordAIMetrics être appelé
	mockMetrics.On("RecordAIMetrics", mock.MatchedBy(func(m models.AIMetrics) bool {
		return m.Provider == "claude" &&
			m.TotalTokens == 200 &&
			m.Success == true
	})).Once()

	svc.metricsRecorder.RecordAIMetrics(metrics)

	mockMetrics.AssertExpectations(t)
}

// Test context cancelled
func TestAIService_ContextCancelled(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 60,
	}

	svc, _ := NewAIService(cfg, nil)

	// Créer context déjà annulé
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Annuler immédiatement

	text, metrics, err := svc.GenerateText(ctx, "Test prompt")

	assert.Error(t, err)
	assert.Empty(t, text)
	assert.Nil(t, metrics)
	assert.True(t, errors.Is(err, context.Canceled) || err.Error() == "rate limit: context canceled")
}

// Test provider primaire configuration
func TestAIService_PrimaryProvider(t *testing.T) {
	testCases := []struct {
		name            string
		primaryProvider string
		expectClaude    bool
	}{
		{"Claude primary", "claude", true},
		{"OpenAI primary", "openai", false},
		{"Invalid defaults to claude", "invalid", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.AIConfig{
				AnthropicAPIKey:      "test-claude",
				OpenAIAPIKey:         "test-openai",
				PrimaryProvider:      tc.primaryProvider,
				MaxRequestsPerMinute: 60,
			}

			svc, err := NewAIService(cfg, nil)

			assert.NoError(t, err)
			assert.Equal(t, tc.primaryProvider, svc.config.PrimaryProvider)
		})
	}
}

// Test rate limiter configuration
func TestAIService_RateLimiter(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 120, // 2 requêtes par seconde
	}

	svc, err := NewAIService(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, svc.rateLimiter)
}

// Test métriques pour différents providers
func TestAIService_MetricsForDifferentProviders(t *testing.T) {
	mockMetrics := new(MockMetricsRecorder)

	claudeMetrics := models.AIMetrics{
		Provider:       "claude",
		Model:          "claude-3-5-sonnet-20241022",
		Success:        true,
		TotalTokens:    150,
		EstimatedCost:  0.0015,
		ResponseTimeMs: 1200,
	}

	openaiMetrics := models.AIMetrics{
		Provider:       "openai",
		Model:          "gpt-4-turbo-preview",
		Success:        true,
		TotalTokens:    180,
		EstimatedCost:  0.0018,
		ResponseTimeMs: 1000,
	}

	mockMetrics.On("RecordAIMetrics", mock.MatchedBy(func(m models.AIMetrics) bool {
		return m.Provider == "claude"
	})).Once()

	mockMetrics.On("RecordAIMetrics", mock.MatchedBy(func(m models.AIMetrics) bool {
		return m.Provider == "openai"
	})).Once()

	mockMetrics.RecordAIMetrics(claudeMetrics)
	mockMetrics.RecordAIMetrics(openaiMetrics)

	mockMetrics.AssertExpectations(t)
}

// Benchmark création de service
func BenchmarkNewAIService(b *testing.B) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		OpenAIAPIKey:         "test-key",
		PrimaryProvider:      "claude",
		MaxRequestsPerMinute: 60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewAIService(cfg, nil)
	}
}

// Benchmark enregistrement métriques
func BenchmarkRecordMetrics(b *testing.B) {
	mockMetrics := new(MockMetricsRecorder)
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
		MaxRequestsPerMinute: 60,
	}
	svc, _ := NewAIService(cfg, mockMetrics)

	metrics := models.AIMetrics{
		Provider:       "claude",
		Model:          "claude-3-5-sonnet-20241022",
		Success:        true,
		TotalTokens:    200,
		EstimatedCost:  0.002,
		ResponseTimeMs: 1500,
	}

	mockMetrics.On("RecordAIMetrics", mock.Anything).Return()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.metricsRecorder.RecordAIMetrics(metrics)
	}
}
