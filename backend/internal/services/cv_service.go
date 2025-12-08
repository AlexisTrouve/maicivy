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
	db             *gorm.DB
	redis          *redis.Client
	scoringService *CVScoringService
}

// NewCVService crée une nouvelle instance
func NewCVService(db *gorm.DB, redisClient *redis.Client) *CVService {
	return &CVService{
		db:             db,
		redis:          redisClient,
		scoringService: NewCVScoringService(),
	}
}

// AdaptiveCVResponse représente la réponse complète du CV adaptatif
type AdaptiveCVResponse struct {
	Theme       config.CVTheme      `json:"theme"`
	Experiences []models.Experience `json:"experiences"`
	Skills      []models.Skill      `json:"skills"`
	Projects    []models.Project    `json:"projects"`
	GeneratedAt time.Time           `json:"generated_at"`
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
