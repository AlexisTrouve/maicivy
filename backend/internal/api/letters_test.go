// backend/internal/api/letters_test.go
package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"maicivy/internal/api/dto"
	"maicivy/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock de LetterQueueService
type MockLetterQueueService struct {
	mock.Mock
}

func (m *MockLetterQueueService) EnqueueJob(visitorID, companyName, jobTitle, theme string) (string, error) {
	args := m.Called(visitorID, companyName, jobTitle, theme)
	return args.String(0), args.Error(1)
}

func (m *MockLetterQueueService) GetJobStatus(jobID string) (*services.LetterJob, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LetterJob), args.Error(1)
}

func (m *MockLetterQueueService) UpdateJobStatus(jobID string, status services.JobStatus, progress int) error {
	args := m.Called(jobID, status, progress)
	return args.Error(0)
}

func (m *MockLetterQueueService) CompleteJob(jobID string, motivationID, antiMotivationID uuid.UUID) error {
	args := m.Called(jobID, motivationID, antiMotivationID)
	return args.Error(0)
}

func (m *MockLetterQueueService) FailJob(jobID string, errorMsg string) error {
	args := m.Called(jobID, errorMsg)
	return args.Error(0)
}

func (m *MockLetterQueueService) RetryJob(jobID string) error {
	args := m.Called(jobID)
	return args.Error(0)
}

func (m *MockLetterQueueService) PopJob() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockLetterQueueService) GetQueueLength() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLetterQueueService) CleanupOldJobs() error {
	args := m.Called()
	return args.Error(0)
}

// Test POST /api/v1/letters/generate - Success
func TestGenerateLetters_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	mockQueue := new(MockLetterQueueService)

	// Handler simplifié pour test (sans Redis/DB)
	handler := &LettersHandler{
		queueService: mockQueue,
	}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		// Simuler middleware AccessGate
		c.Locals("session_id", "test-session-123")
		c.Locals("has_access", true)
		c.Locals("rate_limit_remaining", 4)
		return handler.GenerateLetter(c)
	})

	// Mock queue success
	mockQueue.On("EnqueueJob", "test-session-123", "Google", "", "backend").
		Return("job-abc-123", nil)

	// Request body
	reqBody := dto.GenerateLetterRequest{
		CompanyName: "Google",
		Theme:       "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Request
	req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode) // 202 Accepted

	var result dto.LetterGenerationResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "job-abc-123", result.JobID)
	assert.Equal(t, "queued", result.Status)
	assert.Contains(t, result.Message, "Génération en cours")

	mockQueue.AssertExpectations(t)
}

// Test POST /api/v1/letters/generate - Validation Errors
func TestGenerateLetters_ValidationErrors(t *testing.T) {
	app := fiber.New()
	handler := &LettersHandler{}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		c.Locals("has_access", true)
		return handler.GenerateLetter(c)
	})

	testCases := []struct {
		name     string
		body     dto.GenerateLetterRequest
		expected string
	}{
		{
			name:     "Empty company_name",
			body:     dto.GenerateLetterRequest{CompanyName: "", Theme: "backend"},
			expected: "CompanyName",
		},
		{
			name:     "Company name too short",
			body:     dto.GenerateLetterRequest{CompanyName: "A", Theme: "backend"},
			expected: "min",
		},
		{
			name:     "Invalid theme",
			body:     dto.GenerateLetterRequest{CompanyName: "Google", Theme: "invalid123"},
			expected: "oneof",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)

			var errorResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errorResp)

			// Vérifier que l'erreur contient le champ attendu
			details, ok := errorResp["details"].(string)
			assert.True(t, ok)
			assert.Contains(t, details, tc.expected)
		})
	}
}

// Test POST /api/v1/letters/generate - Missing Session
func TestGenerateLetters_MissingSession(t *testing.T) {
	// Setup
	app := fiber.New()
	handler := &LettersHandler{}

	app.Post("/api/v1/letters/generate", handler.GenerateLetter)

	// Request body valide mais sans session
	reqBody := dto.GenerateLetterRequest{
		CompanyName: "Google",
		Theme:       "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "Session")
	assert.Equal(t, "SESSION_REQUIRED", errorResp["code"])
}

// Test POST /api/v1/letters/generate - Queue Error
func TestGenerateLetters_QueueError(t *testing.T) {
	// Setup
	app := fiber.New()
	mockQueue := new(MockLetterQueueService)

	handler := &LettersHandler{
		queueService: mockQueue,
	}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		c.Locals("has_access", true)
		return handler.GenerateLetter(c)
	})

	// Mock queue error
	mockQueue.On("EnqueueJob", "test-session", "Google", "", "backend").
		Return("", assert.AnError)

	// Request
	reqBody := dto.GenerateLetterRequest{CompanyName: "Google", Theme: "backend"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "Failed to enqueue")
	assert.Equal(t, "QUEUE_ERROR", errorResp["code"])

	mockQueue.AssertExpectations(t)
}

// Test avec job_title optionnel
func TestGenerateLetters_WithJobTitle(t *testing.T) {
	// Setup
	app := fiber.New()
	mockQueue := new(MockLetterQueueService)

	handler := &LettersHandler{
		queueService: mockQueue,
	}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		c.Locals("has_access", true)
		c.Locals("rate_limit_remaining", 3)
		return handler.GenerateLetter(c)
	})

	// Mock avec job title
	mockQueue.On("EnqueueJob", "test-session", "Microsoft", "Senior Backend Engineer", "backend").
		Return("job-xyz-456", nil)

	// Request avec job_title
	reqBody := dto.GenerateLetterRequest{
		CompanyName: "Microsoft",
		JobTitle:    "Senior Backend Engineer",
		Theme:       "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode)

	var result dto.LetterGenerationResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "job-xyz-456", result.JobID)

	mockQueue.AssertExpectations(t)
}

// Test validation job_title trop court
func TestGenerateLetters_JobTitleTooShort(t *testing.T) {
	app := fiber.New()
	handler := &LettersHandler{}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		return handler.GenerateLetter(c)
	})

	reqBody := dto.GenerateLetterRequest{
		CompanyName: "Google",
		JobTitle:    "A", // Trop court (< 2 caractères)
		Theme:       "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["details"], "min")
}

// Test tous les thèmes valides
func TestGenerateLetters_ValidThemes(t *testing.T) {
	validThemes := []string{"backend", "frontend", "fullstack", "devops", "data", "ai"}

	app := fiber.New()
	mockQueue := new(MockLetterQueueService)
	handler := &LettersHandler{queueService: mockQueue}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "test-session")
		c.Locals("has_access", true)
		c.Locals("rate_limit_remaining", 5)
		return handler.GenerateLetter(c)
	})

	for _, theme := range validThemes {
		t.Run(theme, func(t *testing.T) {
			mockQueue.On("EnqueueJob", "test-session", "TestCorp", "", theme).
				Return("job-123", nil).Once()

			reqBody := dto.GenerateLetterRequest{
				CompanyName: "TestCorp",
				Theme:       theme,
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 202, resp.StatusCode)
		})
	}

	mockQueue.AssertExpectations(t)
}

// Benchmark génération de lettre
func BenchmarkGenerateLetters(b *testing.B) {
	app := fiber.New()
	mockQueue := new(MockLetterQueueService)
	handler := &LettersHandler{queueService: mockQueue}

	app.Post("/api/v1/letters/generate", func(c *fiber.Ctx) error {
		c.Locals("session_id", "bench-session")
		c.Locals("has_access", true)
		c.Locals("rate_limit_remaining", 100)
		return handler.GenerateLetter(c)
	})

	mockQueue.On("EnqueueJob", "bench-session", "BenchCorp", "", "backend").
		Return("job-bench", nil)

	reqBody := dto.GenerateLetterRequest{
		CompanyName: "BenchCorp",
		Theme:       "backend",
	}
	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/letters/generate", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}
}
