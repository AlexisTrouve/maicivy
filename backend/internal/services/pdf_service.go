package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFService gère la génération de PDFs
type PDFService struct {
	templates *template.Template
}

// NewPDFService crée une nouvelle instance
func NewPDFService() *PDFService {
	// Charger templates (à créer dans templates/cv/)
	tmpl, err := template.ParseGlob("templates/cv/*.html")
	if err != nil {
		// Si templates pas encore créés, utiliser template par défaut
		tmpl = template.New("cv_base.html")
	}

	return &PDFService{
		templates: tmpl,
	}
}

// GenerateCVPDF génère un PDF du CV
func (s *PDFService) GenerateCVPDF(cv *AdaptiveCVResponse, lang string) ([]byte, error) {
	// 1. Générer HTML depuis template avec la langue
	html, err := s.renderCVHTML(cv, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}

	// 2. Configure chromedp allocator options for container environment
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox, // Required for running in container as non-root
		chromedp.Headless,
		chromedp.Flag("disable-dev-shm-usage", true), // Overcome limited /dev/shm in containers
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("single-process", true),
	)

	// Check for custom Chrome path (Alpine uses chromium-browser)
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else if _, err := os.Stat("/usr/bin/chromium-browser"); err == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/chromium-browser"))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create browser context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Timeout pour génération PDF
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var pdfBuffer []byte

	if err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Injecter HTML encodé en base64 pour préserver UTF-8
			encoded := base64.StdEncoding.EncodeToString([]byte(html))
			script := fmt.Sprintf(`
				const bytes = Uint8Array.from(atob('%s'), c => c.charCodeAt(0));
				const html = new TextDecoder('utf-8').decode(bytes);
				document.open('text/html', 'replace');
				document.write(html);
				document.close();
			`, encoded)
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Attendre que les fonts soient chargées
			script := `
				new Promise((resolve) => {
					if (document.fonts) {
						document.fonts.ready.then(() => {
							setTimeout(resolve, 500);
						});
					} else {
						setTimeout(resolve, 2000);
					}
				});
			`
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Générer PDF avec options pour préserver UTF-8
			var err error
			pdfBuffer, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	); err != nil {
		return nil, fmt.Errorf("chromedp failed: %w", err)
	}

	return pdfBuffer, nil
}

// renderCVHTML génère le HTML du CV depuis template
func (s *PDFService) renderCVHTML(cv *AdaptiveCVResponse, lang string) (string, error) {
	var buf strings.Builder

	// Préparer les données avec les traductions
	data := struct {
		*AdaptiveCVResponse
		Lang   string
		Labels map[string]string
	}{
		AdaptiveCVResponse: cv,
		Lang:               lang,
		Labels:             getLabels(lang),
	}

	// Vérifier si template existe
	tmpl := s.templates.Lookup("cv_base.html")
	if tmpl == nil {
		// Utiliser template basique si pas trouvé
		return s.renderBasicHTML(cv, lang), nil
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// getLabels retourne les labels traduits pour une langue
func getLabels(lang string) map[string]string {
	labels := map[string]map[string]string{
		"fr": {
			"cv":                   "CV",
			"experiences":          "Expériences Professionnelles",
			"skills":               "Compétences",
			"projects":             "Projets",
			"present":              "Présent",
			"years":                "ans",
			"generated_on":         "Généré le",
			"theme":                "Thème",
		},
		"en": {
			"cv":                   "Resume",
			"experiences":          "Professional Experience",
			"skills":               "Skills",
			"projects":             "Projects",
			"present":              "Present",
			"years":                "years",
			"generated_on":         "Generated on",
			"theme":                "Theme",
		},
	}

	if l, ok := labels[lang]; ok {
		return l
	}
	return labels["fr"] // Default to French
}

// renderBasicHTML génère un HTML simple si template pas disponible
func (s *PDFService) renderBasicHTML(cv *AdaptiveCVResponse, lang string) string {
	var buf strings.Builder
	labels := getLabels(lang)

	buf.WriteString(`<!DOCTYPE html>
<html lang="` + lang + `">
<head>
    <meta charset="UTF-8">
    <title>` + labels["cv"] + ` - ` + cv.Theme.Name + `</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 20px; }
        h1 { color: #2563eb; }
        h2 { color: #1e40af; margin-top: 20px; }
        .section { margin-bottom: 20px; }
        .item { margin-bottom: 10px; padding-left: 10px; border-left: 2px solid #e2e8f0; }
    </style>
</head>
<body>
    <h1>` + labels["cv"] + ` - ` + cv.Theme.Name + `</h1>
    <p>` + cv.Theme.Description + `</p>
`)

	// Expériences
	if len(cv.Experiences) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["experiences"] + `</h2>`)
		for _, exp := range cv.Experiences {
			buf.WriteString(fmt.Sprintf(`<div class="item">
				<strong>%s</strong> - %s<br>
				%s
			</div>`, exp.Title, exp.Company, exp.Description))
		}
		buf.WriteString(`</div>`)
	}

	// Skills
	if len(cv.Skills) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["skills"] + `</h2>`)
		for _, skill := range cv.Skills {
			buf.WriteString(fmt.Sprintf(`<div class="item">
				<strong>%s</strong> - %s (%d %s)
			</div>`, skill.Name, skill.Level, skill.YearsExperience, labels["years"]))
		}
		buf.WriteString(`</div>`)
	}

	// Projets
	if len(cv.Projects) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["projects"] + `</h2>`)
		for _, project := range cv.Projects {
			buf.WriteString(fmt.Sprintf(`<div class="item">
				<strong>%s</strong><br>
				%s
			</div>`, project.Title, project.Description))
		}
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</body></html>`)
	return buf.String()
}
