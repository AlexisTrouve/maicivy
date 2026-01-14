package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// ScoredExperienceResponse représente une expérience avec son score pour la réponse JSON
type ScoredExperienceResponse struct {
	models.Experience
	Score float64 `json:"score"`
}

// ScoredSkillResponse représente une compétence avec son score pour la réponse JSON
type ScoredSkillResponse struct {
	models.Skill
	Score float64 `json:"score"`
}

// ScoredProjectResponse représente un projet avec son score pour la réponse JSON
type ScoredProjectResponse struct {
	models.Project
	Score float64 `json:"score"`
}

// AdaptiveCVResponse représente la réponse complète du CV adaptatif
type AdaptiveCVResponse struct {
	Theme       config.CVTheme             `json:"theme"`
	Experiences []ScoredExperienceResponse `json:"experiences"`
	Skills      []ScoredSkillResponse      `json:"skills"`
	Projects    []ScoredProjectResponse    `json:"projects"`
	GeneratedAt time.Time                  `json:"generatedAt"`
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

	if err := s.db.Order("start_date DESC").Find(&experiences).Error; err != nil {
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

	// 5. Extraire les items AVEC scores pour la réponse
	filteredExperiences := make([]ScoredExperienceResponse, 0, len(scoredExp))
	for _, se := range scoredExp {
		filteredExperiences = append(filteredExperiences, ScoredExperienceResponse{
			Experience: se.Experience,
			Score:      se.Score,
		})
	}

	// Sort experiences by start date (most recent first)
	sort.Slice(filteredExperiences, func(i, j int) bool {
		return filteredExperiences[i].StartDate.After(filteredExperiences[j].StartDate)
	})

	filteredSkills := make([]ScoredSkillResponse, 0, len(scoredSkills))
	for _, ss := range scoredSkills {
		filteredSkills = append(filteredSkills, ScoredSkillResponse{
			Skill: ss.Skill,
			Score: ss.Score,
		})
	}

	filteredProjects := make([]ScoredProjectResponse, 0, len(scoredProjects))
	for _, sp := range scoredProjects {
		filteredProjects = append(filteredProjects, ScoredProjectResponse{
			Project: sp.Project,
			Score:   sp.Score,
		})
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
