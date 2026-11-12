package config

// CVTheme représente un thème de CV avec ses tags prioritaires
type CVTheme struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Icon        string             `json:"icon"`       // Emoji or icon identifier
	TagWeights  map[string]float64 `json:"tagWeights"` // tag → poids (0.0-1.0)

	// Pitch : résumé affiché en tête du CV PDF pour ce thème, par langue (fr/en). Vide = pas de
	// bloc. Le résumé LLM d'un CV taillé sur une offre reste prioritaire (cf. cv_base.html).
	// POURQUOI par langue et pas une seule chaîne : le CV est bilingue, et un pitch français sur
	// un CV anglais casse la lecture dès la première ligne.
	Pitch map[string]string `json:"pitch,omitempty"`

	// NameI18n : titre du bandeau du CV PDF dans une langue où l'intitulé anglais ne se lit pas
	// (zh). Absent = Name. fr/en gardent volontairement l'intitulé anglais (usage du marché et
	// terme cherché par les ATS). Le sélecteur de thème du site, lui, affiche toujours Name.
	NameI18n map[string]string `json:"-"`
}

// NameFor rend le titre du CV dans la langue demandée, Name à défaut. Un thème dynamique
// (/cv/generate) n'a pas de NameI18n : il garde le titre que le modèle a écrit dans la langue
// du CV.
func (t CVTheme) NameFor(lang string) string {
	if n := t.NameI18n[lang]; n != "" {
		return n
	}
	return t.Name
}

// PitchFor rend le pitch dans la langue demandée, avec repli sur l'ANGLAIS (même règle que le
// contenu maiProFiles : natif si disponible, sinon anglais, jamais le français par défaut).
func (t CVTheme) PitchFor(lang string) string {
	if p := t.Pitch[lang]; p != "" {
		return p
	}
	return t.Pitch["en"]
}

// GetAvailableThemes retourne tous les thèmes configurés
func GetAvailableThemes() map[string]CVTheme {
	return map[string]CVTheme{
		// Positionnement décidé le 10/10/2026, titre renommé le 09/11/2026 en « AI Orchestration
		// Engineer — Technomancer » (choix d'Alexis). Le titre
		// reprend le terme que clients et ATS cherchent ; « Technomancer » est la marque, accolée,
		// jamais seule. C++, Flutter et full-stack sont présentés comme les PREUVES de la
		// spécialité (« je construis tout ce qui entoure les agents »), pas comme des métiers à
		// part. Le nom et la Description alimentent aussi le prompt du scoring LLM des projets
		// (llm_scoring.go) : ils décident donc quels projets passent en tête.
		"ai": {
			ID:       "ai",
			Name:     "AI Orchestration Engineer — Technomancer",
			NameI18n: map[string]string{"zh": "AI 编排工程师 — Technomancer"},
			// ⚠️ Affichée telle quelle dans le sélecteur de thème du site (CVThemeSelector) : rester
			// court et lisible par un visiteur, pas une consigne au modèle.
			Description: "Systèmes multi-agents IA en production : orchestration, outils MCP, construits de bout en bout (C++, Flutter, Go)",
			Icon:        "\U0001F916",
			Pitch: map[string]string{
				"fr": "Systèmes multi-agents en production, construits de bout en bout : moteur, application, backend, agents. " +
					"Je conçois et mets en production des systèmes d'agents IA pour les startups et scale-ups. " +
					"Comme je construis aussi ce qui les entoure (moteur C++, IDE Flutter, backend Go/Postgres), ces agents dépassent le stade de la démo. " +
					"En mission pour une start-up SaaS B2B, près de 80 % des merge requests fusionnées sont les miennes, livrées par une équipe d'agents que j'orchestre.",
				"en": "Multi-agent systems in production, built end-to-end: engine, app, backend, agents. " +
					"I design and ship AI agent systems for startups and scale-ups. " +
					"Because I also build what surrounds them (a C++ engine, a Flutter IDE, a Go/Postgres backend), these agents get past the demo stage. " +
					"On my current engagement with a B2B SaaS start-up, nearly 80% of merged merge requests are mine, delivered by a team of agents I orchestrate.",
				// zh : sans lui, PitchFor repliait sur l'anglais et le CV chinois s'ouvrait sur un
				// paragraphe anglais (mesuré sur le PDF de prod le 28/10).
				// ⚠️ Mission : client jamais nommé, et jamais « en production chez des clients ».
				"zh": "端到端构建的生产级多智能体系统：引擎、应用、后端、智能体。" +
					"我为初创公司和成长型企业设计并交付 AI 智能体系统。" +
					"由于我同时构建它们周围的一切（C++ 引擎、Flutter IDE、Go/Postgres 后端），这些智能体能够走出演示阶段。" +
					"目前在为一家 B2B SaaS 初创公司提供服务，已合并的合并请求中近 80% 出自我手，由我编排的智能体团队交付。",
			},
			TagWeights: map[string]float64{
				"ai":          1.0,
				"llm":         1.0,
				"claude":      1.0,
				"claude api":  1.0,
				"openai":      0.9,
				"mcp":         0.9,
				"mcp sdk":     0.9,
				"agents":      0.9,
				"rag":         0.8,
				"python":      0.7,
				"go":          0.7,
				"flutter":     0.6,
				"c++17":       0.6,
				"api":         0.6,
				"automation":  0.6,
				"ml":          0.5,
				"fine-tuning": 0.5,
			},
		},
		"backend": {
			ID:          "backend",
			Name:        "Backend Developer",
			NameI18n:    map[string]string{"zh": "后端开发工程师"},
			Description: "Focus sur développement backend, APIs, bases de données",
			Icon:        "\U0001F5A5",
			TagWeights: map[string]float64{
				"go":            1.0,
				"api":           1.0,
				"backend":       1.0,
				"postgresql":    0.9,
				"redis":         0.9,
				"docker":        0.8,
				"microservices": 0.8,
				"restful":       0.8,
				"grpc":          0.7,
				"kubernetes":    0.7,
				"nodejs":        0.6,
				"python":        0.5,
			},
		},
		"cpp": {
			ID:          "cpp",
			Name:        "C++ Developer",
			NameI18n:    map[string]string{"zh": "C++ 开发工程师"},
			Description: "Focus sur développement C++, systèmes bas niveau",
			Icon:        "\U0001F527",
			TagWeights: map[string]float64{
				"c++":          1.0,
				"cpp":          1.0,
				"c":            0.9,
				"systems":      0.9,
				"embedded":     0.8,
				"performance":  0.8,
				"optimization": 0.8,
				"low-level":    0.8,
				"memory":       0.7,
				"algorithms":   0.7,
				"qt":           0.6,
				"boost":        0.6,
			},
		},
		"artistique": {
			ID:          "artistique",
			Name:        "Creative & Artistic",
			NameI18n:    map[string]string{"zh": "创意与艺术"},
			Description: "Focus sur projets créatifs, design, visualisation",
			Icon:        "\U0001F3A8",
			TagWeights: map[string]float64{
				"design":      1.0,
				"art":         1.0,
				"creative":    1.0,
				"ui":          0.9,
				"ux":          0.9,
				"3d":          0.9,
				"graphics":    0.9,
				"animation":   0.8,
				"threejs":     0.8,
				"webgl":       0.8,
				"photoshop":   0.7,
				"illustrator": 0.7,
				"blender":     0.7,
			},
		},
		"fullstack": {
			ID:          "fullstack",
			// Titre du bandeau = celui du thème "ai" (choix d'Alexis le 09/11/2026, dossier pf_71da886381ac) :
			// intitulé du marché (« AI Orchestration Engineer ») puis la marque. Les thèmes backend, cpp,
			// devops et artistique gardent leur intitulé métier, sans la marque.
			Name:        "AI Orchestration Engineer — Technomancer",
			NameI18n:    map[string]string{"zh": "AI 编排工程师 — Technomancer"},
			Description: "Focus sur développement full-stack, frontend + backend",
			Icon:        "\U0001F4BB",
			TagWeights: map[string]float64{
				"fullstack":  1.0,
				"frontend":   0.9,
				"backend":    0.9,
				"react":      0.9,
				"nextjs":     0.9,
				"typescript": 0.9,
				"api":        0.8,
				"nodejs":     0.8,
				"go":         0.8,
				"postgresql": 0.7,
				"tailwind":   0.7,
				"docker":     0.6,
			},
		},
		"devops": {
			ID:          "devops",
			Name:        "DevOps Engineer",
			NameI18n:    map[string]string{"zh": "DevOps 工程师"},
			Description: "Focus sur infrastructure, CI/CD, monitoring",
			Icon:        "\U0001F433",
			TagWeights: map[string]float64{
				"devops":     1.0,
				"docker":     1.0,
				"kubernetes": 1.0,
				"ci/cd":      1.0,
				"terraform":  0.9,
				"ansible":    0.9,
				"aws":        0.9,
				"monitoring": 0.8,
				"prometheus": 0.8,
				"grafana":    0.8,
				"nginx":      0.7,
				"jenkins":    0.7,
				"gitops":     0.7,
			},
		},
	}
}

// GetTheme retourne un thème spécifique ou nil
func GetTheme(themeID string) *CVTheme {
	themes := GetAvailableThemes()
	if theme, exists := themes[themeID]; exists {
		return &theme
	}
	return nil
}
