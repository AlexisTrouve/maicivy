package services

import (
	"os"
	"strings"
	"testing"
	"unicode"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/stretchr/testify/require"
)

// Tests du PDF de CV en chinois (export standard ET /cv/generate).
// POURQUOI : avant, un CV zh sortait en anglais (TailoredLanguage zh→en) ou avec des libellés
// français (getLabels retombait sur le FR, gabarit truffé de chaînes FR en dur).

// contientCJK : vrai si le texte porte au moins un idéogramme (Han).
func contientCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// TestLabelsZh_CompletsEtChinois : chaque libellé existant en fr existe en zh, en chinois.
func TestLabelsZh_CompletsEtChinois(t *testing.T) {
	fr, en, zh := getLabels("fr"), getLabels("en"), getLabels("zh")
	require.Len(t, zh, len(fr), "zh doit couvrir exactement les mêmes clés que fr")
	for k := range fr {
		v, ok := zh[k]
		require.True(t, ok, "libellé zh manquant : %s", k)
		require.True(t, contientCJK(v), "libellé zh %s = %q n'est pas en chinois", k, v)
		if k == "lang_zh" {
			continue // « 中文 » : le nom du chinois s'écrit en chinois dans le gabarit fr aussi
		}
		require.NotEqual(t, fr[k], v, "libellé zh %s identique au français", k)
		require.NotEqual(t, en[k], v, "libellé zh %s identique à l'anglais", k)
	}
}

// TestLabels_LangueInconnueEnAnglais : de/it/inconnue → anglais (règle GetDefaultLanguage),
// jamais le français.
func TestLabels_LangueInconnueEnAnglais(t *testing.T) {
	for _, l := range []string{"de", "it", ""} {
		require.Equal(t, getLabels("en"), getLabels(l), "lang %q", l)
	}
}

// TestRenderZh_AucunLibelleFrancaisNiAnglais : le HTML rendu par le VRAI gabarit en zh ne
// contient plus aucune des chaînes visibles françaises/anglaises qui étaient en dur.
func TestRenderZh_AucunLibelleFrancaisNiAnglais(t *testing.T) {
	s := newPDFServiceFrom("../../templates/cv")
	require.True(t, s.hasTemplates)
	cv := &AdaptiveCVResponse{
		Theme:  config.CVTheme{ID: "fullstack", Name: "全栈开发"},
		Skills: []ScoredSkillResponse{{Skill: models.Skill{Name: "Go", Category: "后端", Level: models.SkillLevelExpert, YearsExperience: 5}}},
	}
	html, err := s.renderCVHTML(cv, "zh", "")
	require.NoError(t, err)

	for _, k := range []string{"skills", "languages", "soft_skills", "education", "lvl_native", "degree"} {
		require.Contains(t, html, getLabels("zh")[k], "libellé zh %s absent du rendu", k)
	}
	for _, interdit := range []string{
		"Natif", "Bilingue", "Basique", "Diplôme", "Expert en Technologies", "Organisation",
		"Automatisation", "France · Remote", "Remote · Epitech", "Compétences", "Langues",
		">Skills<", ">Languages<", ">Education<", "Native", "Fluent",
	} {
		require.NotContains(t, html, interdit, "chaîne non chinoise dans le PDF zh")
	}
	require.Contains(t, html, `lang="zh"`)
}

// TestStealthZh_EnChinois : le texte ATS zh est en chinois (et garde les termes techniques
// anglais que les ATS cherchent tels quels).
func TestStealthZh_EnChinois(t *testing.T) {
	s := buildStealthText([]string{"Go"}, "zh")
	require.True(t, contientCJK(s))
	require.NotContains(t, s, "Compétences")
	require.Contains(t, s, "Go")
	// Termes GÉNÉRIQUES en chinois (2026-10-16) : un ATS chinois cherche « 软件工程 », pas
	// « Software Engineering ». Les NOMS de technos (Go…), eux, restent tels quels.
	require.Contains(t, s, "软件工程")
	require.Contains(t, s, "代码审查")
	require.NotContains(t, s, "Software Engineering")
	require.NotContains(t, s, "Code Review")
}

// TestAtsAnchorZh_EnChinois : les ancres ATS cachées des titres de section suivent la langue
// du CV. Mesuré le 2026-10-16 sur le PDF zh de prod : « PROFESSIONAL EXPERIENCE. WORK
// HISTORY » et « EDUCATION. ACADEMIC BACKGROUND » en anglais, aucun mot-clé chinois.
func TestAtsAnchorZh_EnChinois(t *testing.T) {
	s := newPDFServiceFrom("../../templates/cv")
	require.True(t, s.hasTemplates)
	cv := &AdaptiveCVResponse{
		Theme:       config.CVTheme{ID: "fullstack", Name: "全栈开发"},
		Experiences: []ScoredExperienceResponse{{Experience: models.Experience{Title: "开发者", Company: "X"}}},
	}
	html, err := s.renderCVHTML(cv, "zh", "")
	require.NoError(t, err)
	require.Contains(t, html, `<span class="ats-anchor">`+getLabels("zh")["ats_experiences"]+`</span>`)
	require.Contains(t, html, `<span class="ats-anchor">`+getLabels("zh")["ats_education"]+`</span>`)
	require.True(t, contientCJK(getLabels("zh")["ats_experiences"]))
	for _, interdit := range []string{"Work History", "Professional Experience", "Academic Background"} {
		require.NotContains(t, html, interdit)
	}
	// fr / en : ancres inchangées (anglais, la forme que les ATS occidentaux indexent).
	for _, l := range []string{"fr", "en"} {
		require.Equal(t, "Professional Experience. Work History.", getLabels(l)["ats_experiences"])
		require.Equal(t, "Education. Academic Background.", getLabels(l)["ats_education"])
	}
}

// TestStripATS_ZhDansLeBudget : même invariant que le FR — le strip zh enrichi tient dans le
// budget (mesuré en octets ; cf. budgetStripATS pour la mesure de rendu CJK).
func TestStripATS_ZhDansLeBudget(t *testing.T) {
	ecartes := make([]ScoredProjectResponse, 0, 60)
	for i := 0; i < 60; i++ {
		ecartes = append(ecartes, ScoredProjectResponse{Project: models.Project{
			Technologies: []string{"techno-orpheline-" + strings.Repeat("x", i%7), "go"},
		}})
	}
	base := buildStealthText([]string{"Go"}, "zh")
	strip := enrichirStripAvecEcartes(base, ecartes)
	require.LessOrEqual(t, len(strip), budgetStripATS)
	require.Contains(t, strip, base)
}

// TestPreuveZh_EnChinois : la ligne de preuve des projets est en chinois sur un CV zh.
func TestPreuveZh_EnChinois(t *testing.T) {
	got := ligneDePreuve("production", MPFStats{LOC: 93586, Tests: 1415}, "zh")
	require.Equal(t, "已投产 · 94k 行代码 · 1.4k 项测试", got)
}

// TestPromptGeneration_ZhExigeLeChinois : le prompt du CV tailoré zh impose une sortie en
// chinois simplifié (summary, titre, catchphrases) ; fr/en n'ont pas cette consigne.
func TestPromptGeneration_ZhExigeLeChinois(t *testing.T) {
	s := &CVGenerationService{}
	require.Contains(t, s.buildGenerationPrompt("offre", nil, nil, nil, "zh"), "简体中文")
	require.NotContains(t, s.buildGenerationPrompt("offre", nil, nil, nil, "fr"), "简体中文")
}

// TestPoliceCJK_TrueTypeAvantNoto : le chinois du PDF est rendu par une police TrueType
// (WenQuanYi Micro Hei), PAS par Noto Sans CJK.
// POURQUOI : Noto CJK est une police CFF ; le moteur PDF de Chromium (Skia) ne sait pas
// l'embarquer et la convertit en Type3. Mesuré le 2026-10-16 dans l'image de prod : 267
// polices Type3 dans le CV zh, et pypdf ne retrouvait que 2 mots chinois sur 5 (les autres
// extracteurs les coupent caractère par caractère) — un ATS chinois ne lisait pas le CV.
// Avec WenQuanYi Micro Hei : police Type0 embarquée, 5/5 sur les trois extracteurs, PDF
// 3× plus léger. Les polices latines (DejaVu) restent en tête : fr/en inchangés.
func TestPoliceCJK_TrueTypeAvantNoto(t *testing.T) {
	gabarit, err := os.ReadFile("../../templates/cv/cv_base.html")
	require.NoError(t, err)
	g := string(gabarit)
	wqy := strings.Index(g, "'WenQuanYi Micro Hei'")
	noto := strings.Index(g, "'Noto Sans CJK SC'")
	require.GreaterOrEqual(t, wqy, 0, "WenQuanYi Micro Hei absente du font-family")
	require.Less(t, wqy, noto, "WenQuanYi Micro Hei doit passer AVANT Noto Sans CJK")
	require.Less(t, strings.Index(g, "'DejaVu Sans'"), wqy, "le latin doit rester en DejaVu")

	dockerfile, err := os.ReadFile("../../Dockerfile")
	require.NoError(t, err)
	require.Contains(t, string(dockerfile), "fonts-wqy-microhei", "la police doit être installée dans l'image")
}

// TestThemeNameZh_TitreChinois : sur un CV zh, le titre du bandeau (« Full-Stack Developer »)
// sort en chinois. fr/en (et de/it, repliés) gardent le titre anglais : c'est l'usage en
// France et l'intitulé que les ATS occidentaux cherchent. Un thème dynamique (/cv/generate)
// garde le titre produit par le modèle, déjà dans la langue du CV.
func TestThemeNameZh_TitreChinois(t *testing.T) {
	for id, th := range config.GetAvailableThemes() {
		require.True(t, contientCJK(th.NameFor("zh")), "thème %s : titre zh %q", id, th.NameFor("zh"))
		for _, l := range []string{"fr", "en", "de", ""} {
			require.Equal(t, th.Name, th.NameFor(l), "thème %s lang %q", id, l)
		}
	}
	dyn := config.CVTheme{ID: "dynamic", Name: "高级 Go 工程师"}
	require.Equal(t, dyn.Name, dyn.NameFor("zh"))

	s := newPDFServiceFrom("../../templates/cv")
	th := config.GetAvailableThemes()["fullstack"]
	html, err := s.renderCVHTML(&AdaptiveCVResponse{Theme: th}, "zh", "")
	require.NoError(t, err)
	require.Contains(t, html, `<div class="header-role">`+th.NameFor("zh")+`</div>`)
	require.NotContains(t, html, "Full-Stack Developer")
}

// TestCategoriesZh_EnChinois : les titres de catégories de compétences (clés anglaises de
// maiProFiles : « Languages », « Backend »…) sont traduits sur un CV zh ; une catégorie
// inconnue passe telle quelle ; fr/en inchangés.
func TestCategoriesZh_EnChinois(t *testing.T) {
	cats := []string{"AI", "Backend", "Creative", "Databases", "DevOps", "Frameworks", "Languages", "Methodologies"}
	skills := make([]ScoredSkillResponse, 0, len(cats)+1)
	for _, c := range append(cats, "Inconnue") {
		skills = append(skills, ScoredSkillResponse{Skill: models.Skill{Name: "X" + c, Category: c}})
	}
	s := newPDFServiceFrom("../../templates/cv")
	html, err := s.renderCVHTML(&AdaptiveCVResponse{Theme: config.CVTheme{ID: "t", Name: "t"}, Skills: skills}, "zh", "")
	require.NoError(t, err)
	for _, c := range cats {
		require.NotContains(t, html, `<div class="skill-cat">`+c+`</div>`, "catégorie %s non traduite", c)
	}
	require.Contains(t, html, `<div class="skill-cat">编程语言</div>`)
	require.Contains(t, html, `<div class="skill-cat">Inconnue</div>`)

	htmlEn, err := s.renderCVHTML(&AdaptiveCVResponse{Theme: config.CVTheme{ID: "t", Name: "t"}, Skills: skills}, "en", "")
	require.NoError(t, err)
	require.Contains(t, htmlEn, `<div class="skill-cat">Languages</div>`)
}

// TestGrasZh_700PourLesTitresEn600 : WenQuanYi Micro Hei n'a pas de graisse grasse ; Chromium
// la SIMULE à 700 mais pas à 600 (mesuré le 2026-10-16 dans l'image de prod) — les noms de
// compétences chinois (600) sortaient en maigre à côté des noms latins en gras. Sur un CV zh,
// tout ce qui est en 600 passe en 700. Côté latin rien ne bouge : DejaVu n'a qu'une graisse
// Bold, que 600 sélectionnait déjà. (Le contour de glyphe est écarté : il fait
// extraire certains mots en double.)
func TestGrasZh_700PourLesTitresEn600(t *testing.T) {
	g, err := os.ReadFile("../../templates/cv/cv_base.html")
	require.NoError(t, err)
	css := string(g)
	debut := strings.Index(css, `html[lang="zh"] .pill`)
	require.GreaterOrEqual(t, debut, 0, "règle de gras zh absente")
	regle := css[debut : debut+strings.Index(css[debut:], "}")]
	// Les titres serif du design sombre (.section-label, .exp-title) et le rôle du bandeau sont
	// en 600 et portent du chinois : même règle que les autres.
	for _, sel := range []string{".pill", ".proj-title", ".mini-proj-title", ".proj-proof", ".mini-proj-proof", ".mini-pill", ".skill-name",
		".section-label", ".exp-title", ".header-role"} {
		require.Contains(t, regle, `html[lang="zh"] `+sel, "sélecteur %s absent", sel)
	}
	require.Contains(t, regle, "font-weight: 700")
	require.NotContains(t, css, "text-stroke")
}
