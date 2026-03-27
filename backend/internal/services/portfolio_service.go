package services

// PortfolioService fournit les données du portfolio d'Alexi (projets, skills, expérience).
// Données stub hardcodées — pas de DB.

// StatItem représente une métrique affichable sur une fiche projet
type StatItem struct {
	Label string
	Value string
}

// PortfolioEntry représente un projet du portfolio
type PortfolioEntry struct {
	Name        string
	Title       string
	Category    string
	ShortDesc   string
	KeyFeatures []string
	TechStack   []string
	Stats       []StatItem
	SkillsTags  []string
}

// ExperienceData contient la bio et les expériences professionnelles
type ExperienceData struct {
	Bio        string
	Headline   string
	TJM        string
	Dispo      string
	Experience []ExperienceItem
}

// ExperienceItem représente une expérience professionnelle
type ExperienceItem struct {
	Company  string
	Role     string
	Period   string
	Summary  string
}

// SkillCategory groupe des skills par catégorie
type SkillCategory struct {
	Name   string
	Skills []string
}

// PortfolioService fournit les données du portfolio
type PortfolioService struct {
	projects map[string]PortfolioEntry
	skills   []SkillCategory
	exp      ExperienceData
}

// NewPortfolioService crée une instance avec les données stub
func NewPortfolioService() *PortfolioService {
	svc := &PortfolioService{}
	svc.projects = svc.buildProjects()
	svc.skills = svc.buildSkills()
	svc.exp = svc.buildExperience()
	return svc
}

// GetProject retourne les détails d'un projet par son nom (insensible à la casse)
func (s *PortfolioService) GetProject(name string) (PortfolioEntry, bool) {
	// Cherche avec le nom exact d'abord
	if p, ok := s.projects[name]; ok {
		return p, true
	}
	// Fallback : cherche dans les titres/noms de façon insensible
	lower := toLower(name)
	for k, p := range s.projects {
		if toLower(k) == lower || toLower(p.Title) == lower {
			return p, true
		}
	}
	return PortfolioEntry{}, false
}

// ListProjects retourne tous les projets sous forme de slice
func (s *PortfolioService) ListProjects() []PortfolioEntry {
	list := make([]PortfolioEntry, 0, len(s.projects))
	// Ordre déterministe
	order := []string{"maicivy", "aria", "cogesco", "liveconf", "freelance-dashboard"}
	for _, k := range order {
		if p, ok := s.projects[k]; ok {
			list = append(list, p)
		}
	}
	return list
}

// ListSkills retourne les skills groupés par catégorie
func (s *PortfolioService) ListSkills() []SkillCategory {
	return s.skills
}

// GetExperience retourne la bio et l'expérience d'Alexi
func (s *PortfolioService) GetExperience() ExperienceData {
	return s.exp
}

// SearchProjects retourne les projets dont le titre/desc/stack contient le terme
func (s *PortfolioService) SearchProjects(query string) []PortfolioEntry {
	q := toLower(query)
	var results []PortfolioEntry
	for _, p := range s.projects {
		if strContains(toLower(p.Title), q) ||
			strContains(toLower(p.ShortDesc), q) ||
			strContains(toLower(p.Category), q) ||
			strContainsAny(p.TechStack, q) {
			results = append(results, p)
		}
	}
	return results
}

// --- Données stub ---

func (s *PortfolioService) buildProjects() map[string]PortfolioEntry {
	return map[string]PortfolioEntry{
		"maicivy": {
			Name:      "maicivy",
			Title:     "maicivy — CV IA interactif",
			Category:  "Full-Stack / AI",
			ShortDesc: "Portfolio interactif avec génération de lettres de motivation, CV dynamique adaptatif, analytics temps réel et assistant IA conversationnel.",
			KeyFeatures: []string{
				"Génération de lettres de motivation personnalisées par IA (Claude Opus)",
				"CV adaptatif selon le profil visiteur (backend / frontend / DevOps / AI)",
				"Analytics temps réel WebSocket avec dashboard public",
				"Stealth ATS : injection de mots-clés invisibles dans le PDF",
				"Assistant chat avec tool_use (ce que tu utilises là !)",
			},
			TechStack:  []string{"Go", "Fiber", "Next.js", "TypeScript", "PostgreSQL", "Redis", "Claude API", "Docker"},
			Stats:      []StatItem{{Label: "Lignes de code", Value: "~12k"}, {Label: "Endpoints API", Value: "25+"}, {Label: "Temps de réponse", Value: "< 200ms"}},
			SkillsTags: []string{"Go", "Next.js", "Claude API", "SSE", "PostgreSQL", "Redis", "Docker"},
		},
		"aria": {
			Name:      "aria",
			Title:     "Aria — Agent IA autonome",
			Category:  "AI / Agents",
			ShortDesc: "Système multi-agent avec mémoire persistante, planification de tâches et exécution d'actions sur des outils externes (web, code, fichiers).",
			KeyFeatures: []string{
				"Boucle agentic avec tool_use (Claude)",
				"Mémoire persistante par session (embeddings + vector store)",
				"Orchestration multi-agents avec délégation",
				"Outils : browser, terminal, filesystem, APIs",
				"Interface web temps réel avec streaming SSE",
			},
			TechStack:  []string{"Go", "TypeScript", "Claude API", "pgvector", "Redis", "Docker"},
			Stats:      []StatItem{{Label: "Tools disponibles", Value: "12"}, {Label: "Agents parallèles", Value: "jusqu'à 8"}, {Label: "Latence tool", Value: "< 50ms"}},
			SkillsTags: []string{"AI Agents", "Claude API", "Tool Use", "Streaming", "pgvector"},
		},
		"cogesco": {
			Name:      "cogesco",
			Title:     "Cogesco — SEO IA",
			Category:  "SaaS / AI",
			ShortDesc: "Plateforme SaaS de génération de contenu SEO par IA : articles, cocons sémantiques, meta tags, optimisation automatique.",
			KeyFeatures: []string{
				"Génération d'articles SEO longs (2000-5000 mots) par IA",
				"Cocons sémantiques : planification de maillage interne automatique",
				"Scoring SEO en temps réel (readability, densité, structure)",
				"Intégration CMS (WordPress, Webflow) via API",
				"Dashboard multi-sites avec analytics SEO",
			},
			TechStack:  []string{"Go", "Next.js", "TypeScript", "PostgreSQL", "Redis", "Claude API", "OpenAI"},
			Stats:      []StatItem{{Label: "Articles générés", Value: "50k+"}, {Label: "Sites clients", Value: "120+"}, {Label: "Uptime", Value: "99.9%"}},
			SkillsTags: []string{"Go", "SaaS", "SEO", "Claude API", "Next.js", "PostgreSQL"},
		},
		"liveconf": {
			Name:      "liveconf",
			Title:     "LiveConf — Conférences live",
			Category:  "Real-Time / Full-Stack",
			ShortDesc: "Plateforme de conférences en ligne avec streaming vidéo, chat temps réel, Q&A interactif et tableau de bord speaker.",
			KeyFeatures: []string{
				"Streaming vidéo WebRTC P2P + fallback TURN",
				"Chat temps réel avec WebSocket (10k+ connexions simultanées)",
				"Q&A avec upvote et modération temps réel",
				"Tableau de bord speaker : slides, timer, questions en attente",
				"Replay automatique avec découpe par segment",
			},
			TechStack:  []string{"Go", "React", "WebRTC", "WebSocket", "Redis", "PostgreSQL", "FFmpeg"},
			Stats:      []StatItem{{Label: "Connexions max", Value: "10k+"}, {Label: "Latence vidéo", Value: "< 300ms"}, {Label: "Événements live", Value: "500+"}},
			SkillsTags: []string{"WebRTC", "WebSocket", "Go", "React", "Redis", "Streaming"},
		},
		"freelance-dashboard": {
			Name:      "freelance-dashboard",
			Title:     "Freelance Dashboard",
			Category:  "Tooling / Productivity",
			ShortDesc: "Dashboard de gestion freelance : suivi des missions, facturation, TJM tracker, pipeline commercial et reporting mensuel.",
			KeyFeatures: []string{
				"Pipeline commercial : prospects → devis → mission → facture",
				"TJM tracker avec graphes de revenus mensuels",
				"Génération de devis et factures PDF",
				"Intégration Malt/LinkedIn pour synchroniser les leads",
				"Alertes relances automatiques",
			},
			TechStack:  []string{"Next.js", "TypeScript", "PostgreSQL", "Prisma", "Tailwind"},
			Stats:      []StatItem{{Label: "Missions trackées", Value: "50+"}, {Label: "CA suivi", Value: "> 200k€"}, {Label: "Temps gagné/mois", Value: "~4h"}},
			SkillsTags: []string{"Next.js", "TypeScript", "PostgreSQL", "Prisma", "Tailwind"},
		},
	}
}

func (s *PortfolioService) buildSkills() []SkillCategory {
	return []SkillCategory{
		{
			Name:   "Languages",
			Skills: []string{"Go", "TypeScript", "JavaScript", "Python", "C++", "SQL"},
		},
		{
			Name:   "Backend",
			Skills: []string{"Fiber", "Gin", "Node.js", "REST", "gRPC", "WebSocket", "SSE"},
		},
		{
			Name:   "Frontend",
			Skills: []string{"Next.js", "React", "Tailwind CSS", "Framer Motion", "shadcn/ui"},
		},
		{
			Name:   "Data & DB",
			Skills: []string{"PostgreSQL", "Redis", "SQLite", "pgvector", "GORM", "Prisma"},
		},
		{
			Name:   "AI / ML",
			Skills: []string{"Claude API", "OpenAI API", "Tool Use", "RAG", "Embeddings", "Prompt Engineering"},
		},
		{
			Name:   "Infra / DevOps",
			Skills: []string{"Docker", "GitHub Actions", "Nginx", "Linux", "SSH", "Monitoring"},
		},
	}
}

func (s *PortfolioService) buildExperience() ExperienceData {
	return ExperienceData{
		Bio:      "Développeur freelance full-stack avec une spécialisation croissante en IA et systèmes agentiques. Je construis des produits complets — de l'API au frontend — avec une obsession pour la performance et l'expérience utilisateur.",
		Headline: "Développeur Full-Stack & IA · Freelance",
		TJM:      "600-800€/jour",
		Dispo:    "Disponible",
		Experience: []ExperienceItem{
			{
				Company: "Freelance",
				Role:    "Développeur Full-Stack & IA",
				Period:  "2022 — présent",
				Summary: "Missions variées : SaaS B2B, outils IA, plateformes temps réel. Stack principale : Go + Next.js + Claude API.",
			},
			{
				Company: "Startup SaaS",
				Role:    "Lead Developer",
				Period:  "2020 — 2022",
				Summary: "Lead technique d'une startup SaaS B2B (10 → 120 clients). Architecture micro-services, CI/CD, recrutement junior.",
			},
			{
				Company: "Agence Web",
				Role:    "Développeur Full-Stack",
				Period:  "2018 — 2020",
				Summary: "Développement de projets clients variés (e-commerce, marketplaces, outils internes). Stack React + Node.js.",
			},
		},
	}
}

// --- helpers ---

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

// strContains vérifie si s contient sub (strings, distinct de la fonction contains de cv_scoring.go qui opère sur des slices)
func strContains(s, sub string) bool {
	if sub == "" {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// strContainsAny vérifie si un élément de la liste contient sub
func strContainsAny(list []string, sub string) bool {
	for _, item := range list {
		if strContains(toLower(item), sub) {
			return true
		}
	}
	return false
}
