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
	hasTemplates bool         // true si des templates ont été chargés depuis le disque
	profileImg   template.URL // data URI base64 de la photo de profil (vide si absent)
}

// SkillGroup regroupe les compétences par catégorie pour le template
type SkillGroup struct {
	Category string
	Skills   []ScoredSkillResponse
}

// NewPDFService crée une nouvelle instance
func NewPDFService() *PDFService {
	return newPDFServiceFrom("templates/cv")
}

// newPDFServiceFrom charge les gabarits depuis `dir` (relatif au cwd). Séparé de NewPDFService
// pour que les tests — dont le cwd est internal/services — rendent le VRAI gabarit.
func newPDFServiceFrom(dir string) *PDFService {
	funcMap := cvTemplateFuncs()

	tmpl, err := template.New("cv_base.html").Funcs(funcMap).ParseGlob(dir + "/*.html")
	if err != nil {
		// Templates pas encore créés → fallback renderBasicHTML
		return &PDFService{templates: nil, hasTemplates: false}
	}

	// Charger la photo de profil en base64 pour l'embed dans le PDF.
	// chromedp navigue sur about:blank — pas de serveur de fichiers, donc data URI obligatoire.
	//
	// POIDS : la photo est affichée à 76 px CSS, mais l'ancienne source (PNG 864×1184, 1,3 Mo) était
	// embarquée en pleine définition — Chromium la recompressait en Flate : 1,85 Mo sur un PDF de
	// 2,08 Mo (mesuré le 09/11 sur /cv/export). profile.jpg est le carré HAUT de l'original (ce que
	// `object-fit:cover; object-position:center top` montrait déjà) réduit à 320 px (> 4× l'affichage,
	// > 400 dpi imprimé) en JPEG q90 4:4:4 : 34 Ko, et Chromium embarque un JPEG tel quel (DCTDecode).
	var profileImg template.URL
	if imgData, err := os.ReadFile(dir + "/profile.jpg"); err == nil {
		profileImg = template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imgData))
	}

	return &PDFService{
		templates:    tmpl,
		hasTemplates: true,
		profileImg:   profileImg,
	}
}

// cvTemplateFuncs : fonctions utilitaires disponibles dans les templates HTML du CV.
func cvTemplateFuncs() template.FuncMap {
	return template.FuncMap{
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
		// firstN : les n premiers éléments d'une liste (technos affichées sur une carte projet).
		// La liste complète reste dans la couche ATS — on ne coupe que l'affichage.
		"firstN": func(items []string, n int) []string {
			if len(items) <= n {
				return items
			}
			return items[:n]
		},
		// shortTitle : nom du projet réduit à son identité (« ProjectMind — Serveur MCP… » →
		// « ProjectMind »). L'accroche est affichée juste dessous : la répéter dans le titre
		// doublait la carte. Même règle que l'inventaire du chat (séparateur ENTOURÉ d'espaces,
		// sinon « wechat-homework-bot » deviendrait « wechat »).
		"shortTitle": nomProjetCourt,
		// chunkWords découpe un texte en phrases de max N mots séparées par ". "
		// Évite les paragraphes > 40 mots qui triggent les warnings ATS
		"chunkWords": func(text string, maxWords int) string {
			words := strings.Fields(text)
			if len(words) == 0 {
				return ""
			}
			var chunks []string
			for i := 0; i < len(words); i += maxWords {
				end := i + maxWords
				if end > len(words) {
					end = len(words)
				}
				chunks = append(chunks, strings.Join(words[i:end], " "))
			}
			return strings.Join(chunks, ". ")
		},
	}
}

// GenerateCVPDF génère un PDF du CV (sans stealth ATS)
func (s *PDFService) GenerateCVPDF(cv *AdaptiveCVResponse, lang string) ([]byte, error) {
	html, err := s.renderCVHTML(cv, lang, "")
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}
	return htmlToPDF(html)
}

// GenerateTailoredPDF génère un PDF avec une couche stealth ATS.
// stealthText : texte brut transmis au template → rendu à une unité de la couleur du fond du
// bandeau (jamais égale) → invisible visuellement, glyphes correctement encodés dans le stream PDF.
func (s *PDFService) GenerateTailoredPDF(cv *AdaptiveCVResponse, lang string, stealthText string) ([]byte, error) {
	html, err := s.renderCVHTML(cv, lang, stealthText)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}
	return htmlToPDF(html)
}

// htmlToPDF : convertit du HTML en PDF via ChromeDP (Chromium headless). Le moteur commun des
// documents imprimés du site : CV (PDFService) et dossier de conception (DossierPDFService).
// Le format de page vient du CSS du document (`@page`, PreferCSSPageSize).
func htmlToPDF(html string) ([]byte, error) {
	// Configure chromedp allocator options selon l'environnement
	// Les flags container (--single-process, --disable-dev-shm-usage) causent des crashes sur Windows/macOS
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.Headless,
	)

	// Flags spécifiques aux conteneurs Linux (pas sur Windows/macOS)
	// --single-process : requis pour Chromium ~120 en container (Debian Bookworm)
	if runtime.GOOS == "linux" {
		opts = append(opts,
			chromedp.NoSandbox,
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-setuid-sandbox", true),
			chromedp.Flag("single-process", true),
		)
	}

	// Custom Chrome path: CHROME_PATH > Debian /usr/bin/chromium > Alpine chromium-browser > défaut
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else if _, err := os.Stat("/usr/bin/chromium"); err == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/chromium"))
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

// categoriesParLangue : titres des catégories de compétences pour les langues où la clé
// anglaise de maiProFiles (« Languages », « Backend »…) ne se lit pas. fr/en affichent la clé.
// Une catégorie absente de la table passe telle quelle (nouvelle catégorie côté maiProFiles :
// mieux vaut l'anglais qu'un titre vide).
var categoriesParLangue = map[string]map[string]string{
	"zh": {
		"AI":            "人工智能",
		"Backend":       "后端",
		"Creative":      "创意",
		"Databases":     "数据库",
		"DevOps":        "DevOps 运维",
		"Frameworks":    "框架",
		"Languages":     "编程语言",
		"Methodologies": "方法论",
	},
}

// traduireCategories remplace le titre de chaque groupe par sa traduction dans la langue du CV.
// Appliqué APRÈS le regroupement : le regroupement se fait sur la clé brute, la traduction
// n'est qu'un affichage.
func traduireCategories(groups []SkillGroup, lang string) []SkillGroup {
	table := categoriesParLangue[lang]
	if table == nil {
		return groups
	}
	for i := range groups {
		if t, ok := table[groups[i].Category]; ok {
			groups[i].Category = t
		}
	}
	return groups
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

// renderCVHTML génère le HTML du CV depuis template.
// stealthText : texte brut à afficher dans le strip ATS (vide = strip omis).
// Transmis au template via {{.StealthText}} — rendu en couleur fond, invisible visuellement
// mais correctement encodé dans le stream PDF (pas d'opacity qui corrompt le ToUnicode).
func (s *PDFService) renderCVHTML(cv *AdaptiveCVResponse, lang, stealthText string) (string, error) {
	return s.renderCVHTMLPlafond(cv, lang, stealthText, compactsMax)
}

// GenerateTailoredPDFEnPages : GenerateTailoredPDF borné à `pagesMax` pages.
//
// QUOI : rend le CV ; s'il déborde, retire une rangée de la grille compacte et recommence
// (cf. ajusterAuxPages). Un CV qui tient du premier coup est rendu à l'identique de
// GenerateTailoredPDF.
// POURQUOI : la page 2 de l'export standard est pleine à ~99 %. Le CV taillé y ajoute un résumé
// et des accroches rédigés par le modèle, de longueur variable : mesuré en prod le 11/11/2026,
// le pied de page basculait seul sur une page 3 vide. Aucune borne sur la longueur d'un texte
// rédigé ne garantit la pagination — seule la mesure du rendu le fait.
func (s *PDFService) GenerateTailoredPDFEnPages(cv *AdaptiveCVResponse, lang, stealthText string, pagesMax int) ([]byte, error) {
	return ajusterAuxPages(pagesMax, func(plafond int) ([]byte, error) {
		html, err := s.renderCVHTMLPlafond(cv, lang, stealthText, plafond)
		if err != nil {
			return nil, fmt.Errorf("failed to render HTML: %w", err)
		}
		return htmlToPDF(html)
	})
}

// renderCVHTMLPlafond : renderCVHTML avec le plafond de la grille compacte fixé par l'appelant
// (compactsMax pour l'export standard ; moins quand le CV taillé doit regagner de la place).
func (s *PDFService) renderCVHTMLPlafond(cv *AdaptiveCVResponse, lang, stealthText string, plafondCompacts int) (string, error) {
	var buf strings.Builder

	// Répartition des projets : cartes détaillées, grille compacte, et écartés du PDF.
	// Le CV ne déroule plus le catalogue entier (voir cv_projects_selection.go). Les mots-clés
	// des projets écartés ne disparaissent pas complètement : leurs technologies orphelines
	// rejoignent le strip ATS, dans la limite de son budget.
	topProjects, otherProjects, ecartes := selectionnerProjetsPlafond(cv.Projects, plafondCompacts)
	stealthText = enrichirStripAvecEcartes(stealthText, ecartes)

	// Préparer les données avec les traductions + pré-processing
	data := struct {
		*AdaptiveCVResponse
		Lang          string
		Labels        map[string]string
		TopProjects   []ScoredProjectResponse
		OtherProjects []ScoredProjectResponse // projets 7+ en format compact
		SkillGroups   []SkillGroup
		ProfileImg    template.URL // data URI base64 de la photo (vide si absent)
		StealthText   string       // termes ATS injectés en couleur-fond (invisible)
		Pitch         string       // pitch du thème dans la langue du CV (affiché si pas de Summary)
		RoleTitle     string       // titre du bandeau dans la langue du CV (Theme.NameFor)
	}{
		Pitch:              cv.Theme.PitchFor(lang),
		AdaptiveCVResponse: cv,
		Lang:               lang,
		Labels:             getLabels(lang),
		TopProjects:        topProjects,
		OtherProjects:      otherProjects,
		SkillGroups:        traduireCategories(groupSkillsByCategory(cv.Skills), lang),
		RoleTitle:          cv.Theme.NameFor(lang),
		ProfileImg:         s.profileImg,
		StealthText:        stealthText,
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
			"cv":                "CV",
			"experiences":       "Expériences Professionnelles",
			"skills":            "Compétences",
			"projects":          "Projets",
			"present":           "Présent",
			"years":             "ans",
			"yr":                "a",
			"generated_on":      "Généré le",
			"theme":             "Thème",
			"other_experiences": "Autres expériences",
			"other_projects":    "Autres projets",
			"languages":         "Langues",
			"soft_skills":       "Soft Skills",
			"header_meta":       "Remote · Epitech Nantes 2017",
			"location":          "France · Remote",
			"lang_fr":           "Français",
			"lang_en":           "English",
			"lang_ja":           "日本語",
			"lang_zh":           "中文",
			"lvl_native":        "Natif",
			"lvl_bilingual":     "Bilingue",
			"lvl_basic":         "Basique",
			"soft_organisation": "Organisation",
			"soft_automation":   "Automatisation",
			"soft_ai":           "IA Avancée / LLM",
			"soft_rpg":          "JDR & Worldbuilding",
			"soft_learning":     "Tech Learning",
			"education":         "Formation",
			"degree":            "Diplôme d'Ingénieur",
			"degree_detail":     "Expert en Technologies de l'Information",
			// ats_* : ancres CACHÉES (.ats-anchor) des titres de section, lues par les ATS.
			// En anglais sur un CV fr — forme que les ATS occidentaux indexent (inchangé).
			"ats_experiences": "Professional Experience. Work History.",
			"ats_education":   "Education. Academic Background.",
		},
		"en": {
			"cv":                "Resume",
			"experiences":       "Professional Experience",
			"skills":            "Skills",
			"projects":          "Projects",
			"present":           "Present",
			"years":             "years",
			"yr":                "y",
			"generated_on":      "Generated on",
			"theme":             "Theme",
			"other_experiences": "Other experience",
			"other_projects":    "Other projects",
			"languages":         "Languages",
			"soft_skills":       "Soft Skills",
			"header_meta":       "Remote · Epitech Nantes 2017",
			"location":          "France · Remote",
			"lang_fr":           "French",
			"lang_en":           "English",
			"lang_ja":           "Japanese",
			"lang_zh":           "Chinese",
			"lvl_native":        "Native",
			"lvl_bilingual":     "Fluent",
			"lvl_basic":         "Basic",
			"soft_organisation": "Organization",
			"soft_automation":   "Automation",
			"soft_ai":           "Advanced AI / LLM",
			"soft_rpg":          "TTRPG & Worldbuilding",
			"soft_learning":     "Tech Learning",
			"education":         "Education",
			"degree":            "Engineering Degree",
			"degree_detail":     "Information Technology Expert",
			"ats_experiences":   "Professional Experience. Work History.",
			"ats_education":     "Education. Academic Background.",
		},
		// zh : PDF entièrement chinois (export standard comme /cv/generate). Rendu par les polices
		// WenQuanYi Micro Hei (TrueType, fonts-wqy-microhei), placée avant Noto CJK dans le
		// font-family : Noto CJK est en CFF, que Chromium convertit en Type3 → mots chinois
		// non extractibles par un ATS. Cf. TestPoliceCJK_TrueTypeAvantNoto.
		"zh": {
			"cv":                "简历",
			"experiences":       "工作经历",
			"skills":            "专业技能",
			"projects":          "项目",
			"present":           "至今",
			"years":             "年",
			"yr":                "年",
			"generated_on":      "生成于",
			"theme":             "方向",
			"other_experiences": "其他经历",
			"other_projects":    "其他项目",
			"languages":         "语言能力",
			"soft_skills":       "软技能",
			"header_meta":       "远程 · Epitech 南特 2017",
			"location":          "法国 · 远程",
			"lang_fr":           "法语",
			"lang_en":           "英语",
			"lang_ja":           "日语",
			"lang_zh":           "中文",
			"lvl_native":        "母语",
			"lvl_bilingual":     "流利",
			"lvl_basic":         "基础",
			"soft_organisation": "条理性",
			"soft_automation":   "自动化",
			"soft_ai":           "高级人工智能 / 大语言模型",
			"soft_rpg":          "桌游角色扮演与世界观构建",
			"soft_learning":     "技术学习",
			"education":         "教育背景",
			"degree":            "工程师学位",
			"degree_detail":     "信息技术专家",
			// Synonymes que les ATS chinois cherchent (工作经验 / 职业经历 / 学历), pas une redite
			// du titre visible : l'ancre doit APPORTER des mots-clés.
			"ats_experiences": "工作经验. 职业经历.",
			"ats_education":   "学历. 教育经历.",
		},
	}

	if l, ok := labels[lang]; ok {
		return l
	}
	// Langue sans gabarit (de/it, inconnue) → anglais, jamais le français (cf. GetDefaultLanguage).
	return labels["en"]
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
    <title>` + labels["cv"] + ` - ` + cv.Theme.NameFor(lang) + `</title>
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
    <h1>` + labels["cv"] + ` - ` + cv.Theme.NameFor(lang) + `</h1>
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
