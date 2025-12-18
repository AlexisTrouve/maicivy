// backend/internal/api/cv_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/models"
	"maicivy/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock du CVService
type MockCVService struct {
	mock.Mock
}

func (m *MockCVService) GetAdaptiveCV(ctx context.Context, themeID string) (*services.AdaptiveCVResponse, error) {
	args := m.Called(ctx, themeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AdaptiveCVResponse), args.Error(1)
}

func (m *MockCVService) GetAvailableThemes() []config.CVTheme {
	args := m.Called()
	return args.Get(0).([]config.CVTheme)
}

func (m *MockCVService) InvalidateCache(ctx context.Context, themeID string) error {
	args := m.Called(ctx, themeID)
	return args.Error(0)
}

func (m *MockCVService) GetAllExperiences(ctx context.Context) ([]models.Experience, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Experience), args.Error(1)
}

func (m *MockCVService) GetAllSkills(ctx context.Context) ([]models.Skill, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Skill), args.Error(1)
}

func (m *MockCVService) GetAllProjects(ctx context.Context) ([]models.Project, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Project), args.Error(1)
}

// Test GET /api/v1/cv (sans thème - retourne fullstack par défaut)
func TestGetCV_DefaultTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/cv", handler.GetAdaptiveCV)

	// Mock response
	now := time.Now()
	cvResp := &services.AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "fullstack",
			Name:        "Full-Stack Developer",
			Description: "Full-stack development",
		},
		Experiences: []models.Experience{
			{
				Title:        "Full-Stack Dev",
				Company:      "TechCorp",
				Tags:         pq.StringArray{"go", "react"},
				Technologies: pq.StringArray{"go", "react"},
				StartDate:    now.AddDate(-2, 0, 0),
			},
		},
		Skills: []models.Skill{
			{Name: "Go", Level: models.SkillLevelExpert},
			{Name: "React", Level: models.SkillLevelAdvanced},
		},
		Projects: []models.Project{
			{Title: "maicivy", Technologies: pq.StringArray{"go", "react"}},
		},
		GeneratedAt: now,
	}

	mockService.On("GetAdaptiveCV", mock.Anything, "fullstack").Return(cvResp, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/v1/cv", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result services.AdaptiveCVResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "fullstack", result.Theme.ID)
	assert.Len(t, result.Experiences, 1)
	assert.Len(t, result.Skills, 2)
	assert.Len(t, result.Projects, 1)

	mockService.AssertExpectations(t)
}

// Test GET /api/v1/cv?theme=backend
func TestGetCV_BackendTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/cv", handler.GetAdaptiveCV)

	// Mock response
	now := time.Now()
	cvResp := &services.AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Backend development",
			TagWeights: map[string]float64{
				"go":         1.0,
				"postgresql": 0.9,
			},
		},
		Experiences: []models.Experience{
			{
				Title:        "Backend Dev",
				Company:      "BackendCorp",
				Tags:         pq.StringArray{"go", "postgresql"},
				Technologies: pq.StringArray{"go", "postgresql"},
				Category:     "backend",
				StartDate:    now.AddDate(-3, 0, 0),
			},
		},
		Skills: []models.Skill{
			{Name: "Go", Level: models.SkillLevelExpert, Category: "backend"},
			{Name: "PostgreSQL", Level: models.SkillLevelAdvanced, Category: "database"},
		},
		Projects:    []models.Project{},
		GeneratedAt: now,
	}

	mockService.On("GetAdaptiveCV", mock.Anything, "backend").Return(cvResp, nil)

	// Request avec query param
	req := httptest.NewRequest("GET", "/api/v1/cv?theme=backend", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result services.AdaptiveCVResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "backend", result.Theme.ID)
	assert.Equal(t, "Backend Dev", result.Experiences[0].Title)
	assert.Equal(t, "backend", result.Experiences[0].Category)

	mockService.AssertExpectations(t)
}

// Test GET /api/v1/cv/themes
func TestGetThemes(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/cv/themes", handler.GetThemes)

	// Mock data
	themes := []config.CVTheme{
		{
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Backend development",
			TagWeights: map[string]float64{
				"go":         1.0,
				"postgresql": 0.9,
			},
		},
		{
			ID:          "frontend",
			Name:        "Frontend Developer",
			Description: "Frontend development",
			TagWeights: map[string]float64{
				"react":      1.0,
				"typescript": 0.9,
			},
		},
	}

	mockService.On("GetAvailableThemes").Return(themes)

	// Request
	req := httptest.NewRequest("GET", "/api/v1/cv/themes", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(2), result["count"])
	assert.NotNil(t, result["themes"])

	mockService.AssertExpectations(t)
}

// Test GET /api/v1/experiences
func TestGetExperiences(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/experiences", handler.GetExperiences)

	// Mock data
	experiences := []models.Experience{
		{
			Title:        "Backend Dev",
			Company:      "Company A",
			Tags:         pq.StringArray{"go", "postgresql"},
			Technologies: pq.StringArray{"go"},
			Category:     "backend",
			StartDate:    time.Now().AddDate(-2, 0, 0),
		},
		{
			Title:        "Frontend Dev",
			Company:      "Company B",
			Tags:         pq.StringArray{"react"},
			Technologies: pq.StringArray{"react"},
			Category:     "frontend",
			StartDate:    time.Now().AddDate(-1, 0, 0),
		},
	}

	mockService.On("GetAllExperiences", mock.Anything).Return(experiences, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/v1/experiences", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(2), result["count"])

	mockService.AssertExpectations(t)
}

// Test GET /api/v1/skills
func TestGetSkills(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/skills", handler.GetSkills)

	// Mock data
	skills := []models.Skill{
		{Name: "Go", Level: models.SkillLevelExpert, Category: "backend"},
		{Name: "React", Level: models.SkillLevelAdvanced, Category: "frontend"},
		{Name: "Docker", Level: models.SkillLevelIntermediate, Category: "devops"},
	}

	mockService.On("GetAllSkills", mock.Anything).Return(skills, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/v1/skills", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(3), result["count"])

	mockService.AssertExpectations(t)
}

// Test erreur thème invalide
func TestGetCV_InvalidTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/cv", handler.GetAdaptiveCV)

	// Mock erreur
	mockService.On("GetAdaptiveCV", mock.Anything, "invalidtheme123").
		Return((*services.AdaptiveCVResponse)(nil), assert.AnError)

	// Request avec thème invalide
	req := httptest.NewRequest("GET", "/api/v1/cv?theme=invalidtheme123", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "Invalid theme")

	mockService.AssertExpectations(t)
}

// Test erreur database
func TestGetExperiences_DatabaseError(t *testing.T) {
	// Setup
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/experiences", handler.GetExperiences)

	// Mock error
	mockService.On("GetAllExperiences", mock.Anything).
		Return([]models.Experience{}, assert.AnError)

	// Request
	req := httptest.NewRequest("GET", "/api/v1/experiences", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "Failed to fetch")

	mockService.AssertExpectations(t)
}

// Benchmark endpoint GET /api/v1/cv
func BenchmarkGetCV(b *testing.B) {
	app := fiber.New()
	mockService := new(MockCVService)
	handler := &CVHandler{cvService: mockService}

	app.Get("/api/v1/cv", handler.GetAdaptiveCV)

	cvResp := &services.AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend",
		},
		Experiences: []models.Experience{
			{Title: "Backend Dev", Company: "Test", StartDate: time.Now()},
		},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	mockService.On("GetAdaptiveCV", mock.Anything, "fullstack").Return(cvResp, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/cv", nil)
		app.Test(req)
	}
}
