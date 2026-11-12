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
		AnthropicAPIKey:      "test-gateway-key",
		MaxRequestsPerMinute: 60,
	}

	mockMetrics := new(MockMetricsRecorder)
	svc, err := NewAIService(cfg, mockMetrics)

	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.client)
}

// Test création service sans clé de passerelle.
// POURQUOI la clé est devenue OBLIGATOIRE : elle ouvre la seule route existante. Le repli passant
// désormais par la même passerelle, il n'y a plus de configuration où l'absence de cette clé
// laisserait un chemin de génération utilisable.
func TestNewAIService_NoAPIKey(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey: "",
	}

	svc, err := NewAIService(cfg, nil)

	assert.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "gateway API key required")
}

// Un SEUL client est construit, quelle que soit la configuration — y compris quand un modèle de
// repli d'un autre fournisseur est demandé. C'est l'invariant du changement : une destination réseau.
func TestNewAIService_UnSeulClientMemeAvecRepli(t *testing.T) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-gateway-key",
		AnthropicBaseURL:     "https://ai.etheryale.com",
		FallbackModel:        "gpt-5.6-terra",
		MaxRequestsPerMinute: 60,
	}

	svc, err := NewAIService(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, svc.client)
	assert.Equal(t, "gpt-5.6-terra", svc.config.FallbackModel,
		"le repli est un identifiant de modèle, pas un second client")
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
		FallbackModel:        "gpt-5.6-terra",
	}
	svc, _ := NewAIService(cfg, nil)

	assert.Equal(t, "claude-3-5-sonnet-20241022", svc.config.ClaudeModel)
	assert.Equal(t, "gpt-5.6-terra", svc.config.FallbackModel)
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

// Test métriques pour différents modèles.
// Le champ Provider vaut désormais "etheryale" pour TOUS les appels — c'est le transport réel, et
// c'est le Model qui distingue le fournisseur en bout de chaîne.
func TestAIService_MetricsForDifferentModels(t *testing.T) {
	mockMetrics := new(MockMetricsRecorder)

	claudeMetrics := models.AIMetrics{
		Provider:       gatewayProvider,
		Model:          "claude-3-5-sonnet-20241022",
		Success:        true,
		TotalTokens:    150,
		EstimatedCost:  0.0015,
		ResponseTimeMs: 1200,
	}

	fallbackMetrics := models.AIMetrics{
		Provider:       gatewayProvider,
		Model:          "gpt-5.6-terra",
		Success:        true,
		TotalTokens:    180,
		EstimatedCost:  0,
		ResponseTimeMs: 1000,
	}

	mockMetrics.On("RecordAIMetrics", mock.MatchedBy(func(m models.AIMetrics) bool {
		return m.Provider == gatewayProvider && m.Model == "claude-3-5-sonnet-20241022"
	})).Once()

	mockMetrics.On("RecordAIMetrics", mock.MatchedBy(func(m models.AIMetrics) bool {
		return m.Provider == gatewayProvider && m.Model == "gpt-5.6-terra"
	})).Once()

	mockMetrics.RecordAIMetrics(claudeMetrics)
	mockMetrics.RecordAIMetrics(fallbackMetrics)

	mockMetrics.AssertExpectations(t)
}

// Benchmark création de service
func BenchmarkNewAIService(b *testing.B) {
	cfg := &config.AIConfig{
		AnthropicAPIKey:      "test-key",
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
