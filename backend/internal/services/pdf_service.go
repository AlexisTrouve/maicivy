package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"maicivy/internal/models"
)

// PDFService gère la génération de PDFs
type PDFService struct {
	templates    *template.Template
	hasTemplates bool       // true si des templates ont été chargés depuis le disque
	profileImg   template.URL // data URI base64 de la photo de profil (vide si absent)
}

// SkillGroup regroupe les compétences par catégorie pour le template
type SkillGroup struct {
	Category string
	Skills   []ScoredSkillResponse
}

// NewPDFService crée une nouvelle instance
func NewPDFService() *PDFService {
	// FuncMap : fonctions utilitaires disponibles dans les templates HTML
	funcMap := template.FuncMap{
		// levelWidth convertit un models.SkillLevel en largeur de barre (0-100)
		"levelWidth": func(level models.SkillLevel) int {
			switch level {
			case models.SkillLevelExpert:
				return 100
			case models.SkillLevelAdvanced:
				return 75
			case models.SkillLevelIntermediate:
				return 50
			default:
				return 25
			}
		},
		// scorePercent convertit un score float (0-1) en entier (0-100)
		"scorePercent": func(score float64) int {
			return int(score * 100)
		},
		// add additionne deux entiers (pour compteurs dans les templates)
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("cv_base.html").Funcs(funcMap).ParseGlob("templates/cv/*.html")
	if err != nil {
		// Templates pas encore créés → fallback renderBasicHTML
		return &PDFService{templates: nil, hasTemplates: false}
	}

	// Charger la photo de profil en base64 pour l'embed dans le PDF.
	// chromedp navigue sur about:blank — pas de serveur de fichiers, donc data URI obligatoire.
	var profileImg template.URL
	if imgData, err := os.ReadFile("templates/cv/profile.png"); err == nil {
		profileImg = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(imgData))
	}

	return &PDFService{
		templates:    tmpl,
		hasTemplates: true,
		profileImg:   profileImg,
	}
}

// GenerateCVPDF génère un PDF du CV
func (s *PDFService) GenerateCVPDF(cv *AdaptiveCVResponse, lang string) ([]byte, error) {
	html, err := s.renderCVHTML(cv, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}
	return s.htmlToPDF(html)
}

// GenerateTailoredPDF génère un PDF avec une couche stealth injectée avant </body>.
// Le stealthHTML est quasi-invisible visuellement mais présent dans le flux texte
// du PDF — extrait par les parseurs ATS et lu par les LLMs de screening.
func (s *PDFService) GenerateTailoredPDF(cv *AdaptiveCVResponse, lang string, stealthHTML string) ([]byte, error) {
	html, err := s.renderCVHTML(cv, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}

	// Injecter le stealth juste après <body> — premiers tokens du flux PDF,
	// prioritaires pour les parseurs ATS et les LLMs de screening.
	if stealthHTML != "" {
		if idx := strings.Index(html, "<body"); idx >= 0 {
			// Trouver la fin du tag <body ...> pour injecter après
			closeTag := strings.Index(html[idx:], ">")
			if closeTag >= 0 {
				insertAt := idx + closeTag + 1
				html = html[:insertAt] + stealthHTML + html[insertAt:]
			}
		} else {
			// Fallback : prepend si pas de <body>
			html = stealthHTML + html
		}
	}

	return s.htmlToPDF(html)
}

// htmlToPDF : convertit du HTML en PDF via ChromeDP (Chromium headless)
func (s *PDFService) htmlToPDF(html string) ([]byte, error) {
	// Configure chromedp allocator options selon l'environnement
	// Les flags container (--single-process, --disable-dev-shm-usage) causent des crashes sur Windows/macOS
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.Headless,
	)

	// Flags spécifiques aux conteneurs Linux (pas sur Windows/macOS)
	if runtime.GOOS == "linux" {
		opts = append(opts,
			chromedp.NoSandbox,
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-setuid-sandbox", true),
			chromedp.Flag("single-process", true),
		)
	}

	// Custom Chrome path: CHROME_PATH > Alpine chromium-browser > défaut
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else if _, err := os.Stat("/usr/bin/chromium-browser"); err == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/chromium-browser"))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

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

// groupSkillsByCategory regroupe les compétences par catégorie, triées par score DESC
func groupSkillsByCategory(skills []ScoredSkillResponse) []SkillGroup {
	order := []string{}
	groups := map[string][]ScoredSkillResponse{}
	for _, s := range skills {
		cat := s.Category
		if cat == "" {
			cat = "Autres"
		}
		if _, exists := groups[cat]; !exists {
			order = append(order, cat)
		}
		groups[cat] = append(groups[cat], s)
	}
	result := make([]SkillGroup, 0, len(order))
	for _, cat := range order {
		result = append(result, SkillGroup{Category: cat, Skills: groups[cat]})
	}
	return result
}

// renderCVHTML génère le HTML du CV depuis template
func (s *PDFService) renderCVHTML(cv *AdaptiveCVResponse, lang string) (string, error) {
	var buf strings.Builder

	// Top 6 projets en détail, le reste en mode compressé
	const topN = 6
	topProjects := cv.Projects
	var otherProjects []ScoredProjectResponse
	if len(cv.Projects) > topN {
		topProjects = cv.Projects[:topN]
		otherProjects = cv.Projects[topN:]
	}

	// Préparer les données avec les traductions + pré-processing
	data := struct {
		*AdaptiveCVResponse
		Lang          string
		Labels        map[string]string
		TopProjects   []ScoredProjectResponse
		OtherProjects []ScoredProjectResponse // projets 7+ en format compact
		SkillGroups   []SkillGroup
		ProfileImg    template.URL // data URI base64 de la photo (vide si absent)
	}{
		AdaptiveCVResponse: cv,
		Lang:               lang,
		Labels:             getLabels(lang),
		TopProjects:        topProjects,
		OtherProjects:      otherProjects,
		SkillGroups:        groupSkillsByCategory(cv.Skills),
		ProfileImg:         s.profileImg,
	}

	// Utiliser template HTML si disponible, sinon fallback renderBasicHTML
	if !s.hasTemplates {
		return s.renderBasicHTML(cv, lang), nil
	}

	tmpl := s.templates.Lookup("cv_base.html")
	if tmpl == nil {
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
			"yr":                   "a",
			"generated_on":         "Généré le",
			"theme":                "Thème",
			"other_experiences":    "Autres expériences",
			"other_projects":       "Autres projets",
		},
		"en": {
			"cv":                   "Resume",
			"experiences":          "Professional Experience",
			"skills":               "Skills",
			"projects":             "Projects",
			"present":              "Present",
			"years":                "years",
			"yr":                   "y",
			"generated_on":         "Generated on",
			"theme":                "Theme",
			"other_experiences":    "Other experience",
			"other_projects":       "Other projects",
		},
	}

	if l, ok := labels[lang]; ok {
		return l
	}
	return labels["fr"] // Default to French
}

// scoreThreshold détermine si une expérience/skill est "pertinent" (rendu détaillé vs compact)
const scoreThreshold = 0.1

// maxDetailedProjectsPDF : top N projets en détail, le reste en section "Autres projets"
// Les projets arrivent déjà triés par score LLM DESC dans AdaptiveCVResponse
const maxDetailedProjectsPDF = 6

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
        body { font-family: Arial, sans-serif; padding: 20px; color: #1a1a2e; line-height: 1.5; }
        h1 { color: #2563eb; margin-bottom: 4px; }
        h2 { color: #1e40af; margin-top: 24px; margin-bottom: 8px; border-bottom: 1px solid #e2e8f0; padding-bottom: 4px; }
        h3 { color: #475569; font-size: 0.9em; margin-top: 16px; margin-bottom: 6px; }
        .section { margin-bottom: 16px; }
        .item { margin-bottom: 10px; padding-left: 10px; border-left: 2px solid #3b82f6; }
        .item-minor { margin-bottom: 4px; padding-left: 10px; border-left: 2px solid #e2e8f0; color: #64748b; font-size: 0.9em; }
        .skill-grid { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
        .skill-tag { background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 4px; padding: 2px 8px; font-size: 0.85em; }
        .skill-tag-minor { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 4px; padding: 2px 8px; font-size: 0.8em; color: #94a3b8; }
        .dates { color: #64748b; font-size: 0.9em; }
        .tech { color: #6366f1; font-size: 0.85em; }
    </style>
</head>
<body>
    <h1>` + labels["cv"] + ` - ` + cv.Theme.Name + `</h1>
    <p>` + cv.Theme.Description + `</p>
`)

	// Expériences — pertinentes en détail, autres en compact
	if len(cv.Experiences) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["experiences"] + `</h2>`)
		hasMinor := false
		for _, exp := range cv.Experiences {
			if exp.Score > scoreThreshold {
				dates := formatDate(exp.StartDate.Format("01/2006"))
				if exp.EndDate != nil {
					dates += " - " + exp.EndDate.Format("01/2006")
				} else {
					dates += " - " + labels["present"]
				}
				buf.WriteString(fmt.Sprintf(`<div class="item">
					<strong>%s</strong> - %s <span class="dates">(%s)</span><br>
					%s`, exp.Title, exp.Company, dates, exp.Description))
				if len(exp.Technologies) > 0 {
					buf.WriteString(`<br><span class="tech">` + strings.Join(exp.Technologies, ", ") + `</span>`)
				}
				buf.WriteString(`</div>`)
			} else {
				hasMinor = true
			}
		}
		if hasMinor {
			buf.WriteString(`<h3>` + labels["other_experiences"] + `</h3>`)
			for _, exp := range cv.Experiences {
				if exp.Score <= scoreThreshold {
					dates := exp.StartDate.Format("2006")
					if exp.EndDate != nil {
						dates += "-" + exp.EndDate.Format("2006")
					}
					buf.WriteString(fmt.Sprintf(`<div class="item-minor">
						<strong>%s</strong> - %s (%s)
					</div>`, exp.Title, exp.Company, dates))
				}
			}
		}
		buf.WriteString(`</div>`)
	}

	// Skills — pertinentes en tags visuels, autres en liste compacte
	if len(cv.Skills) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["skills"] + `</h2>`)
		buf.WriteString(`<div class="skill-grid">`)
		hasMinor := false
		for _, skill := range cv.Skills {
			if skill.Score > scoreThreshold {
				buf.WriteString(fmt.Sprintf(`<span class="skill-tag"><strong>%s</strong> · %s · %d%s</span>`,
					skill.Name, skill.Level, skill.YearsExperience, labels["yr"]))
			} else {
				hasMinor = true
			}
		}
		buf.WriteString(`</div>`)
		if hasMinor {
			buf.WriteString(`<div class="skill-grid" style="margin-top:4px;">`)
			for _, skill := range cv.Skills {
				if skill.Score <= scoreThreshold {
					buf.WriteString(fmt.Sprintf(`<span class="skill-tag-minor">%s</span>`, skill.Name))
				}
			}
			buf.WriteString(`</div>`)
		}
		buf.WriteString(`</div>`)
	}

	// Projets — top maxDetailedProjectsPDF en détail (déjà triés par score LLM DESC),
	// le reste en section "Autres projets" compacte
	if len(cv.Projects) > 0 {
		buf.WriteString(`<div class="section"><h2>` + labels["projects"] + `</h2>`)
		detailCount := maxDetailedProjectsPDF
		if detailCount > len(cv.Projects) {
			detailCount = len(cv.Projects)
		}
		for _, project := range cv.Projects[:detailCount] {
			buf.WriteString(fmt.Sprintf(`<div class="item">
				<strong>%s</strong><br>
				%s`, project.Title, project.Description))
			if len(project.Technologies) > 0 {
				buf.WriteString(`<br><span class="tech">` + strings.Join(project.Technologies, ", ") + `</span>`)
			}
			buf.WriteString(`</div>`)
		}
		hasMinor := len(cv.Projects) > detailCount
		if hasMinor {
			buf.WriteString(`<h3>` + labels["other_projects"] + `</h3>`)
			for _, project := range cv.Projects[detailCount:] {
				buf.WriteString(fmt.Sprintf(`<div class="item-minor">
					<strong>%s</strong> — %s
				</div>`, project.Title, project.Catchphrase))
			}
		}
		buf.WriteString(`</div>`)
	}

	buf.WriteString(`</body></html>`)
	return buf.String()
}

func formatDate(d string) string {
	return d // placeholder for custom formatting
}
