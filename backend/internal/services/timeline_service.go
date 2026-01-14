package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// TimelineService gère la logique métier de la timeline
type TimelineService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewTimelineService crée une nouvelle instance du service
func NewTimelineService(db *gorm.DB, redis *redis.Client) *TimelineService {
	return &TimelineService{
		db:    db,
		redis: redis,
	}
}

// TimelineEventDTO représente un événement de timeline
type TimelineEventDTO struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Subtitle  string     `json:"subtitle"`
	Content   string     `json:"content"`
	StartDate time.Time  `json:"startDate"`
	EndDate   *time.Time `json:"endDate,omitempty"`
	Tags      []string   `json:"tags"`
	Category  string     `json:"category"`
	Image     string     `json:"image,omitempty"`
	IsCurrent bool       `json:"isCurrent"`
	Duration  string     `json:"duration"` // Format: "2 years 3 months"
}

// TimelineFilters représente les filtres de recherche
type TimelineFilters struct {
	Category string
	FromDate *time.Time
	ToDate   *time.Time
	Type     string // "experience", "project", ou vide pour tous
}

// GetTimeline récupère et agrège les données chronologiques
func (s *TimelineService) GetTimeline(ctx context.Context, filters TimelineFilters) ([]TimelineEventDTO, error) {
	// Essayer le cache d'abord
	cacheKey := s.buildCacheKey(filters)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var events []TimelineEventDTO
		if err := json.Unmarshal([]byte(cached), &events); err == nil {
			return events, nil
		}
	}

	// Récupérer depuis la base de données
	events, err := s.fetchTimelineFromDB(filters)
	if err != nil {
		return nil, err
	}

	// Mettre en cache (TTL 1h)
	data, _ := json.Marshal(events)
	s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)

	return events, nil
}

// fetchTimelineFromDB récupère les données depuis PostgreSQL
func (s *TimelineService) fetchTimelineFromDB(filters TimelineFilters) ([]TimelineEventDTO, error) {
	var events []TimelineEventDTO

	// Récupérer expériences si nécessaire
	if filters.Type == "" || filters.Type == "experience" {
		var experiences []models.Experience
		query := s.db.Model(&models.Experience{})

		// Appliquer filtres
		if filters.Category != "" {
			query = query.Where("category = ?", filters.Category)
		}
		if filters.FromDate != nil {
			query = query.Where("start_date >= ?", *filters.FromDate)
		}
		if filters.ToDate != nil {
			query = query.Where("start_date <= ?", *filters.ToDate)
		}

		query.Order("start_date DESC").Find(&experiences)

		// Convertir en DTO
		for _, exp := range experiences {
			events = append(events, TimelineEventDTO{
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
				Duration:  s.formatDuration(exp.StartDate, exp.EndDate),
			})
		}
	}

	// Récupérer projets si nécessaire
	if filters.Type == "" || filters.Type == "project" {
		var projects []models.Project
		query := s.db.Model(&models.Project{})

		// Appliquer filtres
		if filters.Category != "" {
			query = query.Where("category = ?", filters.Category)
		}
		if filters.FromDate != nil {
			query = query.Where("created_at >= ?", *filters.FromDate)
		}
		if filters.ToDate != nil {
			query = query.Where("created_at <= ?", *filters.ToDate)
		}

		query.Order("created_at DESC").Find(&projects)

		// Convertir en DTO
		for _, proj := range projects {
			var endDate *time.Time
			if !proj.InProgress && proj.UpdatedAt.After(proj.CreatedAt) {
				endDate = &proj.UpdatedAt
			}

			events = append(events, TimelineEventDTO{
				ID:        fmt.Sprintf("proj_%d", proj.ID),
				Type:      "project",
				Title:     proj.Title,
				Subtitle:  fmt.Sprintf("Projet %s", proj.Category),
				Content:   proj.Description,
				StartDate: proj.CreatedAt,
				EndDate:   endDate,
				Tags:      proj.Technologies,
				Category:  proj.Category,
				Image:     proj.ImageURL,
				IsCurrent: proj.InProgress,
				Duration:  s.formatDuration(proj.CreatedAt, endDate),
			})
		}
	}

	// Trier par date décroissante
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartDate.After(events[j].StartDate)
	})

	return events, nil
}

// formatDuration convertit une durée en format lisible
func (s *TimelineService) formatDuration(startDate time.Time, endDate *time.Time) string {
	end := time.Now()
	if endDate != nil {
		end = *endDate
	}

	duration := end.Sub(startDate)
	years := int(duration.Hours() / 24 / 365.25)
	months := int(duration.Hours()/24/30.44) % 12

	if years == 0 && months == 0 {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 jour"
		}
		return fmt.Sprintf("%d jours", days)
	}

	var result string
	if years > 0 {
		if years == 1 {
			result = "1 an"
		} else {
			result = fmt.Sprintf("%d ans", years)
		}
	}

	if months > 0 {
		if result != "" {
			result += " "
		}
		if months == 1 {
			result += "1 mois"
		} else {
			result += fmt.Sprintf("%d mois", months)
		}
	}

	return result
}

// buildCacheKey génère une clé de cache unique basée sur les filtres
func (s *TimelineService) buildCacheKey(filters TimelineFilters) string {
	key := "timeline"

	if filters.Category != "" {
		key += ":cat:" + filters.Category
	}
	if filters.Type != "" {
		key += ":type:" + filters.Type
	}
	if filters.FromDate != nil {
		key += ":from:" + filters.FromDate.Format("2006-01-02")
	}
	if filters.ToDate != nil {
		key += ":to:" + filters.ToDate.Format("2006-01-02")
	}

	return key
}

// CalculateOverlaps détecte les périodes où plusieurs expériences/projets se chevauchent
func (s *TimelineService) CalculateOverlaps(ctx context.Context) ([]TimelineOverlap, error) {
	events, err := s.GetTimeline(ctx, TimelineFilters{})
	if err != nil {
		return nil, err
	}

	var overlaps []TimelineOverlap

	// Comparer chaque paire d'événements
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if overlap := s.checkOverlap(events[i], events[j]); overlap != nil {
				overlaps = append(overlaps, *overlap)
			}
		}
	}

	return overlaps, nil
}

// TimelineOverlap représente un chevauchement entre deux événements
type TimelineOverlap struct {
	Event1    TimelineEventDTO `json:"event1"`
	Event2    TimelineEventDTO `json:"event2"`
	StartDate time.Time        `json:"startDate"`
	EndDate   time.Time        `json:"endDate"`
	Duration  string           `json:"duration"`
}

// checkOverlap vérifie si deux événements se chevauchent
func (s *TimelineService) checkOverlap(e1, e2 TimelineEventDTO) *TimelineOverlap {
	// Déterminer les dates de fin effectives
	end1 := time.Now()
	if e1.EndDate != nil {
		end1 = *e1.EndDate
	}

	end2 := time.Now()
	if e2.EndDate != nil {
		end2 = *e2.EndDate
	}

	// Vérifier chevauchement
	overlapStart := e1.StartDate
	if e2.StartDate.After(e1.StartDate) {
		overlapStart = e2.StartDate
	}

	overlapEnd := end1
	if end2.Before(end1) {
		overlapEnd = end2
	}

	// Si overlap_start > overlap_end, pas de chevauchement
	if overlapStart.After(overlapEnd) {
		return nil
	}

	return &TimelineOverlap{
		Event1:    e1,
		Event2:    e2,
		StartDate: overlapStart,
		EndDate:   overlapEnd,
		Duration:  s.formatDuration(overlapStart, &overlapEnd),
	}
}

// GetYearlyBreakdown retourne un breakdown par année
func (s *TimelineService) GetYearlyBreakdown(ctx context.Context) (map[int]YearStats, error) {
	events, err := s.GetTimeline(ctx, TimelineFilters{})
	if err != nil {
		return nil, err
	}

	breakdown := make(map[int]YearStats)

	for _, event := range events {
		year := event.StartDate.Year()

		stats, exists := breakdown[year]
		if !exists {
			stats = YearStats{
				Year:   year,
				Events: []TimelineEventDTO{},
			}
		}

		stats.Events = append(stats.Events, event)

		if event.Type == "experience" {
			stats.ExperiencesCount++
		} else {
			stats.ProjectsCount++
		}

		breakdown[year] = stats
	}

	return breakdown, nil
}

// YearStats représente les stats pour une année donnée
type YearStats struct {
	Year             int                `json:"year"`
	ExperiencesCount int                `json:"experiencesCount"`
	ProjectsCount    int                `json:"projectsCount"`
	Events           []TimelineEventDTO `json:"events"`
}

// InvalidateCache invalide le cache de la timeline
func (s *TimelineService) InvalidateCache(ctx context.Context) error {
	// Supprimer toutes les clés commençant par "timeline"
	iter := s.redis.Scan(ctx, 0, "timeline*", 0).Iterator()
	for iter.Next(ctx) {
		s.redis.Del(ctx, iter.Val())
	}
	return iter.Err()
}
