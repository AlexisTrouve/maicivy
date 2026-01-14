package services

import (
	"html/template"
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
	// Créer service avec template par défaut
	suite.service = NewPDFService()
}

// Test NewPDFService crée une instance valide
func (suite *PDFServiceTestSuite) TestNewPDFService() {
	service := NewPDFService()

	assert.NotNil(suite.T(), service)
	assert.NotNil(suite.T(), service.templates)
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
		Experiences: []models.Experience{
			{
				Title:       "Senior Backend Developer",
				Company:     "TechCorp",
				Description: "Building scalable APIs",
				StartDate:   now.AddDate(-2, 0, 0),
			},
		},
		Skills: []models.Skill{
			{
				Name:            "Go",
				Level:           models.SkillLevelExpert,
				YearsExperience: 5,
			},
		},
		Projects: []models.Project{
			{
				Title:       "maicivy",
				Description: "Interactive CV with AI",
			},
		},
		GeneratedAt: now,
	}

	html := suite.service.renderBasicHTML(cv)

	// Assertions
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Backend Developer")
	assert.Contains(suite.T(), html, "Expert in backend development")
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
		Experiences: []models.Experience{},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	html := suite.service.renderBasicHTML(cv)

	// Doit contenir structure HTML même si vide
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Full-Stack Developer")
	assert.Contains(suite.T(), html, "Full-stack expertise")
}

// Test renderBasicHTML avec expérience actuelle (pas de EndDate)
func (suite *PDFServiceTestSuite) TestRenderBasicHTML_CurrentJob() {
	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend",
		},
		Experiences: []models.Experience{
			{
				Title:     "Current Job",
				Company:   "CurrentCorp",
				StartDate: time.Now(),
				EndDate:   nil, // Emploi actuel
			},
		},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	html := suite.service.renderBasicHTML(cv)

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
		Experiences: []models.Experience{},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	html, err := suite.service.renderCVHTML(cv)

	// Pas d'erreur, fallback vers basic
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Backend Developer")
}

// Test renderCVHTML avec template custom
func (suite *PDFServiceTestSuite) TestRenderCVHTML_CustomTemplate() {
	// Créer template custom
	tmplStr := `<!DOCTYPE html>
<html>
<head><title>{{.Theme.Name}}</title></head>
<body>
	<h1>{{.Theme.Name}}</h1>
	{{range .Experiences}}<p>{{.Title}} @ {{.Company}}</p>{{end}}
</body>
</html>`

	tmpl, err := template.New("cv_base.html").Parse(tmplStr)
	assert.NoError(suite.T(), err)

	suite.service.templates = tmpl

	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:   "backend",
			Name: "Backend Dev",
		},
		Experiences: []models.Experience{
			{Title: "Dev", Company: "Corp"},
		},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	html, err := suite.service.renderCVHTML(cv)

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "<h1>Backend Dev</h1>")
	assert.Contains(suite.T(), html, "<p>Dev @ Corp</p>")
}

// Test GenerateCVPDF - génération complète
// NOTE: Ce test nécessite chromedp (Chrome headless) installé
// On peut le skip en environnement CI si Chrome n'est pas disponible
func (suite *PDFServiceTestSuite) TestGenerateCVPDF_Integration() {
	// Skip si environnement CI sans Chrome
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	cv := &AdaptiveCVResponse{
		Theme: config.CVTheme{
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Backend expertise",
		},
		Experiences: []models.Experience{
			{
				Title:        "Backend Dev",
				Company:      "TechCorp",
				Description:  "Building APIs",
				Tags:         pq.StringArray{"go", "postgresql"},
				Technologies: pq.StringArray{"go", "postgresql"},
				StartDate:    time.Now().AddDate(-2, 0, 0),
			},
		},
		Skills: []models.Skill{
			{
				Name:            "Go",
				Level:           models.SkillLevelExpert,
				Category:        "backend",
				YearsExperience: 5,
			},
		},
		Projects: []models.Project{
			{
				Title:        "maicivy",
				Description:  "Interactive CV",
				Technologies: pq.StringArray{"go", "react"},
				Category:     "fullstack",
			},
		},
		GeneratedAt: time.Now(),
	}

	pdfBytes, err := suite.service.GenerateCVPDF(cv)

	// Si chromedp n'est pas disponible, erreur attendue
	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available, skipping PDF generation test")
			return
		}
		assert.NoError(suite.T(), err)
	}

	// Vérifier que PDF n'est pas vide
	assert.NotNil(suite.T(), pdfBytes)
	assert.Greater(suite.T(), len(pdfBytes), 100, "PDF should have content")

	// Vérifier signature PDF (commence par %PDF-)
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
		Experiences: []models.Experience{},
		Skills:      []models.Skill{},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	pdfBytes, err := suite.service.GenerateCVPDF(cv)

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available")
			return
		}
		assert.NoError(suite.T(), err)
	}

	// Doit générer un PDF même vide
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
		Experiences: []models.Experience{
			{Title: "Dev", Company: "Corp", StartDate: time.Now()},
		},
		Skills: []models.Skill{
			{Name: "Go", Level: models.SkillLevelExpert},
		},
		Projects:    []models.Project{},
		GeneratedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.renderBasicHTML(cv)
	}
}
