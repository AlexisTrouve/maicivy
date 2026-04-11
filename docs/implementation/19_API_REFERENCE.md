# API_REFERENCE

## 📋 Métadonnées

- **Phase:** 6 (Continu)
- **Priorité:** 🟢 MOYENNE
- **Complexité:** ⭐⭐ (2/5)
- **Prérequis:** Toutes APIs implémentées (docs 06, 09, 11)
- **Temps estimé:** 1-2 jours (setup auto-génération)
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Fournir une documentation complète et interactive de l'API maicivy via OpenAPI/Swagger, incluant tous les endpoints de CV, lettres IA et analytics. Auto-générer la spécification via swaggo et exposer Swagger UI pour test interactif.

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────────────────────────────────────────────────┐
│                    API Documentation                    │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  OpenAPI 3.0 Specification (YAML)                       │
│  ├─ Schemas (Request/Response)                          │
│  ├─ Endpoints (Paths)                                   │
│  ├─ Authentication (Cookies)                            │
│  └─ Error Codes                                         │
│                                                           │
│  Auto-Generation (swaggo)                               │
│  ├─ Parse annotations Go                                │
│  ├─ Generate openapi.json                               │
│  └─ Update Swagger UI files                             │
│                                                           │
│  Swagger UI (/api/docs)                                 │
│  ├─ Interactive endpoint testing                        │
│  ├─ Schema visualization                                │
│  └─ Curl examples generation                            │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

### Design Decisions

1. **OpenAPI 3.0** : standard industriel, largement supporté
2. **Auto-génération** : swaggo lit les annotations Go, évite désynchronisation
3. **Swagger UI** : interface web interactive pour test endpoints
4. **Exemples curl** : facilite integration client/debugging
5. **Standardisation** : réponses JSON structurées, erreurs cohérentes

---

## 📦 Dépendances

### Bibliothèques Go

```bash
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/files
go get -u github.com/swaggo/gin-swagger
```

### Packages NPM

Aucun requis (API pure Go)

### Services Externes

Aucun service externe requis pour documentation

---

## 🔨 Implémentation

### Étape 1: Installation et Configuration swaggo

**Description:** Installer swaggo et configurer le générateur de documentation

**Code:**

```bash
# Installation globale swaggo
go install github.com/swaggo/swag/cmd/swag@latest

# Initialiser swaggo dans le projet backend
cd backend
swag init -g cmd/main.go

# Cette commande créera:
# - docs/docs.go
# - docs/swagger.yaml
# - docs/swagger.json
```

**Explications:** swaggo parse les commentaires Go avec pattern `@router` et génère la spec OpenAPI

---

### Étape 2: Structure des Fichiers Documentation

**Description:** Organisation des fichiers OpenAPI et Swagger

**Structure:**

```
backend/
├── cmd/
│   └── main.go                 # Annotations doc générale + routes
├── internal/
│   ├── api/
│   │   ├── cv.go              # Endpoints CV avec annotations
│   │   ├── letters.go         # Endpoints Letters avec annotations
│   │   ├── analytics.go       # Endpoints Analytics avec annotations
│   │   └── health.go          # Health check
│   └── models/
│       ├── request.go         # DTOs request
│       └── response.go        # DTOs response
├── docs/
│   ├── docs.go                # Auto-généré
│   ├── swagger.yaml           # Auto-généré
│   ├── swagger.json           # Auto-généré
│   └── swagger.html           # Swagger UI
└── swag-config.go             # Configuration swaggo
```

---

### Étape 3: Annotations dans main.go

**Description:** Documenter l'API générale et les authentifications

**Code:**

```go
// backend/cmd/main.go

// @title           maicivy API
// @version         1.0
// @description     CV interactif intelligent avec génération de lettres par IA
// @termsOfService  http://swagger.io/terms/
//
// @contact.name   Alexi
// @contact.url    https://maicivy.dev
// @contact.email  contact@maicivy.dev
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host            localhost:5000
// @basePath         /api
// @schemes          http https
//
// @securityDefinitions.apikey CookieAuth
// @in              header
// @name            Cookie
// @description     "Session cookie (automatically set on first visit)"
//
// @externalDocs.description  OpenAPI docs
// @externalDocs.url          https://swagger.io/resources/open-api/
//
// func main() {
//     // ...
// }
```

**Explications:** Ces annotations définissent les métadonnées globales, authentification et schémas de base

---

### Étape 4: DTOs pour Requêtes/Réponses

**Description:** Définir les structures de données documentées

**Code:**

```go
// backend/internal/models/request.go

package models

// CVRequest représente les paramètres de requête pour GET /api/cv
type CVRequest struct {
	Theme string `query:"theme" example:"backend" validate:"required"`
}

// CVExportRequest paramètres pour export PDF
type CVExportRequest struct {
	Theme  string `query:"theme" example:"backend" validate:"required"`
	Format string `query:"format" example:"pdf" validate:"required,oneof=pdf"`
}

// LetterGenerateRequest payload pour POST /api/letters/generate
type LetterGenerateRequest struct {
	CompanyName string `json:"company_name" validate:"required,min=1,max=255" example:"Google"`
}

// AnalyticsEventRequest payload pour POST /api/analytics/event
type AnalyticsEventRequest struct {
	EventType string                 `json:"event_type" validate:"required" example:"page_view"`
	EventData map[string]interface{} `json:"event_data" example:"{\"page\": \"/cv\", \"theme\": \"backend\"}"`
}
```

```go
// backend/internal/models/response.go

package models

// CVResponse représente un CV complet
type CVResponse struct {
	Theme       string         `json:"theme" example:"backend"`
	Title       string         `json:"title" example:"Backend Engineer"`
	Summary     string         `json:"summary" example:"Experienced Go developer..."`
	Experiences []Experience   `json:"experiences"`
	Skills      []Skill        `json:"skills"`
	Projects    []Project      `json:"projects"`
	GeneratedAt string         `json:"generated_at" example:"2025-12-08T10:30:00Z"`
}

// Experience représente une expérience professionnelle
type Experience struct {
	ID            string   `json:"id" example:"exp_001"`
	Title         string   `json:"title" example:"Senior Backend Engineer"`
	Company       string   `json:"company" example:"TechCorp"`
	Description   string   `json:"description" example:"Lead Go development..."`
	StartDate     string   `json:"start_date" example:"2023-01-01"`
	EndDate       string   `json:"end_date" example:"2025-12-08"`
	Technologies  []string `json:"technologies" example:"[\"Go\",\"PostgreSQL\",\"Redis\"]"`
	Tags          []string `json:"tags" example:"[\"backend\",\"databases\",\"devops\"]"`
	Category      string   `json:"category" example:"backend"`
	RelevanceScore float32 `json:"relevance_score" example:"0.95"`
}

// Skill représente une compétence
type Skill struct {
	ID               string   `json:"id" example:"skill_001"`
	Name             string   `json:"name" example:"Go"`
	Level            string   `json:"level" example:"expert" validate:"oneof=beginner intermediate advanced expert"`
	Category         string   `json:"category" example:"backend"`
	Tags             []string `json:"tags" example:"[\"languages\",\"backend\"]"`
	YearsExperience  int      `json:"years_experience" example:"5"`
	RelevanceScore   float32  `json:"relevance_score" example:"0.98"`
}

// Project représente un projet
type Project struct {
	ID              string   `json:"id" example:"proj_001"`
	Title           string   `json:"title" example:"maicivy"`
	Description     string   `json:"description" example:"CV AI-powered..."`
	GitHubURL       string   `json:"github_url" example:"https://github.com/alexi/maicivy"`
	DemoURL         string   `json:"demo_url" example:"https://maicivy.dev"`
	Technologies    []string `json:"technologies" example:"[\"Go\",\"Next.js\",\"PostgreSQL\"]"`
	Category        string   `json:"category" example:"full-stack"`
	Featured        bool     `json:"featured" example:"true"`
	Stars           int      `json:"stars" example:"42"`
	RelevanceScore  float32  `json:"relevance_score" example:"0.88"`
}

// LetterResponse représente une lettre générée
type LetterResponse struct {
	ID              string `json:"id" example:"letter_001"`
	CompanyName     string `json:"company_name" example:"Google"`
	LetterType      string `json:"letter_type" example:"motivation" validate:"oneof=motivation anti-motivation"`
	Content         string `json:"content" example:"Cher Monsieur..."`
	GeneratedAt     string `json:"generated_at" example:"2025-12-08T10:30:00Z"`
	TokensUsed      int    `json:"tokens_used" example:"256"`
	CacheHit        bool   `json:"cache_hit" example:"false"`
}

// AnalyticsStatsResponse statistiques analytics
type AnalyticsStatsResponse struct {
	Period               string                 `json:"period" example:"day" validate:"oneof=day week month"`
	TotalVisitors        int64                  `json:"total_visitors" example:"256"`
	UniqueVisitors       int64                  `json:"unique_visitors" example:"128"`
	PageViews            int64                  `json:"page_views" example:"512"`
	TopThemes            []ThemeCount           `json:"top_themes"`
	LettersGenerated     int64                  `json:"letters_generated" example:"42"`
	AverageSessionLength float64                `json:"average_session_length" example:"245.5"`
	Timestamp            string                 `json:"timestamp" example:"2025-12-08T23:59:59Z"`
}

// ThemeCount représente compteur pour un thème
type ThemeCount struct {
	Theme string `json:"theme" example:"backend"`
	Count int64  `json:"count" example:"87"`
	Percentage float32 `json:"percentage" example:"25.5"`
}

// ErrorResponse structure d'erreur standardisée
type ErrorResponse struct {
	Status    int    `json:"status" example:"400"`
	Code      string `json:"code" example:"INVALID_THEME"`
	Message   string `json:"message" example:"Theme 'xyz' is not valid"`
	Timestamp string `json:"timestamp" example:"2025-12-08T10:30:00Z"`
}

// PaginatedResponse wrapper pour réponses paginées
type PaginatedResponse struct {
	Data      []interface{}      `json:"data"`
	Meta      PaginationMeta     `json:"meta"`
}

// PaginationMeta métadonnées de pagination
type PaginationMeta struct {
	Total      int `json:"total" example:"256"`
	Page       int `json:"page" example:"1"`
	PageSize   int `json:"page_size" example:"20"`
	TotalPages int `json:"total_pages" example:"13"`
}
```

**Explications:** Les DTOs avec tags `json` et `example` sont lus par swaggo pour générer les schemas OpenAPI

---

### Étape 5: Endpoints CV

**Description:** Documenter les endpoints CV avec annotations

**Code:**

```go
// backend/internal/api/cv.go

package api

import (
	"github.com/gofiber/fiber/v2"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

// GetCV récupère le CV adapté selon le thème
// @Summary      Get adaptive CV
// @Description  Retourne un CV filtré et adapté selon le thème spécifié
// @Tags         CV
// @Accept       json
// @Produce      json
// @Param        theme  query    string  true  "Thème CV (backend, frontend, fullstack, etc.)"
// @Success      200    {object}  models.CVResponse
// @Failure      400    {object}  models.ErrorResponse  "Theme invalide"
// @Failure      500    {object}  models.ErrorResponse  "Erreur serveur"
// @Router       /cv [get]
// @Example      curl -X GET "http://localhost:5000/api/cv?theme=backend" -H "Content-Type: application/json"
func (h *Handler) GetCV(c *fiber.Ctx) error {
	theme := c.Query("theme")
	if theme == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "THEME_REQUIRED",
			Message: "Query parameter 'theme' is required",
		})
	}

	cv, err := h.cvService.GetCVByTheme(c.Context(), theme)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(cv)
}

// GetCVThemes récupère la liste des thèmes disponibles
// @Summary      Get available CV themes
// @Description  Retourne la liste complète des thèmes CV disponibles
// @Tags         CV
// @Produce      json
// @Success      200  {array}   string  "Liste de thèmes"
// @Failure      500  {object}  models.ErrorResponse
// @Router       /cv/themes [get]
// @Example      curl -X GET "http://localhost:5000/api/cv/themes" -H "Content-Type: application/json"
func (h *Handler) GetCVThemes(c *fiber.Ctx) error {
	themes := h.cvService.GetAvailableThemes()
	return c.JSON(themes)
}

// GetExperiences récupère toutes les expériences
// @Summary      Get all experiences
// @Description  Retourne la liste complète de toutes les expériences professionnelles
// @Tags         CV
// @Accept       json
// @Produce      json
// @Param        page   query    int     false  "Numéro page (default: 1)"
// @Param        limit  query    int     false  "Résultats par page (default: 20)"
// @Success      200    {object}  models.PaginatedResponse
// @Failure      400    {object}  models.ErrorResponse  "Paramètres invalides"
// @Failure      500    {object}  models.ErrorResponse
// @Router       /experiences [get]
// @Example      curl -X GET "http://localhost:5000/api/experiences?page=1&limit=10" -H "Content-Type: application/json"
func (h *Handler) GetExperiences(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	experiences, total, err := h.cvService.GetExperiences(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(models.PaginatedResponse{
		Data: experiences,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   limit,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

// GetSkills récupère toutes les compétences
// @Summary      Get all skills
// @Description  Retourne la liste complète de toutes les compétences avec niveaux
// @Tags         CV
// @Produce      json
// @Param        page   query    int     false  "Numéro page (default: 1)"
// @Param        limit  query    int     false  "Résultats par page (default: 20)"
// @Success      200    {object}  models.PaginatedResponse
// @Failure      500    {object}  models.ErrorResponse
// @Router       /skills [get]
// @Example      curl -X GET "http://localhost:5000/api/skills?page=1&limit=20" -H "Content-Type: application/json"
func (h *Handler) GetSkills(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	skills, total, err := h.cvService.GetSkills(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(models.PaginatedResponse{
		Data: skills,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   limit,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

// GetProjects récupère tous les projets
// @Summary      Get all projects
// @Description  Retourne la liste complète de tous les projets avec infos GitHub
// @Tags         CV
// @Produce      json
// @Param        page   query    int     false  "Numéro page (default: 1)"
// @Param        limit  query    int     false  "Résultats par page (default: 20)"
// @Success      200    {object}  models.PaginatedResponse
// @Failure      500    {object}  models.ErrorResponse
// @Router       /projects [get]
// @Example      curl -X GET "http://localhost:5000/api/projects?page=1&limit=20" -H "Content-Type: application/json"
func (h *Handler) GetProjects(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	projects, total, err := h.cvService.GetProjects(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(models.PaginatedResponse{
		Data: projects,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   limit,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

// ExportCV exporte le CV en PDF
// @Summary      Export CV as PDF
// @Description  Génère et retourne un PDF du CV adapté au thème
// @Tags         CV
// @Produce      application/pdf
// @Param        theme  query    string  true  "Thème CV"
// @Param        format query    string  true  "Format export (pdf)"
// @Success      200    {file}   string  "PDF file"
// @Failure      400    {object}  models.ErrorResponse  "Paramètres invalides"
// @Failure      500    {object}  models.ErrorResponse
// @Router       /cv/export [get]
// @Example      curl -X GET "http://localhost:5000/api/cv/export?theme=backend&format=pdf" -H "Accept: application/pdf" -o cv.pdf
func (h *Handler) ExportCV(c *fiber.Ctx) error {
	theme := c.Query("theme")
	format := c.Query("format")

	if theme == "" || format == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "MISSING_PARAMS",
			Message: "Query parameters 'theme' and 'format' are required",
		})
	}

	if format != "pdf" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "INVALID_FORMAT",
			Message: "Only 'pdf' format is supported",
		})
	}

	pdfBytes, err := h.cvService.ExportCVAsPDF(c.Context(), theme)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "PDF_GENERATION_ERROR",
			Message: err.Error(),
		})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=cv_"+theme+".pdf")
	return c.Send(pdfBytes)
}
```

**Explications:**
- Les commentaires `@Summary`, `@Description`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Example` sont parsés par swaggo
- Chaque endpoint documenté de manière claire
- Erreurs avec codes standardisés

---

### Étape 6: Endpoints Letters

**Description:** Documenter les endpoints de génération lettres IA

**Code:**

```go
// backend/internal/api/letters.go

package api

import (
	"github.com/gofiber/fiber/v2"
	"maicivy/internal/models"
)

// GenerateLetter génère des lettres de motivation et anti-motivation
// @Summary      Generate motivation and anti-motivation letters
// @Description  Génère deux lettres (motivation + anti-motivation) pour une entreprise. Limité à 5 par jour par session.
// @Tags         Letters
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      models.LetterGenerateRequest  true  "Company details"
// @Success      200      {array}   models.LetterResponse         "Deux lettres générées"
// @Failure      400      {object}  models.ErrorResponse          "Données invalides"
// @Failure      403      {object}  models.ErrorResponse          "Accès refusé (< 3 visites)"
// @Failure      429      {object}  models.ErrorResponse          "Rate limit atteint (5/jour)"
// @Failure      500      {object}  models.ErrorResponse          "Erreur génération"
// @Router       /letters/generate [post]
// @Example      curl -X POST "http://localhost:5000/api/letters/generate" \
//               -H "Content-Type: application/json" \
//               -d '{"company_name": "Google"}' \
//               -b "session=abc123"
func (h *Handler) GenerateLetter(c *fiber.Ctx) error {
	var req models.LetterGenerateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	if req.CompanyName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "COMPANY_NAME_REQUIRED",
			Message: "Field 'company_name' is required",
		})
	}

	sessionID := c.Cookies("session")
	if sessionID == "" {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Status:  403,
			Code:    "NO_SESSION",
			Message: "No valid session. Please visit the site first.",
		})
	}

	// Check access gate (3 visits or detected profile)
	hasAccess, err := h.letterService.CheckAccessGate(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Status:  403,
			Code:    "ACCESS_GATE_LOCKED",
			Message: "AI features available after 3 visits",
		})
	}

	// Check rate limit
	allowed, remaining, err := h.letterService.CheckRateLimit(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	if !allowed {
		c.Set("Retry-After", "86400")
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Status:  429,
			Code:    "RATE_LIMIT_EXCEEDED",
			Message: "Max 5 generations per day. Try again tomorrow.",
		})
	}

	// Generate letters asynchronously
	letters, err := h.letterService.GenerateLetters(c.Context(), req.CompanyName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "GENERATION_ERROR",
			Message: "Failed to generate letters",
		})
	}

	// Track in history
	_ = h.letterService.SaveToHistory(c.Context(), sessionID, req.CompanyName, letters)

	// Return both letters
	return c.JSON(letters)
}

// GetLetter récupère une lettre spécifique
// @Summary      Get letter by ID
// @Description  Retourne une lettre générée antérieurement
// @Tags         Letters
// @Produce      json
// @Security     CookieAuth
// @Param        id  path      string  true  "Letter ID"
// @Success      200  {object}  models.LetterResponse
// @Failure      404  {object}  models.ErrorResponse  "Lettre non trouvée"
// @Failure      403  {object}  models.ErrorResponse  "Accès non autorisé"
// @Failure      500  {object}  models.ErrorResponse
// @Router       /letters/{id} [get]
// @Example      curl -X GET "http://localhost:5000/api/letters/letter_001" \
//               -b "session=abc123"
func (h *Handler) GetLetter(c *fiber.Ctx) error {
	id := c.Params("id")
	sessionID := c.Cookies("session")

	letter, err := h.letterService.GetLetterByID(c.Context(), id, sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Status:  404,
			Code:    "LETTER_NOT_FOUND",
			Message: "Letter not found",
		})
	}

	return c.JSON(letter)
}

// GetLetterPDF retourne une lettre en PDF
// @Summary      Get letter as PDF
// @Description  Retourne un PDF formaté de la lettre
// @Tags         Letters
// @Produce      application/pdf
// @Security     CookieAuth
// @Param        id  path      string  true  "Letter ID"
// @Success      200  {file}   string   "PDF file"
// @Failure      404  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /letters/{id}/pdf [get]
// @Example      curl -X GET "http://localhost:5000/api/letters/letter_001/pdf" \
//               -b "session=abc123" \
//               -o letter.pdf
func (h *Handler) GetLetterPDF(c *fiber.Ctx) error {
	id := c.Params("id")
	sessionID := c.Cookies("session")

	pdfBytes, letterInfo, err := h.letterService.GetLetterPDF(c.Context(), id, sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Status:  404,
			Code:    "LETTER_NOT_FOUND",
			Message: "Letter not found",
		})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=letter_"+letterInfo.CompanyName+".pdf")
	return c.Send(pdfBytes)
}

// GetLetterHistory récupère l'historique des lettres générées
// @Summary      Get letter generation history
// @Description  Retourne l'historique des lettres générées pour la session actuelle
// @Tags         Letters
// @Produce      json
// @Security     CookieAuth
// @Param        page   query    int     false  "Page number (default: 1)"
// @Param        limit  query    int     false  "Results per page (default: 20)"
// @Success      200    {object}  models.PaginatedResponse
// @Failure      403    {object}  models.ErrorResponse  "No valid session"
// @Failure      500    {object}  models.ErrorResponse
// @Router       /letters/history [get]
// @Example      curl -X GET "http://localhost:5000/api/letters/history?page=1&limit=10" \
//               -b "session=abc123"
func (h *Handler) GetLetterHistory(c *fiber.Ctx) error {
	sessionID := c.Cookies("session")
	if sessionID == "" {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Status:  403,
			Code:    "NO_SESSION",
			Message: "No valid session",
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	history, total, err := h.letterService.GetHistory(c.Context(), sessionID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(models.PaginatedResponse{
		Data: history,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   limit,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}
```

**Explications:**
- POST /api/letters/generate : génération asynchrone avec validation accès et rate limiting
- GET /api/letters/:id : récupération lettre spécifique avec vérification permissions
- GET /api/letters/:id/pdf : export PDF avec headers appropriés
- GET /api/letters/history : historique paginé

---

### Étape 7: Endpoints Analytics

**Description:** Documenter les endpoints d'analytics

**Code:**

```go
// backend/internal/api/analytics.go

package api

import (
	"github.com/gofiber/fiber/v2"
	"maicivy/internal/models"
)

// GetRealtimeAnalytics récupère les visiteurs en temps réel
// @Summary      Get realtime visitors
// @Description  Retourne le nombre de visiteurs actuels (dernières 5 secondes)
// @Tags         Analytics
// @Produce      json
// @Success      200  {object}  object  "Nombre visiteurs actuels"
// @Failure      500  {object}  models.ErrorResponse
// @Router       /analytics/realtime [get]
// @Example      curl -X GET "http://localhost:5000/api/analytics/realtime" \
//               -H "Content-Type: application/json"
func (h *Handler) GetRealtimeAnalytics(c *fiber.Ctx) error {
	realtimeData, err := h.analyticsService.GetRealtimeVisitors(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(realtimeData)
}

// GetAnalyticsStats récupère les statistiques agrégées
// @Summary      Get analytics statistics
// @Description  Retourne des statistiques agrégées pour une période donnée
// @Tags         Analytics
// @Produce      json
// @Param        period  query    string  false  "Period: day, week, month (default: day)"
// @Success      200     {object}  models.AnalyticsStatsResponse
// @Failure      400     {object}  models.ErrorResponse  "Paramètres invalides"
// @Failure      500     {object}  models.ErrorResponse
// @Router       /analytics/stats [get]
// @Example      curl -X GET "http://localhost:5000/api/analytics/stats?period=week" \
//               -H "Content-Type: application/json"
func (h *Handler) GetAnalyticsStats(c *fiber.Ctx) error {
	period := c.Query("period", "day")

	if period != "day" && period != "week" && period != "month" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "INVALID_PERIOD",
			Message: "Period must be 'day', 'week', or 'month'",
		})
	}

	stats, err := h.analyticsService.GetStats(c.Context(), period)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(stats)
}

// GetThemeAnalytics récupère les thèmes CV les plus consultés
// @Summary      Get top CV themes
// @Description  Retourne les top 5 thèmes CV consultés avec comptage
// @Tags         Analytics
// @Produce      json
// @Success      200  {array}   models.ThemeCount
// @Failure      500  {object}  models.ErrorResponse
// @Router       /analytics/themes [get]
// @Example      curl -X GET "http://localhost:5000/api/analytics/themes" \
//               -H "Content-Type: application/json"
func (h *Handler) GetThemeAnalytics(c *fiber.Ctx) error {
	themes, err := h.analyticsService.GetTopThemes(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(themes)
}

// GetLettersAnalytics récupère stats génération lettres
// @Summary      Get letters generation statistics
// @Description  Retourne le nombre total de lettres générées et statistiques
// @Tags         Analytics
// @Produce      json
// @Success      200  {object}  object  "Statistiques lettres"
// @Failure      500  {object}  models.ErrorResponse
// @Router       /analytics/letters [get]
// @Example      curl -X GET "http://localhost:5000/api/analytics/letters" \
//               -H "Content-Type: application/json"
func (h *Handler) GetLettersAnalytics(c *fiber.Ctx) error {
	lettersData, err := h.analyticsService.GetLettersStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(lettersData)
}

// TrackEvent enregistre un événement custom
// @Summary      Track custom event
// @Description  Enregistre un événement pour analytics (click, page_view, etc.)
// @Tags         Analytics
// @Accept       json
// @Produce      json
// @Param        request  body      models.AnalyticsEventRequest  true  "Event details"
// @Success      200      {object}  object  "Event enregistré"
// @Failure      400      {object}  models.ErrorResponse  "Données invalides"
// @Failure      500      {object}  models.ErrorResponse
// @Router       /analytics/event [post]
// @Example      curl -X POST "http://localhost:5000/api/analytics/event" \
//               -H "Content-Type: application/json" \
//               -d '{"event_type": "page_view", "event_data": {"page": "/cv", "theme": "backend"}}' \
//               -b "session=abc123"
func (h *Handler) TrackEvent(c *fiber.Ctx) error {
	var req models.AnalyticsEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	if req.EventType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Status:  400,
			Code:    "EVENT_TYPE_REQUIRED",
			Message: "Field 'event_type' is required",
		})
	}

	sessionID := c.Cookies("session")
	if sessionID == "" {
		sessionID = "anonymous"
	}

	err := h.analyticsService.RecordEvent(c.Context(), sessionID, req.EventType, req.EventData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Status:  500,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}

	return c.JSON(fiber.Map{"status": "recorded"})
}
```

**Explications:**
- GET /api/analytics/realtime : données temps réel de visiteurs
- GET /api/analytics/stats : statistiques agrégées par période
- GET /api/analytics/themes : thèmes CV top 5
- GET /api/analytics/letters : stats génération lettres
- POST /api/analytics/event : tracking événements custom

---

### Étape 8: Health Check Endpoint

**Description:** Endpoint de vérification de santé

**Code:**

```go
// backend/internal/api/health.go

package api

import (
	"github.com/gofiber/fiber/v2"
)

// GetHealth vérification de santé shallow
// @Summary      Health check (shallow)
// @Description  Vérification basique que le serveur répond
// @Tags         Health
// @Produce      json
// @Success      200  {object}  object  "Service healthy"
// @Router       /health [get]
// @Example      curl -X GET "http://localhost:5000/health"
func (h *Handler) GetHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"service": "maicivy-api",
		"version": "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetDeepHealth vérification de santé approfondie
// @Summary      Health check (deep)
// @Description  Vérification complète: serveur, DB, Redis
// @Tags         Health
// @Produce      json
// @Success      200  {object}  object  "All services healthy"
// @Failure      503  {object}  object  "Service unavailable"
// @Router       /health/deep [get]
// @Example      curl -X GET "http://localhost:5000/health/deep"
func (h *Handler) GetDeepHealth(c *fiber.Ctx) error {
	dbHealth := h.db.Ping(c.Context()) == nil
	redisHealth := h.redis.Ping(c.Context()).Err() == nil

	if !dbHealth || !redisHealth {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "degraded",
			"database": dbHealth,
			"redis": redisHealth,
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"database": true,
		"redis": true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
```

---

### Étape 9: Setup Swagger UI

**Description:** Intégrer Swagger UI dans l'application

**Code:**

```go
// backend/cmd/main.go - Ajouter imports

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	_ "maicivy/docs" // Import docs générés
)

func main() {
	app := fiber.New()

	// Routes
	app.Get("/health", handlers.GetHealth)
	app.Get("/health/deep", handlers.GetDeepHealth)

	// API v1
	api := app.Group("/api")
	api.Get("/cv", handlers.GetCV)
	api.Get("/cv/themes", handlers.GetCVThemes)
	api.Get("/cv/export", handlers.ExportCV)
	api.Get("/experiences", handlers.GetExperiences)
	api.Get("/skills", handlers.GetSkills)
	api.Get("/projects", handlers.GetProjects)

	api.Post("/letters/generate", handlers.GenerateLetter)
	api.Get("/letters/:id", handlers.GetLetter)
	api.Get("/letters/:id/pdf", handlers.GetLetterPDF)
	api.Get("/letters/history", handlers.GetLetterHistory)

	api.Get("/analytics/realtime", handlers.GetRealtimeAnalytics)
	api.Get("/analytics/stats", handlers.GetAnalyticsStats)
	api.Get("/analytics/themes", handlers.GetThemeAnalytics)
	api.Get("/analytics/letters", handlers.GetLettersAnalytics)
	api.Post("/analytics/event", handlers.TrackEvent)

	// Swagger UI
	app.Get("/api/docs/*", swagger.HandlerDefault)
	app.Get("/api/docs/swagger.json", swagger.FileHandlerDefault)

	app.Listen(":5000")
}
```

---

### Étape 10: Générer et Servir la Documentation

**Description:** Commandes pour générer et tester la documentation

**Code:**

```bash
# Générer la documentation OpenAPI
cd backend
swag init -g cmd/main.go

# Résultat:
# ✅ Generated docs/docs.go
# ✅ Generated docs/swagger.yaml
# ✅ Generated docs/swagger.json

# Compiler et lancer le serveur
go run cmd/main.go

# Tester:
# - Swagger UI : http://localhost:5000/api/docs/
# - OpenAPI spec: http://localhost:5000/api/docs/swagger.json
# - Santé: http://localhost:5000/health
```

**Explications:**
- swag scanne les commentaires Go et génère les fichiers
- Les fichiers sont versionés dans git
- Swagger UI offre UI interactive + test endpoints
- Régénérer après chaque modification endpoint

---

## 🧪 Tests

### Tests Unitaires

```go
// backend/internal/api/cv_test.go

package api

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGetCV(t *testing.T) {
	tests := []struct {
		name      string
		theme     string
		wantCode  int
		wantError bool
	}{
		{"valid backend", "backend", 200, false},
		{"valid frontend", "frontend", 200, false},
		{"empty theme", "", 400, true},
		{"invalid theme", "xyz", 400, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}

func TestExportCV(t *testing.T) {
	tests := []struct {
		name         string
		theme        string
		format       string
		wantCode     int
		wantMimeType string
	}{
		{"pdf export", "backend", "pdf", 200, "application/pdf"},
		{"missing format", "backend", "", 400, ""},
		{"invalid format", "backend", "doc", 400, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test implementation
		})
	}
}
```

### Tests d'Integration API

```go
// backend/e2e/api_test.go

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCVEndpoints(t *testing.T) {
	client := http.DefaultClient

	t.Run("GET /api/cv?theme=backend", func(t *testing.T) {
		resp, err := client.Get("http://localhost:5000/api/cv?theme=backend")
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "backend", result["theme"])
	})

	t.Run("GET /api/cv?theme=invalid", func(t *testing.T) {
		resp, err := client.Get("http://localhost:5000/api/cv?theme=invalid")
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestLettersEndpoints(t *testing.T) {
	client := http.DefaultClient

	t.Run("POST /api/letters/generate", func(t *testing.T) {
		payload := map[string]string{"company_name": "Google"}
		body, _ := json.Marshal(payload)

		resp, err := client.Post(
			"http://localhost:5000/api/letters/generate",
			"application/json",
			bytes.NewReader(body),
		)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
```

### Commandes Test

```bash
# Tests unitaires
go test -v ./internal/api/...

# Coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Tests avec race detector
go test -race ./...

# E2E (après lancer serveur)
go test -v ./e2e/...
```

---

## ⚠️ Points d'Attention

- **⚠️ Synchronisation automatique:** Regénérer les docs après CHAQUE modification endpoint (ajouter à CI/CD)
- **⚠️ Versioning API:** Si breaking changes, versioner les endpoints (/api/v2/...)
- **⚠️ Sécurité:** Ne pas exposer Swagger UI en production sans authentification (optionnel)
- **⚠️ CORS:** Vérifier que Swagger UI peut accéder à l'API (CORS configuré)
- **⚠️ Exemples à jour:** Vérifier que les exemples curl restent valides
- **💡 Documentation progressive:** Documenter les nouveaux endpoints PENDANT le développement, pas après
- **💡 Schemas réutilisables:** Utiliser les mêmes DTOs pour req/res et documentations
- **💡 Consistence:** Tous les endpoints doivent suivre le même pattern d'erreur/réponse

---

## 📚 Ressources

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI 3.0 Specification](https://spec.openapis.org/oas/v3.0.0)
- [Fiber Documentation](https://docs.gofiber.io)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)

---

## ✅ Checklist de Complétion

- [ ] Swaggo installé et configuré
- [ ] Annotations documentées sur tous les endpoints
- [ ] DTOs request/response créés et documentés
- [ ] Swagger UI déployée et accessible
- [ ] Exemples curl validés
- [ ] Codes erreur standardisés et documentés
- [ ] Tests unitaires écrits et passants
- [ ] Tests E2E validant endpoints
- [ ] Documentation auto-generation en CI/CD
- [ ] Swagger disponible à `/api/docs`
- [ ] Review sécurité (pas d'infos sensibles exposées)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
