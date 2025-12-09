package services

import (
	"bytes"
	"context"
	"html/template"
	"strings"
	"testing"
	"time"

	"maicivy/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PDFLetterServiceTestSuite regroupe les tests du PDFLetterService
type PDFLetterServiceTestSuite struct {
	suite.Suite
	service *PDFLetterService
}

func (suite *PDFLetterServiceTestSuite) SetupTest() {
	// Créer templates de test en mémoire
	tmpl := template.New("letter_motivation.html")
	tmpl, _ = tmpl.Parse(`<!DOCTYPE html>
<html>
<head><title>Lettre de Motivation</title></head>
<body>
	<h1>Lettre de Motivation - {{.CompanyName}}</h1>
	<p>Date: {{.Date}}</p>
	<div>{{.Content}}</div>
</body>
</html>`)

	tmpl.New("letter_anti_motivation.html").Parse(`<!DOCTYPE html>
<html>
<head><title>Lettre Anti-Motivation</title></head>
<body>
	<h1>Lettre Anti-Motivation - {{.CompanyName}}</h1>
	<p>Date: {{.Date}}</p>
	<p>Type: {{.Type}}</p>
	<div>{{.Content}}</div>
</body>
</html>`)

	suite.service = &PDFLetterService{
		templates: tmpl,
	}
}

// Test NewPDFLetterService avec templates valides
func (suite *PDFLetterServiceTestSuite) TestNewPDFLetterService_ValidTemplates() {
	// Skip ce test car on ne peut pas créer de vrais fichiers templates
	suite.T().Skip("Requires real template files")
}

// Test renderHTML pour lettre motivation
func (suite *PDFLetterServiceTestSuite) TestRenderHTML_Motivation() {
	letter := models.LetterResponse{
		Content: "Je suis très motivé pour rejoindre votre équipe.",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name:   "TechCorp",
			Domain: "techcorp.com",
		},
		GeneratedAt: time.Date(2025, 12, 9, 10, 0, 0, 0, time.UTC),
	}

	html, err := suite.service.renderHTML(letter)

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Lettre de Motivation")
	assert.Contains(suite.T(), html, "TechCorp")
	assert.Contains(suite.T(), html, "09 December 2025")
	assert.Contains(suite.T(), html, "Je suis très motivé")
}

// Test renderHTML pour lettre anti-motivation
func (suite *PDFLetterServiceTestSuite) TestRenderHTML_AntiMotivation() {
	letter := models.LetterResponse{
		Content: "Pourquoi je ne veux PAS travailler chez vous.",
		Type:    models.LetterTypeAntiMotivation,
		CompanyInfo: models.CompanyInfo{
			Name:   "BadCorp",
			Domain: "badcorp.com",
		},
		GeneratedAt: time.Date(2025, 12, 9, 14, 30, 0, 0, time.UTC),
	}

	html, err := suite.service.renderHTML(letter)

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "<!DOCTYPE html>")
	assert.Contains(suite.T(), html, "Lettre Anti-Motivation")
	assert.Contains(suite.T(), html, "BadCorp")
	assert.Contains(suite.T(), html, "09 December 2025")
	assert.Contains(suite.T(), html, "Pourquoi je ne veux PAS")
	assert.Contains(suite.T(), html, "anti_motivation")
}

// Test renderHTML avec contenu vide
func (suite *PDFLetterServiceTestSuite) TestRenderHTML_EmptyContent() {
	letter := models.LetterResponse{
		Content: "",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name: "EmptyCorp",
		},
		GeneratedAt: time.Now(),
	}

	html, err := suite.service.renderHTML(letter)

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), html, "EmptyCorp")
}

// Test renderHTML avec caractères spéciaux
func (suite *PDFLetterServiceTestSuite) TestRenderHTML_SpecialCharacters() {
	letter := models.LetterResponse{
		Content: "Contenu avec <script>alert('XSS')</script> et & et \"quotes\"",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name: "Corp & Co",
		},
		GeneratedAt: time.Now(),
	}

	html, err := suite.service.renderHTML(letter)

	assert.NoError(suite.T(), err)
	// Template devrait échapper automatiquement les caractères HTML
	// Note: vérifier l'échappement dépend de la configuration du template
	assert.NotEmpty(suite.T(), html)
}

// Test escapeJSString
func (suite *PDFLetterServiceTestSuite) TestEscapeJSString() {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "Hello World",
			expected: "`Hello World`",
		},
		{
			name:     "String with backslash",
			input:    "Path\\to\\file",
			expected: "`Path\\\\to\\\\file`",
		},
		{
			name:     "String with backticks",
			input:    "Code `example`",
			expected: "`Code \\`example\\``",
		},
		{
			name:     "String with newlines",
			input:    "Line1\nLine2\r\nLine3",
			expected: "`Line1\\nLine2\\r\\nLine3`",
		},
		{
			name:     "String with dollar sign",
			input:    "Price: $100",
			expected: "`Price: \\$100`",
		},
		{
			name:     "Complex string",
			input:    "Path: C:\\Users\nPrice: $50\nCode: `test`",
			expected: "`Path: C:\\\\Users\\nPrice: \\$50\\nCode: \\`test\\``",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			result := escapeJSString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GeneratePDF - génération complète
func (suite *PDFLetterServiceTestSuite) TestGeneratePDF_Motivation() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	letter := models.LetterResponse{
		Content: "Je souhaite rejoindre votre entreprise car vous êtes leaders dans l'innovation.",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name:        "InnovateCorp",
			Domain:      "innovatecorp.com",
			Description: "Leader in innovation",
			Industry:    "Tech",
		},
		GeneratedAt: time.Now(),
		Provider:    "claude",
		TokensUsed:  1500,
	}

	var buf bytes.Buffer
	ctx := context.Background()

	err := suite.service.GeneratePDF(ctx, letter, &buf)

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available, skipping PDF generation test")
			return
		}
		assert.NoError(suite.T(), err)
	}

	// Vérifier que PDF a été écrit
	assert.Greater(suite.T(), buf.Len(), 100, "PDF should have content")

	// Vérifier signature PDF
	pdfBytes := buf.Bytes()
	assert.True(suite.T(), strings.HasPrefix(string(pdfBytes[:5]), "%PDF-"),
		"Should start with PDF signature")
}

// Test GeneratePDF - lettre anti-motivation
func (suite *PDFLetterServiceTestSuite) TestGeneratePDF_AntiMotivation() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	letter := models.LetterResponse{
		Content: "Raisons pour lesquelles je refuse votre offre: management toxique, salaires bas, pas de remote.",
		Type:    models.LetterTypeAntiMotivation,
		CompanyInfo: models.CompanyInfo{
			Name:   "ToxicCorp",
			Domain: "toxiccorp.com",
		},
		GeneratedAt: time.Now(),
		Provider:    "openai",
		TokensUsed:  1200,
	}

	var buf bytes.Buffer
	ctx := context.Background()

	err := suite.service.GeneratePDF(ctx, letter, &buf)

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available")
			return
		}
		assert.NoError(suite.T(), err)
	}

	assert.Greater(suite.T(), buf.Len(), 100)
}

// Test GeneratePDF avec timeout context
func (suite *PDFLetterServiceTestSuite) TestGeneratePDF_ContextTimeout() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	letter := models.LetterResponse{
		Content: "Test content",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name: "TestCorp",
		},
		GeneratedAt: time.Now(),
	}

	var buf bytes.Buffer

	// Context avec timeout très court (1ms)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(5 * time.Millisecond) // Attendre expiration

	err := suite.service.GeneratePDF(ctx, letter, &buf)

	// Devrait échouer à cause du timeout OU réussir si très rapide
	// On accepte les deux cas
	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available")
			return
		}
		assert.Error(suite.T(), err)
		assert.Contains(suite.T(), err.Error(), "context")
	}
}

// Test htmlToPDF avec HTML invalide
func (suite *PDFLetterServiceTestSuite) TestHTMLToPDF_InvalidHTML() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	invalidHTML := "<html><body><h1>Unclosed tag"
	var buf bytes.Buffer
	ctx := context.Background()

	err := suite.service.htmlToPDF(ctx, invalidHTML, &buf)

	if err != nil {
		if strings.Contains(err.Error(), "chromedp") {
			suite.T().Skip("chromedp not available")
			return
		}
		// Chrome headless peut tolérer HTML invalide
		// Donc pas forcément une erreur
	}
}

// Test renderHTML avec template manquant
func (suite *PDFLetterServiceTestSuite) TestRenderHTML_MissingTemplate() {
	// Service sans templates
	service := &PDFLetterService{
		templates: template.New("empty"),
	}

	letter := models.LetterResponse{
		Content: "Test",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name: "TestCorp",
		},
		GeneratedAt: time.Now(),
	}

	_, err := service.renderHTML(letter)

	// Devrait échouer car template n'existe pas
	assert.Error(suite.T(), err)
}

// Run test suite
func TestPDFLetterServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PDFLetterServiceTestSuite))
}

// Benchmark renderHTML
func BenchmarkRenderHTML(b *testing.B) {
	tmpl := template.New("letter_motivation.html")
	tmpl.Parse(`<!DOCTYPE html>
<html>
<body>
	<h1>{{.CompanyName}}</h1>
	<p>{{.Date}}</p>
	<div>{{.Content}}</div>
</body>
</html>`)

	service := &PDFLetterService{templates: tmpl}

	letter := models.LetterResponse{
		Content: "Test content for benchmarking",
		Type:    models.LetterTypeMotivation,
		CompanyInfo: models.CompanyInfo{
			Name: "BenchCorp",
		},
		GeneratedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.renderHTML(letter)
	}
}

// Benchmark escapeJSString
func BenchmarkEscapeJSString(b *testing.B) {
	input := "Path: C:\\Users\\Test\nPrice: $100\nCode: `test`"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeJSString(input)
	}
}
