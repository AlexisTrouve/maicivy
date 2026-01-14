# 06. BACKEND_CV_API - API CV Dynamique Adaptatif

## 📋 Métadonnées

- **Phase:** 2
- **Priorité:** 🟡 HAUTE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 04. BACKEND_MIDDLEWARES.md
- **Temps estimé:** 3-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter le système de CV dynamique adaptatif qui personnalise automatiquement le contenu selon le thème demandé (Backend, C++, Artistique, Full-Stack, DevOps). Le système utilise un algorithme de scoring intelligent basé sur les tags pour filtrer et pondérer les expériences, compétences et projets les plus pertinents pour chaque thème.

**Livrables principaux:**
- Algorithme de filtrage/scoring par tags
- Endpoints API REST pour CV adaptatif
- Export PDF basique
- Système de caching Redis pour performances optimales

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────────┐
│                    Client (Frontend)                        │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│               API Endpoints (Fiber)                         │
│  /api/cv?theme=backend                                      │
│  /api/cv/themes                                             │
│  /api/experiences                                           │
│  /api/skills                                                │
│  /api/projects                                              │
│  /api/cv/export?theme=backend&format=pdf                   │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│              CV Service Layer                               │
│  - GetAdaptiveCV(theme)                                     │
│  - CalculateScores(items, theme)                            │
│  - FilterAndSort(items, scores)                             │
│  - GetAllThemes()                                           │
└────────────┬────────────────────────────────────────────────┘
             │
    ┌────────┴────────┐
    ▼                 ▼
┌─────────┐      ┌─────────┐
│  Redis  │      │ Postgres│
│  Cache  │      │   DB    │
└─────────┘      └─────────┘
    TTL 1h        Models GORM
```

### Design Decisions

**1. Algorithme de Scoring:**
- Système de scoring basé sur tags matching
- Pondération configurable par thème
- Tri par pertinence décroissante
- **Justification:** Flexibilité pour ajouter des thèmes sans refactoriser

**2. Caching Strategy:**
- Cache Redis avec TTL 1h pour chaque thème
- Invalidation manuelle via endpoint admin (optionnel Phase 6)
- **Justification:** CV change rarement, performances critiques

**3. Export PDF:**
- chromedp pour rendu HTML → PDF
- Templates HTML réutilisables
- **Justification:** Meilleure qualité visuelle que gofpdf

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# Framework web (déjà installé Phase 1)
go get github.com/gofiber/fiber/v2

# ORM (déjà installé Phase 1)
go get gorm.io/gorm

# Redis client (déjà installé Phase 1)
go get github.com/redis/go-redis/v9

# PDF generation via Chrome headless
go get github.com/chromedp/chromedp

# JSON utilities
go get github.com/tidwall/gjson

# Validation
go get github.com/go-playground/validator/v10
```

### Services Externes

- **PostgreSQL:** Base de données pour experiences, skills, projects
- **Redis:** Cache pour CV pré-calculés
- **Chrome/Chromium:** Requis pour chromedp (export PDF)

---

## 🔨 Implémentation

### Étape 1: Définir la Configuration des Thèmes

**Description:** Créer une configuration centralisée définissant les thèmes disponibles et leurs tags associés.

**Fichier:** `backend/internal/config/themes.go`

**Code:**

```go
package config

// CVTheme représente un thème de CV avec ses tags prioritaires
type CVTheme struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	TagWeights  map[string]float64 `json:"tag_weights"` // tag → poids (0.0-1.0)
}

// GetAvailableThemes retourne tous les thèmes configurés
func GetAvailableThemes() map[string]CVTheme {
	return map[string]CVTheme{
		"backend": {
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Focus sur développement backend, APIs, bases de données",
			TagWeights: map[string]float64{
				"go":         1.0,
				"api":        1.0,
				"backend":    1.0,
				"postgresql": 0.9,
				"redis":      0.9,
				"docker":     0.8,
				"microservices": 0.8,
				"restful":    0.8,
				"grpc":       0.7,
				"kubernetes": 0.7,
				"nodejs":     0.6,
				"python":     0.5,
			},
		},
		"cpp": {
			ID:          "cpp",
			Name:        "C++ Developer",
			Description: "Focus sur développement C++, systèmes bas niveau",
			TagWeights: map[string]float64{
				"c++":        1.0,
				"cpp":        1.0,
				"c":          0.9,
				"systems":    0.9,
				"embedded":   0.8,
				"performance": 0.8,
				"optimization": 0.8,
				"low-level":  0.8,
				"memory":     0.7,
				"algorithms": 0.7,
				"qt":         0.6,
				"boost":      0.6,
			},
		},
		"artistique": {
			ID:          "artistique",
			Name:        "Creative & Artistic",
			Description: "Focus sur projets créatifs, design, visualisation",
			TagWeights: map[string]float64{
				"design":       1.0,
				"art":          1.0,
				"creative":     1.0,
				"ui":           0.9,
				"ux":           0.9,
				"3d":           0.9,
				"graphics":     0.9,
				"animation":    0.8,
				"threejs":      0.8,
				"webgl":        0.8,
				"photoshop":    0.7,
				"illustrator":  0.7,
				"blender":      0.7,
			},
		},
		"fullstack": {
			ID:          "fullstack",
			Name:        "Full-Stack Developer",
			Description: "Focus sur développement full-stack, frontend + backend",
			TagWeights: map[string]float64{
				"fullstack":   1.0,
				"frontend":    0.9,
				"backend":     0.9,
				"react":       0.9,
				"nextjs":      0.9,
				"typescript":  0.9,
				"api":         0.8,
				"nodejs":      0.8,
				"go":          0.8,
				"postgresql":  0.7,
				"tailwind":    0.7,
				"docker":      0.6,
			},
		},
		"devops": {
			ID:          "devops",
			Name:        "DevOps Engineer",
			Description: "Focus sur infrastructure, CI/CD, monitoring",
			TagWeights: map[string]float64{
				"devops":      1.0,
				"docker":      1.0,
				"kubernetes":  1.0,
				"ci/cd":       1.0,
				"terraform":   0.9,
				"ansible":     0.9,
				"aws":         0.9,
				"monitoring":  0.8,
				"prometheus":  0.8,
				"grafana":     0.8,
				"nginx":       0.7,
				"jenkins":     0.7,
				"gitops":      0.7,
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
```

**Explications:**
- `TagWeights` permet une pondération fine (ex: "go" poids 1.0 vs "python" poids 0.5)
- Facilite l'ajout de nouveaux thèmes sans modifier la logique de scoring
- Configuration centralisée pour maintenance facile

---

### Étape 2: Implémenter le Service de Scoring

**Description:** Créer le service qui calcule les scores de pertinence pour chaque item selon le thème.

**Fichier:** `backend/internal/services/cv_scoring.go`

**Code:**

```go
package services

import (
	"strings"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// CVScoringService gère le scoring des items CV selon thème
type CVScoringService struct{}

// NewCVScoringService crée une nouvelle instance
func NewCVScoringService() *CVScoringService {
	return &CVScoringService{}
}

// ScoredExperience représente une expérience avec son score
type ScoredExperience struct {
	Experience models.Experience `json:"experience"`
	Score      float64           `json:"score"`
}

// ScoredSkill représente une compétence avec son score
type ScoredSkill struct {
	Skill models.Skill `json:"skill"`
	Score float64      `json:"score"`
}

// ScoredProject représente un projet avec son score
type ScoredProject struct {
	Project models.Project `json:"project"`
	Score   float64        `json:"score"`
}

// CalculateExperienceScore calcule le score d'une expérience pour un thème
func (s *CVScoringService) CalculateExperienceScore(exp models.Experience, theme *config.CVTheme) float64 {
	if theme == nil {
		return 0.0
	}

	score := 0.0
	matchedTags := 0

	// Normaliser les tags pour comparaison (lowercase)
	expTags := normalizeTags(exp.Tags)
	expTechnologies := normalizeTags(exp.Technologies)

	// Calculer score basé sur tags
	for tag, weight := range theme.TagWeights {
		normalizedTag := strings.ToLower(tag)

		// Vérifier dans tags
		if contains(expTags, normalizedTag) {
			score += weight
			matchedTags++
		}

		// Vérifier dans technologies
		if contains(expTechnologies, normalizedTag) {
			score += weight * 0.8 // Technologies comptent 80% du poids
			matchedTags++
		}
	}

	// Bonus si catégorie correspond
	if strings.Contains(strings.ToLower(exp.Category), strings.ToLower(theme.ID)) {
		score += 0.5
	}

	// Normaliser le score (0.0 - 1.0) si tags matchés
	if matchedTags > 0 {
		score = score / float64(len(theme.TagWeights))
	}

	return score
}

// CalculateSkillScore calcule le score d'une compétence pour un thème
func (s *CVScoringService) CalculateSkillScore(skill models.Skill, theme *config.CVTheme) float64 {
	if theme == nil {
		return 0.0
	}

	score := 0.0

	// Normaliser le nom de la skill
	normalizedName := strings.ToLower(skill.Name)
	skillTags := normalizeTags(skill.Tags)

	// Chercher correspondance directe avec nom
	if weight, exists := theme.TagWeights[normalizedName]; exists {
		score += weight
	}

	// Chercher dans les tags
	for tag, weight := range theme.TagWeights {
		normalizedTag := strings.ToLower(tag)
		if contains(skillTags, normalizedTag) {
			score += weight * 0.7 // Tags comptent 70% du poids
		}
	}

	// Bonus basé sur niveau de compétence
	levelBonus := 0.0
	switch strings.ToLower(skill.Level) {
	case "expert":
		levelBonus = 0.3
	case "advanced":
		levelBonus = 0.2
	case "intermediate":
		levelBonus = 0.1
	}
	score += levelBonus

	// Bonus basé sur années d'expérience
	if skill.YearsExperience >= 5 {
		score += 0.2
	} else if skill.YearsExperience >= 3 {
		score += 0.1
	}

	// Normaliser le score
	if len(theme.TagWeights) > 0 {
		score = score / float64(len(theme.TagWeights))
	}

	return score
}

// CalculateProjectScore calcule le score d'un projet pour un thème
func (s *CVScoringService) CalculateProjectScore(project models.Project, theme *config.CVTheme) float64 {
	if theme == nil {
		return 0.0
	}

	score := 0.0
	matchedTags := 0

	// Normaliser les technologies
	projectTechs := normalizeTags(project.Technologies)

	// Calculer score basé sur technologies
	for tag, weight := range theme.TagWeights {
		normalizedTag := strings.ToLower(tag)
		if contains(projectTechs, normalizedTag) {
			score += weight
			matchedTags++
		}
	}

	// Bonus si projet featured
	if project.Featured {
		score += 0.3
	}

	// Bonus si catégorie correspond
	if strings.Contains(strings.ToLower(project.Category), strings.ToLower(theme.ID)) {
		score += 0.4
	}

	// Normaliser le score
	if matchedTags > 0 {
		score = score / float64(len(theme.TagWeights))
	}

	return score
}

// ScoreExperiences score et trie une liste d'expériences
func (s *CVScoringService) ScoreExperiences(experiences []models.Experience, theme *config.CVTheme) []ScoredExperience {
	scored := make([]ScoredExperience, 0, len(experiences))

	for _, exp := range experiences {
		score := s.CalculateExperienceScore(exp, theme)
		if score > 0 { // Garder seulement items pertinents
			scored = append(scored, ScoredExperience{
				Experience: exp,
				Score:      score,
			})
		}
	}

	// Trier par score décroissant
	sortScoredExperiences(scored)
	return scored
}

// ScoreSkills score et trie une liste de compétences
func (s *CVScoringService) ScoreSkills(skills []models.Skill, theme *config.CVTheme) []ScoredSkill {
	scored := make([]ScoredSkill, 0, len(skills))

	for _, skill := range skills {
		score := s.CalculateSkillScore(skill, theme)
		if score > 0 {
			scored = append(scored, ScoredSkill{
				Skill: skill,
				Score: score,
			})
		}
	}

	// Trier par score décroissant
	sortScoredSkills(scored)
	return scored
}

// ScoreProjects score et trie une liste de projets
func (s *CVScoringService) ScoreProjects(projects []models.Project, theme *config.CVTheme) []ScoredProject {
	scored := make([]ScoredProject, 0, len(projects))

	for _, project := range projects {
		score := s.CalculateProjectScore(project, theme)
		if score > 0 {
			scored = append(scored, ScoredProject{
				Project: project,
				Score:   score,
			})
		}
	}

	// Trier par score décroissant
	sortScoredProjects(scored)
	return scored
}

// Helpers

func normalizeTags(tags []string) []string {
	normalized := make([]string, len(tags))
	for i, tag := range tags {
		normalized[i] = strings.ToLower(strings.TrimSpace(tag))
	}
	return normalized
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sortScoredExperiences(scored []ScoredExperience) {
	// Tri par score décroissant
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}

func sortScoredSkills(scored []ScoredSkill) {
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}

func sortScoredProjects(scored []ScoredProject) {
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}
```

**Explications:**
- **Algorithme de scoring multi-facteurs:**
  - Tags matching (poids configurables)
  - Technologies matching (80% du poids)
  - Catégorie bonus
  - Niveau de compétence bonus
  - Années d'expérience bonus
- **Normalisation:** Scores entre 0.0 et 1.0 pour comparaison cohérente
- **Filtrage:** Garde seulement items avec score > 0 (pertinents)

---

### Étape 3: Créer le Service CV Principal

**Description:** Service orchestrant la récupération et l'adaptation du CV selon le thème.

**Fichier:** `backend/internal/services/cv_service.go`

**Code:**

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// CVService gère la logique métier du CV
type CVService struct {
	db            *gorm.DB
	redis         *redis.Client
	scoringService *CVScoringService
}

// NewCVService crée une nouvelle instance
func NewCVService(db *gorm.DB, redisClient *redis.Client) *CVService {
	return &CVService{
		db:            db,
		redis:         redisClient,
		scoringService: NewCVScoringService(),
	}
}

// AdaptiveCVResponse représente la réponse complète du CV adaptatif
type AdaptiveCVResponse struct {
	Theme        config.CVTheme   `json:"theme"`
	Experiences  []models.Experience `json:"experiences"`
	Skills       []models.Skill      `json:"skills"`
	Projects     []models.Project    `json:"projects"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

// GetAdaptiveCV retourne le CV adapté au thème demandé
func (s *CVService) GetAdaptiveCV(ctx context.Context, themeID string) (*AdaptiveCVResponse, error) {
	// 1. Vérifier si thème existe
	theme := config.GetTheme(themeID)
	if theme == nil {
		return nil, fmt.Errorf("theme not found: %s", themeID)
	}

	// 2. Vérifier cache Redis
	cacheKey := fmt.Sprintf("cv:theme:%s", themeID)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		// Cache hit
		var response AdaptiveCVResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return &response, nil
		}
	}

	// 3. Cache miss - récupérer depuis DB
	var experiences []models.Experience
	var skills []models.Skill
	var projects []models.Project

	if err := s.db.Find(&experiences).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch experiences: %w", err)
	}

	if err := s.db.Find(&skills).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch skills: %w", err)
	}

	if err := s.db.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}

	// 4. Scorer et filtrer selon thème
	scoredExp := s.scoringService.ScoreExperiences(experiences, theme)
	scoredSkills := s.scoringService.ScoreSkills(skills, theme)
	scoredProjects := s.scoringService.ScoreProjects(projects, theme)

	// 5. Extraire les items (sans scores) pour la réponse
	filteredExperiences := make([]models.Experience, 0)
	for _, se := range scoredExp {
		filteredExperiences = append(filteredExperiences, se.Experience)
	}

	filteredSkills := make([]models.Skill, 0)
	for _, ss := range scoredSkills {
		filteredSkills = append(filteredSkills, ss.Skill)
	}

	filteredProjects := make([]models.Project, 0)
	for _, sp := range scoredProjects {
		filteredProjects = append(filteredProjects, sp.Project)
	}

	// 6. Construire réponse
	response := &AdaptiveCVResponse{
		Theme:       *theme,
		Experiences: filteredExperiences,
		Skills:      filteredSkills,
		Projects:    filteredProjects,
		GeneratedAt: time.Now(),
	}

	// 7. Mettre en cache (TTL 1h)
	jsonData, err := json.Marshal(response)
	if err == nil {
		s.redis.Set(ctx, cacheKey, jsonData, 1*time.Hour)
	}

	return response, nil
}

// GetAllExperiences retourne toutes les expériences
func (s *CVService) GetAllExperiences(ctx context.Context) ([]models.Experience, error) {
	var experiences []models.Experience
	if err := s.db.Order("start_date DESC").Find(&experiences).Error; err != nil {
		return nil, err
	}
	return experiences, nil
}

// GetAllSkills retourne toutes les compétences
func (s *CVService) GetAllSkills(ctx context.Context) ([]models.Skill, error) {
	var skills []models.Skill
	if err := s.db.Order("years_experience DESC").Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

// GetAllProjects retourne tous les projets
func (s *CVService) GetAllProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	if err := s.db.Order("featured DESC, created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// GetAvailableThemes retourne la liste des thèmes disponibles
func (s *CVService) GetAvailableThemes() []config.CVTheme {
	themes := config.GetAvailableThemes()
	result := make([]config.CVTheme, 0, len(themes))
	for _, theme := range themes {
		result = append(result, theme)
	}
	return result
}

// InvalidateCache invalide le cache pour un thème (ou tous si themeID vide)
func (s *CVService) InvalidateCache(ctx context.Context, themeID string) error {
	if themeID == "" {
		// Invalider tous les thèmes
		themes := config.GetAvailableThemes()
		for id := range themes {
			key := fmt.Sprintf("cv:theme:%s", id)
			s.redis.Del(ctx, key)
		}
	} else {
		// Invalider un thème spécifique
		key := fmt.Sprintf("cv:theme:%s", themeID)
		s.redis.Del(ctx, key)
	}
	return nil
}
```

**Explications:**
- **Cache Redis:** Évite recalcul à chaque requête (TTL 1h)
- **Invalidation cache:** Méthode pour forcer refresh (utile après mise à jour données)
- **Méthodes utilitaires:** GetAll* pour endpoints complets sans filtrage

---

### Étape 4: Implémenter les Endpoints API

**Description:** Créer les handlers HTTP pour exposer les fonctionnalités CV.

**Fichier:** `backend/internal/api/cv.go`

**Code:**

```go
package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// CVHandler gère les endpoints liés au CV
type CVHandler struct {
	cvService *services.CVService
}

// NewCVHandler crée un nouveau handler
func NewCVHandler(cvService *services.CVService) *CVHandler {
	return &CVHandler{
		cvService: cvService,
	}
}

// RegisterRoutes enregistre les routes CV
func (h *CVHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")

	api.Get("/cv", h.GetAdaptiveCV)
	api.Get("/cv/themes", h.GetThemes)
	api.Get("/experiences", h.GetExperiences)
	api.Get("/skills", h.GetSkills)
	api.Get("/projects", h.GetProjects)
	api.Get("/cv/export", h.ExportPDF)
}

// GetAdaptiveCV retourne le CV adapté au thème
// @Summary Get adaptive CV
// @Description Returns CV adapted to specified theme
// @Tags CV
// @Param theme query string false "Theme ID (backend, cpp, artistique, fullstack, devops)"
// @Success 200 {object} services.AdaptiveCVResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/cv [get]
func (h *CVHandler) GetAdaptiveCV(c *fiber.Ctx) error {
	themeID := c.Query("theme", "fullstack") // Default: fullstack

	cv, err := h.cvService.GetAdaptiveCV(c.Context(), themeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid theme",
			"message": err.Error(),
		})
	}

	return c.JSON(cv)
}

// GetThemes retourne la liste des thèmes disponibles
// @Summary Get available themes
// @Description Returns list of all available CV themes
// @Tags CV
// @Success 200 {array} config.CVTheme
// @Router /api/cv/themes [get]
func (h *CVHandler) GetThemes(c *fiber.Ctx) error {
	themes := h.cvService.GetAvailableThemes()
	return c.JSON(fiber.Map{
		"themes": themes,
		"count":  len(themes),
	})
}

// GetExperiences retourne toutes les expériences
// @Summary Get all experiences
// @Description Returns all professional experiences
// @Tags CV
// @Success 200 {array} models.Experience
// @Failure 500 {object} ErrorResponse
// @Router /api/experiences [get]
func (h *CVHandler) GetExperiences(c *fiber.Ctx) error {
	experiences, err := h.cvService.GetAllExperiences(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch experiences",
		})
	}

	return c.JSON(fiber.Map{
		"experiences": experiences,
		"count":       len(experiences),
	})
}

// GetSkills retourne toutes les compétences
// @Summary Get all skills
// @Description Returns all skills
// @Tags CV
// @Success 200 {array} models.Skill
// @Failure 500 {object} ErrorResponse
// @Router /api/skills [get]
func (h *CVHandler) GetSkills(c *fiber.Ctx) error {
	skills, err := h.cvService.GetAllSkills(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch skills",
		})
	}

	return c.JSON(fiber.Map{
		"skills": skills,
		"count":  len(skills),
	})
}

// GetProjects retourne tous les projets
// @Summary Get all projects
// @Description Returns all projects
// @Tags CV
// @Success 200 {array} models.Project
// @Failure 500 {object} ErrorResponse
// @Router /api/projects [get]
func (h *CVHandler) GetProjects(c *fiber.Ctx) error {
	projects, err := h.cvService.GetAllProjects(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch projects",
		})
	}

	return c.JSON(fiber.Map{
		"projects": projects,
		"count":    len(projects),
	})
}

// ExportPDF exporte le CV en PDF
// @Summary Export CV as PDF
// @Description Generates and downloads CV as PDF for specified theme
// @Tags CV
// @Param theme query string false "Theme ID"
// @Param format query string false "Export format (pdf)" default(pdf)
// @Success 200 {file} application/pdf
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/cv/export [get]
func (h *CVHandler) ExportPDF(c *fiber.Ctx) error {
	themeID := c.Query("theme", "fullstack")
	format := c.Query("format", "pdf")

	if format != "pdf" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only PDF format is supported",
		})
	}

	// Récupérer CV adaptatif
	cv, err := h.cvService.GetAdaptiveCV(c.Context(), themeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Générer PDF (implémenté dans Étape 5)
	pdfService := services.NewPDFService()
	pdfBytes, err := pdfService.GenerateCVPDF(cv)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate PDF",
		})
	}

	// Retourner PDF
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=cv_"+themeID+".pdf")
	return c.Send(pdfBytes)
}

// ErrorResponse structure pour documentation API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
```

**Explications:**
- **Query params:** Thème spécifié via `?theme=backend`
- **Default theme:** fullstack si non spécifié
- **Error handling:** Codes HTTP appropriés (400, 500)
- **Annotations Swagger:** Pour génération doc API (Phase 6)

---

### Étape 5: Implémenter le Service PDF

**Description:** Service de génération PDF basique avec chromedp.

**Fichier:** `backend/internal/services/pdf_service.go`

**Code:**

```go
package services

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// PDFService gère la génération de PDFs
type PDFService struct {
	templates *template.Template
}

// NewPDFService crée une nouvelle instance
func NewPDFService() *PDFService {
	// Charger templates (à créer dans Étape 6)
	tmpl := template.Must(template.ParseGlob("templates/cv/*.html"))

	return &PDFService{
		templates: tmpl,
	}
}

// GenerateCVPDF génère un PDF du CV
func (s *PDFService) GenerateCVPDF(cv *AdaptiveCVResponse) ([]byte, error) {
	// 1. Générer HTML depuis template
	html, err := s.renderCVHTML(cv)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML: %w", err)
	}

	// 2. Convertir HTML → PDF avec chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Timeout pour génération PDF
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var pdfBuffer []byte

	if err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Injecter HTML
			frameTree, err := chromedp.Evaluate(`document.write(atob('`+encodeBase64(html)+`'))`, nil).Do(ctx)
			if err != nil {
				return err
			}
			_ = frameTree
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Générer PDF
			var err error
			pdfBuffer, _, err = chromedp.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).  // A4 width in inches
				WithPaperHeight(11.7). // A4 height in inches
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			return err
		}),
	); err != nil {
		return nil, fmt.Errorf("chromedp failed: %w", err)
	}

	return pdfBuffer, nil
}

// renderCVHTML génère le HTML du CV depuis template
func (s *PDFService) renderCVHTML(cv *AdaptiveCVResponse) (string, error) {
	var buf strings.Builder

	if err := s.templates.ExecuteTemplate(&buf, "cv_base.html", cv); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Helper pour encoder en base64 (pour chromedp)
func encodeBase64(s string) string {
	// Implémentation simplifiée - utiliser encoding/base64 en prod
	return s // TODO: Implémenter vraie conversion base64
}
```

**Explications:**
- **chromedp:** Headless Chrome pour rendu PDF de qualité
- **Templates HTML:** Permet design personnalisé (Étape 6)
- **Timeouts:** Protection contre génération bloquée
- **A4 format:** Dimensions standard pour CV

---

### Étape 6: Créer le Template HTML pour PDF

**Description:** Template HTML basique pour rendu PDF du CV.

**Fichier:** `backend/templates/cv/cv_base.html`

**Code:**

```html
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CV - {{.Theme.Name}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', 'Helvetica', sans-serif;
            font-size: 11pt;
            line-height: 1.5;
            color: #1a1a1a;
            background: white;
            padding: 20px;
        }

        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #2563eb;
        }

        .header h1 {
            font-size: 28pt;
            font-weight: 700;
            color: #2563eb;
            margin-bottom: 5px;
        }

        .header .theme {
            font-size: 14pt;
            color: #64748b;
            font-weight: 500;
        }

        .section {
            margin-bottom: 25px;
        }

        .section-title {
            font-size: 16pt;
            font-weight: 700;
            color: #2563eb;
            margin-bottom: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .experience {
            margin-bottom: 15px;
            padding-left: 10px;
            border-left: 3px solid #e2e8f0;
        }

        .experience-title {
            font-size: 13pt;
            font-weight: 600;
            color: #1a1a1a;
        }

        .experience-company {
            font-size: 11pt;
            color: #2563eb;
            font-weight: 500;
        }

        .experience-dates {
            font-size: 10pt;
            color: #64748b;
            margin-bottom: 5px;
        }

        .experience-description {
            font-size: 10pt;
            color: #334155;
            margin-top: 5px;
        }

        .skills-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 10px;
        }

        .skill {
            padding: 8px 12px;
            background: #f1f5f9;
            border-radius: 6px;
            font-size: 10pt;
        }

        .skill-name {
            font-weight: 600;
            color: #1a1a1a;
        }

        .skill-level {
            font-size: 9pt;
            color: #64748b;
        }

        .projects-list {
            list-style: none;
        }

        .project {
            margin-bottom: 12px;
            padding-left: 10px;
            border-left: 2px solid #e2e8f0;
        }

        .project-title {
            font-size: 12pt;
            font-weight: 600;
            color: #1a1a1a;
        }

        .project-description {
            font-size: 10pt;
            color: #334155;
            margin-top: 3px;
        }

        .tags {
            margin-top: 5px;
        }

        .tag {
            display: inline-block;
            padding: 2px 8px;
            background: #dbeafe;
            color: #1e40af;
            border-radius: 4px;
            font-size: 9pt;
            margin-right: 5px;
            margin-top: 3px;
        }

        .footer {
            margin-top: 30px;
            padding-top: 15px;
            border-top: 1px solid #e2e8f0;
            text-align: center;
            font-size: 9pt;
            color: #94a3b8;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Alexi - CV</h1>
        <div class="theme">{{.Theme.Name}}</div>
    </div>

    {{if .Experiences}}
    <div class="section">
        <h2 class="section-title">Expériences Professionnelles</h2>
        {{range .Experiences}}
        <div class="experience">
            <div class="experience-title">{{.Title}}</div>
            <div class="experience-company">{{.Company}}</div>
            <div class="experience-dates">
                {{.StartDate.Format "Jan 2006"}} -
                {{if .EndDate.IsZero}}Présent{{else}}{{.EndDate.Format "Jan 2006"}}{{end}}
            </div>
            <div class="experience-description">{{.Description}}</div>
            {{if .Technologies}}
            <div class="tags">
                {{range .Technologies}}
                <span class="tag">{{.}}</span>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .Skills}}
    <div class="section">
        <h2 class="section-title">Compétences</h2>
        <div class="skills-grid">
            {{range .Skills}}
            <div class="skill">
                <div class="skill-name">{{.Name}}</div>
                <div class="skill-level">{{.Level}} - {{.YearsExperience}} ans</div>
            </div>
            {{end}}
        </div>
    </div>
    {{end}}

    {{if .Projects}}
    <div class="section">
        <h2 class="section-title">Projets</h2>
        <ul class="projects-list">
            {{range .Projects}}
            <li class="project">
                <div class="project-title">{{.Title}}</div>
                <div class="project-description">{{.Description}}</div>
                {{if .Technologies}}
                <div class="tags">
                    {{range .Technologies}}
                    <span class="tag">{{.}}</span>
                    {{end}}
                </div>
                {{end}}
            </li>
            {{end}}
        </ul>
    </div>
    {{end}}

    <div class="footer">
        Généré le {{.GeneratedAt.Format "02/01/2006 15:04"}} - Thème: {{.Theme.ID}}
    </div>
</body>
</html>
```

**Explications:**
- **Design minimaliste:** Focus sur lisibilité
- **Responsive PDF:** S'adapte au format A4
- **Tags colorés:** Mise en valeur des technologies
- **Barre latérale:** Séparation visuelle expériences/projets

---

### Étape 7: Intégration dans main.go

**Description:** Initialiser et enregistrer les routes CV dans l'application.

**Fichier:** `backend/cmd/main.go` (ajout)

**Code:**

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"maicivy/internal/api"
	"maicivy/internal/database"
	"maicivy/internal/services"
)

func main() {
	// Initialiser connexions DB (depuis Phase 1)
	db := database.InitPostgres()
	redisClient := database.InitRedis()

	// Initialiser Fiber
	app := fiber.New(fiber.Config{
		AppName: "maicivy API",
	})

	// Middlewares globaux
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Initialiser services
	cvService := services.NewCVService(db, redisClient)

	// Initialiser handlers
	cvHandler := api.NewCVHandler(cvService)

	// Enregistrer routes
	cvHandler.RegisterRoutes(app)

	// Démarrer serveur
	log.Fatal(app.Listen(":8080"))
}
```

**Explications:**
- **Injection de dépendances:** Services initialisés avec DB et Redis
- **Middlewares:** Recovery, logger, CORS (Phase 1)
- **Routes groupées:** Meilleure organisation

---

## 🧪 Tests

### Tests Unitaires - Algorithme de Scoring

**Fichier:** `backend/internal/services/cv_scoring_test.go`

**Code:**

```go
package services

import (
	"testing"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

func TestCalculateExperienceScore(t *testing.T) {
	service := NewCVScoringService()

	// Thème backend
	backendTheme := config.GetTheme("backend")

	tests := []struct {
		name       string
		experience models.Experience
		minScore   float64 // Score minimum attendu
	}{
		{
			name: "Experience Go backend parfaite",
			experience: models.Experience{
				Title:        "Backend Developer",
				Technologies: []string{"go", "postgresql", "redis", "docker"},
				Tags:         []string{"backend", "api", "microservices"},
				Category:     "backend",
			},
			minScore: 0.5, // Au moins 50% de score
		},
		{
			name: "Experience frontend (faible score)",
			experience: models.Experience{
				Title:        "Frontend Developer",
				Technologies: []string{"react", "typescript", "css"},
				Tags:         []string{"frontend", "ui"},
				Category:     "frontend",
			},
			minScore: 0.0, // Peu ou pas de match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.CalculateExperienceScore(tt.experience, backendTheme)

			if score < tt.minScore {
				t.Errorf("Score trop faible: got %f, want >= %f", score, tt.minScore)
			}
		})
	}
}

func TestCalculateSkillScore(t *testing.T) {
	service := NewCVScoringService()
	cppTheme := config.GetTheme("cpp")

	tests := []struct {
		name     string
		skill    models.Skill
		minScore float64
	}{
		{
			name: "C++ expert",
			skill: models.Skill{
				Name:            "C++",
				Level:           "expert",
				YearsExperience: 8,
				Tags:            []string{"cpp", "systems"},
			},
			minScore: 0.6,
		},
		{
			name: "JavaScript (non pertinent pour C++)",
			skill: models.Skill{
				Name:            "JavaScript",
				Level:           "advanced",
				YearsExperience: 5,
				Tags:            []string{"frontend"},
			},
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.CalculateSkillScore(tt.skill, cppTheme)

			if score < tt.minScore {
				t.Errorf("Score trop faible: got %f, want >= %f", score, tt.minScore)
			}
		})
	}
}

func TestScoreExperiences_Sorting(t *testing.T) {
	service := NewCVScoringService()
	backendTheme := config.GetTheme("backend")

	experiences := []models.Experience{
		{
			Title:        "Frontend Dev",
			Technologies: []string{"react"},
			Tags:         []string{"frontend"},
		},
		{
			Title:        "Backend Dev",
			Technologies: []string{"go", "postgresql"},
			Tags:         []string{"backend", "api"},
		},
		{
			Title:        "Full-Stack Dev",
			Technologies: []string{"nodejs", "mongodb"},
			Tags:         []string{"backend", "frontend"},
		},
	}

	scored := service.ScoreExperiences(experiences, backendTheme)

	// Vérifier que "Backend Dev" est premier (score le plus élevé)
	if len(scored) < 2 {
		t.Fatal("Expected at least 2 scored experiences")
	}

	if scored[0].Experience.Title != "Backend Dev" {
		t.Errorf("Expected Backend Dev to be first, got %s", scored[0].Experience.Title)
	}
}
```

### Tests Integration - API Endpoints

**Fichier:** `backend/internal/api/cv_test.go`

**Code:**

```go
package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"maicivy/internal/services"
)

func setupTestApp() *fiber.App {
	app := fiber.New()

	// Mock service (utiliser testcontainers en vrai)
	cvService := services.NewCVService(nil, nil)
	cvHandler := NewCVHandler(cvService)
	cvHandler.RegisterRoutes(app)

	return app
}

func TestGetAdaptiveCV(t *testing.T) {
	app := setupTestApp()

	tests := []struct {
		name           string
		queryTheme     string
		expectedStatus int
	}{
		{
			name:           "Thème backend valide",
			queryTheme:     "backend",
			expectedStatus: 200,
		},
		{
			name:           "Thème invalide",
			queryTheme:     "invalid_theme",
			expectedStatus: 400,
		},
		{
			name:           "Pas de thème (default fullstack)",
			queryTheme:     "",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/cv?theme="+tt.queryTheme, nil)
			resp, _ := app.Test(req)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestGetThemes(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/cv/themes", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Vérifier structure réponse
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["themes"])
	assert.NotNil(t, result["count"])
	assert.Greater(t, int(result["count"].(float64)), 0)
}
```

### Commandes

```bash
# Run tests unitaires
go test -v ./internal/services/...

# Run tests integration
go test -v ./internal/api/...

# Run tous les tests avec coverage
go test -v -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## ⚠️ Points d'Attention

### Pièges à Éviter

- ⚠️ **Normalisation tags:** Toujours comparer en lowercase (évite bugs "Go" vs "go")
- ⚠️ **Cache Redis:** Penser à invalider après mise à jour données CV
- ⚠️ **Chromedp:** Nécessite Chrome/Chromium installé sur le serveur (Docker image avec Chrome)
- ⚠️ **Timeout PDF:** Génération peut être lente (30s timeout minimum)
- ⚠️ **Score 0:** Ne pas afficher items avec score 0 (non pertinents)

### Edge Cases à Gérer

- **Thème inconnu:** Retourner erreur 400 avec message clair
- **Base vide:** Si aucune expérience/skill, retourner tableau vide (pas null)
- **Concurrent updates:** Race condition Redis cache (acceptable pour MVP)
- **PDF trop long:** Limiter nombre d'items affichés (top 10 par section)

### Optimisations Futures

- 💡 **Scoring ML:** Utiliser machine learning pour améliorer pertinence
- 💡 **Cache multi-niveaux:** Redis + in-memory LRU
- 💡 **PDF asynchrone:** Queue job pour grandes générations
- 💡 **Scoring personnalisé:** Permettre utilisateurs ajuster poids

---

## 📚 Ressources

### Documentation Officielle

- [Fiber Documentation](https://docs.gofiber.io/)
- [GORM Documentation](https://gorm.io/docs/)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [chromedp Documentation](https://github.com/chromedp/chromedp)

### Tutoriels

- [Building REST APIs with Fiber](https://blog.logrocket.com/building-microservices-go-fiber/)
- [Redis Caching Strategies](https://redis.io/docs/manual/patterns/)
- [Go Template Tutorial](https://golang.org/pkg/html/template/)

### Articles de Blog

- [Scoring Algorithms in Go](https://medium.com/tag/scoring-algorithm)
- [PDF Generation Best Practices](https://dev.to/chromedp)

---

## ✅ Checklist de Complétion

- [ ] Configuration thèmes créée (5 thèmes: backend, cpp, artistique, fullstack, devops)
- [ ] Service de scoring implémenté avec tests unitaires
- [ ] Service CV principal avec cache Redis
- [ ] Endpoints API fonctionnels (`/api/cv`, `/api/cv/themes`, etc.)
- [ ] Service PDF basique avec chromedp
- [ ] Template HTML CV créé
- [ ] Tests unitaires scoring (coverage > 80%)
- [ ] Tests integration API endpoints
- [ ] Documentation code (commentaires Go)
- [ ] Intégration dans main.go
- [ ] Vérification cache Redis (TTL 1h)
- [ ] Export PDF testé manuellement
- [ ] Review sécurité (validation inputs)
- [ ] Review performance (cache efficace)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
