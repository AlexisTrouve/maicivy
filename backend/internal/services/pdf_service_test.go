package services

import (
	"strings"
	"testing"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PDFServiceTestSuite regroupe les tests du PDFService
type PDFServiceTestSuite struct {
	suite.Suite
	service *PDFService
}

func (suite *PDFServiceTestSuite) SetupTest() {
	suite.service = NewPDFService()
}

// Test NewPDFService crée une instance valide
func (suite *PDFServiceTestSuite) TestNewPDFService() {
	service := NewPDFService()

	// templates peut être nil si les fichiers ne sont pas dans le path courant (tests locaux)
	assert.NotNil(suite.T(), service)
}

// Test renderBasicHTML génère du HTML valide
func (suite *PDFServiceTestSuite) TestRenderBasicHTML() {
	now := time.Now()
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Expert in backend development",
		},
		Experiences: []ScoredExperienceResponse{
			{Experience: models.Experience{
				Title:       "Senior Backend Developer",
				Company:     "TechCorp",
				Description: "Building scalable APIs",
				StartDate:   now.AddDate(-2, 0, 0),
			}},
		},
		Skills: []ScoredSkillResponse{
			{Skill: models.Skill{
				Name:            "Go",
				Level:           models.SkillLevelExpert,
				YearsExperience: 5,
			}},
		},
		Projects: []ScoredProjectResponse{
			{Project: models.Project{
				Title:       "maicivy",
				Description: "Interactive CV with AI",
			}},
		},
		GeneratedAt: now,
	}

	html := suite.service.renderBasicHTML(cv, "fr")

	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Backend Developer")
	assert.Contains(suite.T(), html, "Senior Backend Developer")
	assert.Contains(suite.T(), html, "TechCorp")
	assert.Contains(suite.T(), html, "Go")
	assert.Contains(suite.T(), html, "maicivy")
}

// Test renderBasicHTML avec CV vide
func (suite *PDFServiceTestSuite) TestRenderBasicHTML_EmptyCV() {
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "fullstack",
			Name:        "Full-Stack Developer",
			Description: "Full-stack expertise",
		},
		Experiences: []ScoredExperienceResponse{},
		Skills:      []ScoredSkillResponse{},
		Projects:    []ScoredProjectResponse{},
		GeneratedAt: time.Now(),
	}

	html := suite.service.renderBasicHTML(cv, "fr")

	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Full-Stack Developer")
}

// Test renderBasicHTML avec expérience actuelle (pas de EndDate)
func (suite *PDFServiceTestSuite) TestRenderBasicHTML_CurrentJob() {
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend",
		},
		Experiences: []ScoredExperienceResponse{
			{Experience: models.Experience{
				Title:     "Current Job",
				Company:   "CurrentCorp",
				StartDate: time.Now(),
				EndDate:   nil,
			}},
		},
		Skills:      []ScoredSkillResponse{},
		Projects:    []ScoredProjectResponse{},
		GeneratedAt: time.Now(),
	}

	html := suite.service.renderBasicHTML(cv, "fr")

	assert.Contains(suite.T(), html, "Current Job")
	assert.Contains(suite.T(), html, "CurrentCorp")
}

// Test renderCVHTML avec template inexistant (fallback vers basic)
func (suite *PDFServiceTestSuite) TestRenderCVHTML_NoTemplate() {
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend Developer",
		},
		Experiences: []ScoredExperienceResponse{},
		Skills:      []ScoredSkillResponse{},
		Projects:    []ScoredProjectResponse{},
		GeneratedAt: time.Now(),
	}

	html, err := suite.service.renderCVHTML(cv, "fr", "")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Backend Developer")
}

// Test GenerateCVPDF - génération complète (nécessite Chrome headless)
func (suite *PDFServiceTestSuite) TestGenerateCVPDF_Integration() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Backend expertise",
		},
		Experiences: []ScoredExperienceResponse{
			{Experience: models.Experience{
				Title:        "Backend Dev",
				Company:      "TechCorp",
				Description:  "Building APIs",
				Tags:         pq.StringArray{"go", "postgresql"},
				Technologies: pq.StringArray{"go", "postgresql"},
				StartDate:    time.Now().AddDate(-2, 0, 0),
			}},
		},
		Skills: []ScoredSkillResponse{
			{Skill: models.Skill{
				Name:            "Go",
				Level:           models.SkillLevelExpert,
				Category:        "backend",
				YearsExperience: 5,
			}},
		},
		Projects: []ScoredProjectResponse{
			{Project: models.Project{
				Title:        "maicivy",
				Description:  "Interactive CV",
				Technologies: pq.StringArray{"go", "react"},
				Category:     "fullstack",
			}},
		},
		GeneratedAt: time.Now(),
	}

	pdfBytes, err := suite.service.GenerateCVPDF(cv, "fr")

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available, skipping PDF generation test")
			return
		}
		assert.NoError(suite.T(), err)
	}

	assert.NotNil(suite.T(), pdfBytes)
	assert.Greater(suite.T(), len(pdfBytes), 100, "PDF should have content")
	assert.True(suite.T(), strings.HasPrefix(string(pdfBytes[:5]), "%PDF-"),
		"Should start with PDF signature")
}

// Test GenerateCVPDF avec CV vide
func (suite *PDFServiceTestSuite) TestGenerateCVPDF_EmptyCV() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "fullstack",
			Name: "Full-Stack",
		},
		Experiences: []ScoredExperienceResponse{},
		Skills:      []ScoredSkillResponse{},
		Projects:    []ScoredProjectResponse{},
		GeneratedAt: time.Now(),
	}

	pdfBytes, err := suite.service.GenerateCVPDF(cv, "en")

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available")
			return
		}
		assert.NoError(suite.T(), err)
	}

	assert.NotNil(suite.T(), pdfBytes)
	assert.Greater(suite.T(), len(pdfBytes), 50)
}

// Run test suite
func TestPDFServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PDFServiceTestSuite))
}

// Benchmark renderBasicHTML
func BenchmarkRenderBasicHTML(b *testing.B) {
	service := NewPDFService()
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend Developer",
		},
		Experiences: []ScoredExperienceResponse{
			{Experience: models.Experience{Title: "Dev", Company: "Corp", StartDate: time.Now()}},
		},
		Skills: []ScoredSkillResponse{
			{Skill: models.Skill{Name: "Go", Level: models.SkillLevelExpert}},
		},
		Projects:    []ScoredProjectResponse{},
		GeneratedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.renderBasicHTML(cv, "fr")
	}
}
