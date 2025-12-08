// backend/internal/api/letters_test.go
package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock AI Service
type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) GenerateLetters(companyName, theme string) (*LettersResponse, error) {
	args := m.Called(companyName, theme)
	return args.Get(0).(*LettersResponse), args.Error(1)
}

// Mock Visitor Service
type MockVisitorService struct {
	mock.Mock
}

func (m *MockVisitorService) GetVisitorCount(sessionID string) (int, error) {
	args := m.Called(sessionID)
	return args.Int(0), args.Error(1)
}

func (m *MockVisitorService) HasAIAccess(sessionID string) (bool, error) {
	args := m.Called(sessionID)
	return args.Bool(0), args.Error(1)
}

// Test POST /api/letters/generate - Success
func TestGenerateLetters_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	mockAI := new(MockAIService)
	mockVisitor := new(MockVisitorService)
	handler := NewLettersHandler(mockAI, mockVisitor)

	app.Post("/api/letters/generate", handler.GenerateLetters)

	// Mock visitor avec accès IA (>= 3 visites)
	mockVisitor.On("GetVisitorCount", mock.Anything).Return(5, nil)
	mockVisitor.On("HasAIAccess", mock.Anything).Return(true, nil)

	// Mock AI response
	aiResp := &LettersResponse{
		MotivationLetter:     "Je suis très motivé...",
		AntiMotivationLetter: "Je ne suis pas du tout intéressé...",
		CompanyName:          "Google",
		CompanyInfo:          map[string]string{"industry": "tech"},
	}
	mockAI.On("GenerateLetters", "Google", "backend").Return(aiResp, nil)

	// Request body
	reqBody := map[string]string{
		"company_name": "Google",
		"theme":        "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Request
	req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session_id=test-session")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var lettersResp LettersResponse
	json.NewDecoder(resp.Body).Decode(&lettersResp)

	assert.Contains(t, lettersResp.MotivationLetter, "motivé")
	assert.Contains(t, lettersResp.AntiMotivationLetter, "pas du tout")
	assert.Equal(t, "Google", lettersResp.CompanyName)

	mockVisitor.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

// Test POST /api/letters/generate - Access Denied (< 3 visites)
func TestGenerateLetters_AccessDenied(t *testing.T) {
	// Setup
	app := fiber.New()
	mockAI := new(MockAIService)
	mockVisitor := new(MockVisitorService)
	handler := NewLettersHandler(mockAI, mockVisitor)

	app.Post("/api/letters/generate", handler.GenerateLetters)

	// Mock visitor SANS accès (< 3 visites)
	mockVisitor.On("GetVisitorCount", mock.Anything).Return(2, nil)
	mockVisitor.On("HasAIAccess", mock.Anything).Return(false, nil)

	// Request body
	reqBody := map[string]string{
		"company_name": "Google",
		"theme":        "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Request
	req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session_id=test-session")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "3 visits required")
	assert.Equal(t, 2, int(errorResp["visit_count"].(float64)))

	mockVisitor.AssertExpectations(t)
	// AI NE DEVRAIT PAS être appelé
	mockAI.AssertNotCalled(t, "GenerateLetters")
}

// Test POST /api/letters/generate - Rate Limiting
func TestGenerateLetters_RateLimited(t *testing.T) {
	// Setup
	app := fiber.New()
	mockAI := new(MockAIService)
	mockVisitor := new(MockVisitorService)
	mockRateLimit := new(MockRateLimitService)
	handler := NewLettersHandler(mockAI, mockVisitor).WithRateLimit(mockRateLimit)

	app.Post("/api/letters/generate", handler.GenerateLetters)

	// Mock accès autorisé
	mockVisitor.On("GetVisitorCount", mock.Anything).Return(5, nil)
	mockVisitor.On("HasAIAccess", mock.Anything).Return(true, nil)

	// Mock rate limit dépassé
	mockRateLimit.On("CheckLimit", "ai:test-session").Return(false, nil)

	// Request
	reqBody := map[string]string{"company_name": "Google", "theme": "backend"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session_id=test-session")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "rate limit")

	mockRateLimit.AssertExpectations(t)
	// AI NE DEVRAIT PAS être appelé
	mockAI.AssertNotCalled(t, "GenerateLetters")
}

// Test POST /api/letters/generate - Validation Errors
func TestGenerateLetters_ValidationErrors(t *testing.T) {
	app := fiber.New()
	handler := NewLettersHandler(nil, nil)

	app.Post("/api/letters/generate", handler.GenerateLetters)

	testCases := []struct {
		name     string
		body     map[string]string
		expected string
	}{
		{
			name:     "Missing company_name",
			body:     map[string]string{"theme": "backend"},
			expected: "company_name is required",
		},
		{
			name:     "Empty company_name",
			body:     map[string]string{"company_name": "", "theme": "backend"},
			expected: "company_name cannot be empty",
		},
		{
			name:     "Company name too short",
			body:     map[string]string{"company_name": "AB", "theme": "backend"},
			expected: "company_name must be at least 3 characters",
		},
		{
			name:     "Invalid theme",
			body:     map[string]string{"company_name": "Google", "theme": "invalid123"},
			expected: "invalid theme",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)

			var errorResp map[string]string
			json.NewDecoder(resp.Body).Decode(&errorResp)

			assert.Contains(t, errorResp["error"], tc.expected)
		})
	}
}

// Test POST /api/letters/generate - AI Service Error
func TestGenerateLetters_AIServiceError(t *testing.T) {
	// Setup
	app := fiber.New()
	mockAI := new(MockAIService)
	mockVisitor := new(MockVisitorService)
	handler := NewLettersHandler(mockAI, mockVisitor)

	app.Post("/api/letters/generate", handler.GenerateLetters)

	// Mock accès autorisé
	mockVisitor.On("GetVisitorCount", mock.Anything).Return(5, nil)
	mockVisitor.On("HasAIAccess", mock.Anything).Return(true, nil)

	// Mock AI error
	mockAI.On("GenerateLetters", "Google", "backend").Return((*LettersResponse)(nil), assert.AnError)

	// Request
	reqBody := map[string]string{"company_name": "Google", "theme": "backend"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "session_id=test-session")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "AI service")

	mockAI.AssertExpectations(t)
}

// Test GET /api/letters/history
func TestGetLettersHistory(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockLettersRepository)
	handler := NewLettersHandler(nil, nil).WithRepository(mockRepo)

	app.Get("/api/letters/history", handler.GetHistory)

	// Mock data
	history := []*GeneratedLetter{
		{ID: 1, CompanyName: "Google", CreatedAt: "2025-12-08"},
		{ID: 2, CompanyName: "Microsoft", CreatedAt: "2025-12-07"},
	}

	mockRepo.On("GetLettersBySession", "test-session").Return(history, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/letters/history", nil)
	req.Header.Set("Cookie", "session_id=test-session")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var historyResp []GeneratedLetter
	json.NewDecoder(resp.Body).Decode(&historyResp)

	assert.Len(t, historyResp, 2)
	assert.Equal(t, "Google", historyResp[0].CompanyName)

	mockRepo.AssertExpectations(t)
}
