package api

import (
	"fmt"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// TimelineEvent représente un événement dans la timeline (experience ou project)
type TimelineEvent struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // "experience" ou "project"
	Title     string     `json:"title"`
	Subtitle  string     `json:"subtitle"`
	Content   string     `json:"content"`
	StartDate time.Time  `json:"startDate"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Tags      []string   `json:"tags"`
	Category  string     `json:"category"` // "backend", "frontend", "fullstack", etc.
	Image     string     `json:"image,omitempty"`
	IsCurrent bool       `json:"isCurrent"` // Pour emploi actuel ou projet en cours
}

// TimelineHandler gère les endpoints timeline
type TimelineHandler struct {
	db *gorm.DB
}

// NewTimelineHandler crée une nouvelle instance
func NewTimelineHandler(db *gorm.DB) *TimelineHandler {
	return &TimelineHandler{
		db: db,
	}
}

// GetTimeline retourne tous les événements chronologiques
// GET /api/v1/timeline
// Query params:
// - category: filtrer par catégorie (backend, frontend, fullstack, devops, etc.)
// - from: date de début (YYYY-MM-DD)
// - to: date de fin (YYYY-MM-DD)
func (h *TimelineHandler) GetTimeline(c *fiber.Ctx) error {
	category := c.Query("category", "")
	from := c.Query("from", "")
	to := c.Query("to", "")

	// Récupérer expériences
	var experiences []models.Experience
	expQuery := h.db.Model(&models.Experience{})

	if category != "" {
		expQuery = expQuery.Where("category = ?", category)
	}

	// Filtrage par date
	if from != "" {
		fromDate, err := time.Parse("2006-01-02", from)
		if err == nil {
			expQuery = expQuery.Where("start_date >= ?", fromDate)
		}
	}

	if to != "" {
		toDate, err := time.Parse("2006-01-02", to)
		if err == nil {
			expQuery = expQuery.Where("start_date <= ?", toDate)
		}
	}

	expQuery.Order("start_date DESC").Find(&experiences)

	// Récupérer projets
	var projects []models.Project
	projQuery := h.db.Model(&models.Project{})

	if category != "" {
		projQuery = projQuery.Where("category = ?", category)
	}

	projQuery.Order("created_at DESC").Find(&projects)

	// Combiner et convertir en TimelineEvent
	var events []TimelineEvent

	// Ajouter expériences
	for _, exp := range experiences {
		events = append(events, TimelineEvent{
			ID:        fmt.Sprintf("exp_%d", exp.ID),
			Type:      "experience",
			Title:     exp.Title,
			Subtitle:  exp.Company,
			Content:   exp.Description,
			StartDate: exp.StartDate,
			EndDate:   exp.EndDate,
			Tags:      exp.Technologies,
			Category:  exp.Category,
			IsCurrent: exp.IsCurrentJob(),
		})
	}

	// Ajouter projets
	for _, proj := range projects {
		// Pour les projets, on utilise created_at comme start_date
		// et updated_at comme end_date si le projet n'est pas en cours
		var endDate *time.Time
		if !proj.InProgress && proj.UpdatedAt.After(proj.CreatedAt) {
			endDate = &proj.UpdatedAt
		}

		events = append(events, TimelineEvent{
			ID:        fmt.Sprintf("proj_%d", proj.ID),
			Type:      "project",
			Title:     proj.Title,
			Subtitle:  proj.Description,
			Content:   proj.Description,
			StartDate: proj.CreatedAt,
			EndDate:   endDate,
			Tags:      proj.Technologies,
			Category:  proj.Category,
			Image:     proj.ImageURL,
			IsCurrent: proj.InProgress,
		})
	}

	// Trier par date décroissante (plus récent en premier)
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartDate.After(events[j].StartDate)
	})

	// Calculer quelques stats
	stats := h.calculateTimelineStats(events)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"events": events,
			"total":  len(events),
			"stats":  stats,
		},
	})
}

// GetCategories liste toutes les catégories disponibles
// GET /api/v1/timeline/categories
func (h *TimelineHandler) GetCategories(c *fiber.Ctx) error {
	var expCategories []string
	h.db.Model(&models.Experience{}).
		Distinct("category").
		Pluck("category", &expCategories)

	var projCategories []string
	h.db.Model(&models.Project{}).
		Distinct("category").
		Pluck("category", &projCategories)

	// Combiner et dédupliquer
	categoryMap := make(map[string]bool)
	for _, cat := range expCategories {
		if cat != "" {
			categoryMap[cat] = true
		}
	}
	for _, cat := range projCategories {
		if cat != "" {
			categoryMap[cat] = true
		}
	}

	// Convertir en slice
	var categories []string
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	sort.Strings(categories)

	return c.JSON(fiber.Map{
		"success":    true,
		"categories": categories,
		"total":      len(categories),
	})
}

// GetMilestones retourne les milestones importants
// GET /api/v1/timeline/milestones
func (h *TimelineHandler) GetMilestones(c *fiber.Ctx) error {
	milestones := h.generateMilestones()

	return c.JSON(fiber.Map{
		"success":    true,
		"milestones": milestones,
		"total":      len(milestones),
	})
}

// TimelineMilestone représente un événement important
type TimelineMilestone struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Icon        string    `json:"icon"`
	Type        string    `json:"type"` // "achievement", "career", "education", "project"
}

// generateMilestones génère automatiquement des milestones
func (h *TimelineHandler) generateMilestones() []TimelineMilestone {
	milestones := []TimelineMilestone{}

	// Milestone: Première expérience
	var firstExp models.Experience
	if err := h.db.Order("start_date ASC").First(&firstExp).Error; err == nil {
		milestones = append(milestones, TimelineMilestone{
			ID:          fmt.Sprintf("milestone_first_job"),
			Title:       "Première expérience professionnelle",
			Description: fmt.Sprintf("%s chez %s", firstExp.Title, firstExp.Company),
			Date:        firstExp.StartDate,
			Icon:        "🎯",
			Type:        "career",
		})
	}

	// Milestone: Première expérience backend
	var firstBackend models.Experience
	if err := h.db.Where("category = ?", "backend").Order("start_date ASC").First(&firstBackend).Error; err == nil {
		milestones = append(milestones, TimelineMilestone{
			ID:          fmt.Sprintf("milestone_first_backend"),
			Title:       "Première expérience Backend",
			Description: fmt.Sprintf("%s - %s", firstBackend.Company, firstBackend.StartDate.Format("2006")),
			Date:        firstBackend.StartDate,
			Icon:        "💻",
			Type:        "career",
		})
	}

	// Milestone: Nombre total de projets
	var projectCount int64
	h.db.Model(&models.Project{}).Count(&projectCount)
	if projectCount > 0 {
		var latestProject models.Project
		h.db.Order("created_at DESC").First(&latestProject)

		milestones = append(milestones, TimelineMilestone{
			ID:          fmt.Sprintf("milestone_projects"),
			Title:       fmt.Sprintf("%d projets réalisés", projectCount),
			Description: fmt.Sprintf("Dernier projet: %s", latestProject.Title),
			Date:        latestProject.CreatedAt,
			Icon:        "🚀",
			Type:        "achievement",
		})
	}

	// Trier par date
	sort.Slice(milestones, func(i, j int) bool {
		return milestones[i].Date.Before(milestones[j].Date)
	})

	return milestones
}

// TimelineStats contient des statistiques sur la timeline
type TimelineStats struct {
	TotalExperiences    int               `json:"totalExperiences"`
	TotalProjects       int               `json:"totalProjects"`
	CategoriesBreakdown map[string]int    `json:"categoriesBreakdown"`
	YearsOfExperience   float64           `json:"yearsOfExperience"`
	TopTechnologies     []TechnologyCount `json:"topTechnologies"`
}

// TechnologyCount compte les occurrences d'une technologie
type TechnologyCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// calculateTimelineStats calcule les statistiques de la timeline
func (h *TimelineHandler) calculateTimelineStats(events []TimelineEvent) TimelineStats {
	stats := TimelineStats{
		CategoriesBreakdown: make(map[string]int),
		TopTechnologies:     []TechnologyCount{},
	}

	techCount := make(map[string]int)
	var totalDuration time.Duration

	for _, event := range events {
		if event.Type == "experience" {
			stats.TotalExperiences++

			// Calculer durée
			endDate := time.Now()
			if event.EndDate != nil {
				endDate = *event.EndDate
			}
			totalDuration += endDate.Sub(event.StartDate)
		} else {
			stats.TotalProjects++
		}

		// Compter catégories
		stats.CategoriesBreakdown[event.Category]++

		// Compter technologies
		for _, tech := range event.Tags {
			techCount[tech]++
		}
	}

	// Calculer années d'expérience
	stats.YearsOfExperience = totalDuration.Hours() / 24 / 365.25

	// Top 10 technologies
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range techCount {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// Limiter à 10
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		stats.TopTechnologies = append(stats.TopTechnologies, TechnologyCount{
			Name:  sorted[i].Key,
			Count: sorted[i].Value,
		})
	}

	return stats
}
