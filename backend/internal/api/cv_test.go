// backend/internal/api/cv_test.go
package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock du repository CV
type MockCVRepository struct {
	mock.Mock
}

func (m *MockCVRepository) GetExperiences() ([]*Experience, error) {
	args := m.Called()
	return args.Get(0).([]*Experience), args.Error(1)
}

func (m *MockCVRepository) GetSkills() ([]*Skill, error) {
	args := m.Called()
	return args.Get(0).([]*Skill), args.Error(1)
}

func (m *MockCVRepository) GetProjects() ([]*Project, error) {
	args := m.Called()
	return args.Get(0).([]*Project), args.Error(1)
}

func (m *MockCVRepository) GetThemes() ([]*Theme, error) {
	args := m.Called()
	return args.Get(0).([]*Theme), args.Error(1)
}

// Mock du service de scoring
type MockCVScoringService struct {
	mock.Mock
}

func (m *MockCVScoringService) ScoreAndFilterCV(experiences []*Experience, skills []*Skill, theme string) (*CVResponse, error) {
	args := m.Called(experiences, skills, theme)
	return args.Get(0).(*CVResponse), args.Error(1)
}

// Test GET /api/cv (sans thème - retourne tout)
func TestGetCV_NoTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	handler := NewCVHandler(mockRepo, nil)

	app.Get("/api/cv", handler.GetCV)

	// Mock data
	experiences := []*Experience{
		{ID: 1, Title: "Backend Dev", Tags: []string{"go"}},
		{ID: 2, Title: "Frontend Dev", Tags: []string{"react"}},
	}
	skills := []*Skill{
		{ID: 1, Name: "Go", Level: "Advanced"},
		{ID: 2, Name: "React", Level: "Intermediate"},
	}
	projects := []*Project{
		{ID: 1, Title: "maicivy", Technologies: []string{"go", "react"}},
	}

	mockRepo.On("GetExperiences").Return(experiences, nil)
	mockRepo.On("GetSkills").Return(skills, nil)
	mockRepo.On("GetProjects").Return(projects, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/cv", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response body
	var cvResp CVResponse
	json.NewDecoder(resp.Body).Decode(&cvResp)

	assert.Len(t, cvResp.Experiences, 2)
	assert.Len(t, cvResp.Skills, 2)
	assert.Len(t, cvResp.Projects, 1)

	mockRepo.AssertExpectations(t)
}

// Test GET /api/cv?theme=backend
func TestGetCV_BackendTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	mockScoring := new(MockCVScoringService)
	handler := NewCVHandler(mockRepo, mockScoring)

	app.Get("/api/cv", handler.GetCV)

	// Mock data
	experiences := []*Experience{
		{ID: 1, Title: "Backend Dev", Tags: []string{"go", "postgresql"}},
		{ID: 2, Title: "Frontend Dev", Tags: []string{"react"}},
	}
	skills := []*Skill{
		{ID: 1, Name: "Go", Level: "Advanced", Category: "backend"},
	}

	// Résultat après scoring (filtrées)
	filteredResp := &CVResponse{
		Experiences: []*Experience{experiences[0]}, // Seulement backend
		Skills:      skills,
		Theme:       "backend",
	}

	mockRepo.On("GetExperiences").Return(experiences, nil)
	mockRepo.On("GetSkills").Return(skills, nil)
	mockRepo.On("GetProjects").Return([]*Project{}, nil)
	mockScoring.On("ScoreAndFilterCV", experiences, skills, "backend").Return(filteredResp, nil)

	// Request avec query param
	req := httptest.NewRequest("GET", "/api/cv?theme=backend", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var cvResp CVResponse
	json.NewDecoder(resp.Body).Decode(&cvResp)

	assert.Len(t, cvResp.Experiences, 1, "Devrait filtrer seulement expériences backend")
	assert.Equal(t, "Backend Dev", cvResp.Experiences[0].Title)
	assert.Equal(t, "backend", cvResp.Theme)

	mockRepo.AssertExpectations(t)
	mockScoring.AssertExpectations(t)
}

// Test GET /api/cv/themes
func TestGetThemes(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	handler := NewCVHandler(mockRepo, nil)

	app.Get("/api/cv/themes", handler.GetThemes)

	// Mock data
	themes := []*Theme{
		{ID: 1, Name: "backend", Keywords: []string{"go", "postgresql"}},
		{ID: 2, Name: "frontend", Keywords: []string{"react", "typescript"}},
		{ID: 3, Name: "devops", Keywords: []string{"docker", "kubernetes"}},
	}

	mockRepo.On("GetThemes").Return(themes, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/cv/themes", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var themesResp []Theme
	json.NewDecoder(resp.Body).Decode(&themesResp)

	assert.Len(t, themesResp, 3)
	assert.Equal(t, "backend", themesResp[0].Name)
	assert.Equal(t, "frontend", themesResp[1].Name)

	mockRepo.AssertExpectations(t)
}

// Test erreur repository
func TestGetCV_RepositoryError(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	handler := NewCVHandler(mockRepo, nil)

	app.Get("/api/cv", handler.GetCV)

	// Mock error
	mockRepo.On("GetExperiences").Return([]*Experience{}, assert.AnError)

	// Request
	req := httptest.NewRequest("GET", "/api/cv", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "database")

	mockRepo.AssertExpectations(t)
}

// Test cache Redis (hit)
func TestGetCV_CacheHit(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	mockCache := new(MockCacheService)
	handler := NewCVHandler(mockRepo, nil).WithCache(mockCache)

	app.Get("/api/cv", handler.GetCV)

	// Mock cache hit
	cachedCV := &CVResponse{
		Experiences: []*Experience{{ID: 1, Title: "Cached"}},
		CachedAt:    "2025-12-08T10:00:00Z",
	}

	mockCache.On("Get", "cv:all").Return(cachedCV, nil)

	// Request
	req := httptest.NewRequest("GET", "/api/cv", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var cvResp CVResponse
	json.NewDecoder(resp.Body).Decode(&cvResp)

	assert.Equal(t, "Cached", cvResp.Experiences[0].Title)
	assert.NotEmpty(t, cvResp.CachedAt)

	// Repository NE DEVRAIT PAS être appelé (cache hit)
	mockRepo.AssertNotCalled(t, "GetExperiences")
	mockCache.AssertExpectations(t)
}

// Test cache Redis (miss + set)
func TestGetCV_CacheMiss(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	mockCache := new(MockCacheService)
	handler := NewCVHandler(mockRepo, nil).WithCache(mockCache)

	app.Get("/api/cv", handler.GetCV)

	// Mock cache miss
	mockCache.On("Get", "cv:all").Return(nil, assert.AnError)

	// Mock repository
	experiences := []*Experience{{ID: 1, Title: "Fresh Data"}}
	mockRepo.On("GetExperiences").Return(experiences, nil)
	mockRepo.On("GetSkills").Return([]*Skill{}, nil)
	mockRepo.On("GetProjects").Return([]*Project{}, nil)

	// Mock cache set
	mockCache.On("Set", "cv:all", mock.Anything, 10*time.Minute).Return(nil)

	// Request
	req := httptest.NewRequest("GET", "/api/cv", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// Test validation thème invalide
func TestGetCV_InvalidTheme(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	handler := NewCVHandler(mockRepo, nil)

	app.Get("/api/cv", handler.GetCV)

	// Request avec thème invalide
	req := httptest.NewRequest("GET", "/api/cv?theme=invalidtheme123", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errorResp)

	assert.Contains(t, errorResp["error"], "invalid theme")
}

// Benchmark endpoint GET /api/cv
func BenchmarkGetCV(b *testing.B) {
	app := fiber.New()
	mockRepo := new(MockCVRepository)
	handler := NewCVHandler(mockRepo, nil)

	app.Get("/api/cv", handler.GetCV)

	experiences := []*Experience{
		{ID: 1, Title: "Backend Dev", Tags: []string{"go"}},
	}
	mockRepo.On("GetExperiences").Return(experiences, nil)
	mockRepo.On("GetSkills").Return([]*Skill{}, nil)
	mockRepo.On("GetProjects").Return([]*Project{}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/cv", nil)
		app.Test(req)
	}
}
