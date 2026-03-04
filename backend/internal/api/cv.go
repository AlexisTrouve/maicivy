package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// CVHandler gère les endpoints liés au CV
type CVHandler struct {
	cvService services.CVServiceInterface
	tailoring *services.TailoringService // nil si AI non configurée
}

// NewCVHandler crée un nouveau handler
func NewCVHandler(cvService services.CVServiceInterface, tailoring *services.TailoringService) *CVHandler {
	return &CVHandler{
		cvService: cvService,
		tailoring: tailoring,
	}
}

// RegisterRoutes enregistre les routes CV
func (h *CVHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	api.Get("/cv", h.GetAdaptiveCV)
	api.Get("/cv/list", h.ListCVs)
	api.Get("/cv/themes", h.GetThemes)
	api.Get("/experiences", h.GetExperiences)
	api.Get("/skills", h.GetSkills)
	api.Get("/projects", h.GetProjects)
	api.Get("/cv/export", h.ExportPDF)
	api.Post("/cv/tailor", h.TailorCV) // CV personnalisé par annonce
}

// GetAdaptiveCV retourne le CV adapté au thème et à la langue
// @Summary Get adaptive CV
// @Description Returns CV adapted to specified theme and language
// @Tags CV
// @Param theme query string false "Theme ID (backend, cpp, artistique, fullstack, devops)"
// @Param lang query string false "Language (fr, en)" default(fr)
// @Success 200 {object} services.AdaptiveCVResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/cv [get]
func (h *CVHandler) GetAdaptiveCV(c *fiber.Ctx) error {
	themeID := c.Query("theme", "fullstack") // Default: fullstack
	lang := c.Query("lang", "fr")            // Default: fr

	cv, err := h.cvService.GetAdaptiveCV(c.Context(), themeID, lang)
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
// @Router /api/v1/cv/themes [get]
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
// @Router /api/v1/experiences [get]
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
// @Router /api/v1/skills [get]
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
// @Router /api/v1/projects [get]
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
// @Description Generates and downloads CV as PDF for specified theme and language
// @Tags CV
// @Param theme query string false "Theme ID"
// @Param format query string false "Export format (pdf)" default(pdf)
// @Param lang query string false "Language (fr, en)" default(fr)
// @Success 200 {file} application/pdf
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/cv/export [get]
func (h *CVHandler) ExportPDF(c *fiber.Ctx) error {
	themeID := c.Query("theme", "fullstack")
	format := c.Query("format", "pdf")
	lang := c.Query("lang", "fr")

	if format != "pdf" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only PDF format is supported",
		})
	}

	// Valider la langue
	if lang != "fr" && lang != "en" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Supported languages: fr, en",
		})
	}

	// Récupérer CV adaptatif avec la langue spécifiée
	cv, err := h.cvService.GetAdaptiveCV(c.Context(), themeID, lang)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Générer PDF avec la langue spécifiée
	pdfService := services.NewPDFService()
	pdfBytes, err := pdfService.GenerateCVPDF(cv, lang)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate PDF",
		})
	}

	// Retourner PDF
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=cv_"+themeID+"_"+lang+".pdf")
	return c.Send(pdfBytes)
}

// ListCVs retourne tous les CVs disponibles avec leurs URLs
func (h *CVHandler) ListCVs(c *fiber.Ctx) error {
	themes := h.cvService.GetAvailableThemes()
	langs := []string{"fr", "en"}

	type CVEntry struct {
		Theme       string `json:"theme"`
		ThemeName   string `json:"themeName"`
		Lang        string `json:"lang"`
		JSONURL     string `json:"jsonUrl"`
		PDFURL      string `json:"pdfUrl"`
	}

	base := c.BaseURL()
	entries := make([]CVEntry, 0, len(themes)*len(langs))
	for _, theme := range themes {
		for _, lang := range langs {
			entries = append(entries, CVEntry{
				Theme:     theme.ID,
				ThemeName: theme.Name,
				Lang:      lang,
				JSONURL:   base + "/api/v1/cv?theme=" + theme.ID + "&lang=" + lang,
				PDFURL:    base + "/api/v1/cv/export?theme=" + theme.ID + "&lang=" + lang,
			})
		}
	}

	return c.JSON(fiber.Map{
		"count": len(entries),
		"cvs":   entries,
	})
}

// TailorCV génère un CV PDF personnalisé pour une annonce spécifique.
// indeed-outreach envoie le job + les skills matchés → maicivy choisit le thème,
// réécrit les expériences via Haiku et injecte une couche stealth ATS.
//
// @Summary Tailor CV for a job posting
// @Tags CV
// @Accept json
// @Produce application/pdf
// @Param body body services.TailorRequest true "Job details and matched skills"
// @Success 200 {file} application/pdf
// @Failure 400 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/v1/cv/tailor [post]
func (h *CVHandler) TailorCV(c *fiber.Ctx) error {
	if h.tailoring == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "CV tailoring not available — AI service not configured",
		})
	}

	var req services.TailorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	pdfBytes, err := h.tailoring.TailorAndExport(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=cv_tailored.pdf")
	return c.Send(pdfBytes)
}

// ErrorResponse structure pour documentation API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
