// backend/internal/services/ai_test.go
package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// go:generate mockgen -source=ai.go -destination=ai_mock.go -package=services

func TestAIService_GenerateLetter_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock du client Claude
	mockClient := NewMockClaudeClient(ctrl)

	// Expect appel API avec prompt spécifique
	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{
			Content: "Chère entreprise XYZ, je suis motivé car...",
			Tokens:  150,
		}, nil)

	// Test service avec mock
	service := NewAIService(mockClient)
	letter, err := service.GenerateMotivationLetter(context.Background(), "XYZ Corp", "backend")

	assert.NoError(t, err)
	assert.NotEmpty(t, letter.Content)
	assert.Contains(t, letter.Content, "motivé")
}

func TestAIService_Fallback_ClaudeToGPT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClaude := NewMockClaudeClient(ctrl)
	mockGPT := NewMockGPTClient(ctrl)

	// Claude échoue
	mockClaude.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("API rate limit"))

	// Fallback sur GPT
	mockGPT.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{Content: "Lettre générée par GPT"}, nil)

	service := NewAIService(mockClaude, mockGPT)
	letter, err := service.GenerateMotivationLetter(context.Background(), "ABC Inc", "frontend")

	assert.NoError(t, err)
	assert.Contains(t, letter.Content, "GPT")
}

func TestAIService_GenerateAntiMotivationLetter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockClaudeClient(ctrl)

	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{
			Content: "Je ne suis absolument pas intéressé par votre entreprise car...",
			Tokens:  200,
		}, nil)

	service := NewAIService(mockClient)
	letter, err := service.GenerateAntiMotivationLetter(context.Background(), "BadCorp", "backend")

	assert.NoError(t, err)
	assert.Contains(t, letter.Content, "pas intéressé")
}

func TestAIService_GenerateBoth_DualLetters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockClaudeClient(ctrl)

	// Expect 2 calls (motivation + anti-motivation)
	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{Content: "Motivation letter"}, nil).
		Times(1)

	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{Content: "Anti-motivation letter"}, nil).
		Times(1)

	service := NewAIService(mockClient)
	result, err := service.GenerateBothLetters(context.Background(), "TechCorp", "fullstack")

	assert.NoError(t, err)
	assert.NotEmpty(t, result.MotivationLetter)
	assert.NotEmpty(t, result.AntiMotivationLetter)
	assert.Equal(t, "TechCorp", result.CompanyName)
}

func TestAIService_Timeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockClaudeClient(ctrl)

	// Simulate timeout
	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(nil, context.DeadlineExceeded)

	service := NewAIService(mockClient)
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	_, err := service.GenerateMotivationLetter(ctx, "SlowCorp", "backend")

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestAIService_InvalidCompanyName(t *testing.T) {
	service := NewAIService(nil)

	testCases := []struct {
		name        string
		companyName string
		expectError bool
	}{
		{"Empty", "", true},
		{"Too short", "AB", true},
		{"Valid", "Google", false},
		{"Valid long", "International Business Machines", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateCompanyName(tc.companyName)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAIService_TokenCounting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockClaudeClient(ctrl)

	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{
			Content: "Test letter",
			Tokens:  500,
		}, nil)

	service := NewAIService(mockClient)
	letter, err := service.GenerateMotivationLetter(context.Background(), "TestCorp", "backend")

	assert.NoError(t, err)
	assert.Equal(t, 500, letter.TokensUsed)
}

// Benchmark génération lettre
func BenchmarkGenerateMotivationLetter(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockClient := NewMockClaudeClient(ctrl)

	mockClient.EXPECT().
		SendMessage(gomock.Any(), gomock.Any()).
		Return(&AIResponse{Content: "Test"}, nil).
		AnyTimes()

	service := NewAIService(mockClient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GenerateMotivationLetter(context.Background(), "BenchCorp", "backend")
	}
}
