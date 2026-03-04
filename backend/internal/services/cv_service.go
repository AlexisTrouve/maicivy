package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"maicivy/internal/config"
	"maicivy/internal/content"
	"maicivy/internal/models"
)

// maxDetailedProjects est le nombre max de projets affichés en détail dans le CV/PDF
const maxDetailedProjects = 6

// CVService gère la logique métier du CV
type CVService struct {
	contentLoader  *content.Loader
	redis          *redis.Client
	scoringService *CVScoringService
	l10nHelper     *LocalizationHelper
	llmScoring     *LLMScoringService // nil si non configuré
}

// NewCVService crée une nouvelle instance
func NewCVService(contentLoader *content.Loader, redisClient *redis.Client, llmScoring *LLMScoringService) *CVService {
	return &CVService{
		contentLoader:  contentLoader,
		redis:          redisClient,
		scoringService: NewCVScoringService(),
		l10nHelper:     NewLocalizationHelper(),
		llmScoring:     llmScoring,
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
	Summary     string                     `json:"summary,omitempty"` // résumé dynamique généré par LLM (vide pour thèmes statiques)
	Experiences []ScoredExperienceResponse `json:"experiences"`
	Skills      []ScoredSkillResponse      `json:"skills"`
	Projects    []ScoredProjectResponse    `json:"projects"`
	GeneratedAt time.Time                  `json:"generatedAt"`
}

// GetAdaptiveCV retourne le CV adapté au thème et à la langue demandés
func (s *CVService) GetAdaptiveCV(ctx context.Context, themeID string, lang string) (*AdaptiveCVResponse, error) {
	// 0. Normaliser la langue
	lang = s.l10nHelper.NormalizeLanguage(lang)

	// 1. Vérifier si thème existe
	theme := config.GetTheme(themeID)
	if theme == nil {
		return nil, fmt.Errorf("theme not found: %s", themeID)
	}

	// 2. Vérifier cache Redis (inclure langue dans la clé)
	cacheKey := fmt.Sprintf("cv:theme:%s:lang:%s", themeID, lang)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		// Cache hit
		var response AdaptiveCVResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return &response, nil
		}
	}

	// 3. Cache miss - récupérer depuis le content loader
	experiences := s.contentLoader.GetExperiences()
	skills := s.contentLoader.GetSkills()
	projects := s.contentLoader.GetProjects()

	// 4. Scorer et filtrer selon thème
	scoredExp := s.scoringService.ScoreExperiences(experiences, theme)
	scoredSkills := s.scoringService.ScoreSkills(skills, theme)
	scoredProjects := s.scoringService.ScoreProjects(projects, theme)

	// 5. Localiser et extraire les items AVEC scores pour la réponse
	filteredExperiences := make([]ScoredExperienceResponse, 0, len(scoredExp))
	for _, se := range scoredExp {
		localizedExp := s.l10nHelper.LocalizeExperience(se.Experience, lang)
		filteredExperiences = append(filteredExperiences, ScoredExperienceResponse{
			Experience: localizedExp,
			Score:      se.Score,
		})
	}

	// Sort experiences: score DESC, puis date DESC en tiebreaker
	sort.Slice(filteredExperiences, func(i, j int) bool {
		if filteredExperiences[i].Score != filteredExperiences[j].Score {
			return filteredExperiences[i].Score > filteredExperiences[j].Score
		}
		return filteredExperiences[i].StartDate.After(filteredExperiences[j].StartDate)
	})

	filteredSkills := make([]ScoredSkillResponse, 0, len(scoredSkills))
	for _, ss := range scoredSkills {
		localizedSkill := s.l10nHelper.LocalizeSkill(ss.Skill, lang)
		filteredSkills = append(filteredSkills, ScoredSkillResponse{
			Skill: localizedSkill,
			Score: ss.Score,
		})
	}

	filteredProjects := make([]ScoredProjectResponse, 0, len(scoredProjects))
	for _, sp := range scoredProjects {
		localizedProject := s.l10nHelper.LocalizeProject(sp.Project, lang)
		filteredProjects = append(filteredProjects, ScoredProjectResponse{
			Project: localizedProject,
			Score:   sp.Score,
		})
	}

	// Remplacer les scores projets par les scores LLM si disponibles (1 seul call, cache 6h)
	if s.llmScoring != nil {
		if llmScores, err := s.llmScoring.ScoreProjectsForTheme(ctx, projects, theme); err == nil {
			for i := range filteredProjects {
				slug := toSlug(filteredProjects[i].Title)
				if score, ok := llmScores[slug]; ok {
					// Normaliser 1-100 → 0.0-1.0
					filteredProjects[i].Score = float64(score) / 100.0
				}
			}
		}
		// Erreur LLM → on garde les scores tag-weight (fallback silencieux)
	}

	// Trier les projets par score LLM (ou tag-weight si LLM indispo)
	sort.Slice(filteredProjects, func(i, j int) bool {
		return filteredProjects[i].Score > filteredProjects[j].Score
	})

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
	return s.contentLoader.GetExperiences(), nil
}

// GetAllSkills retourne toutes les compétences
func (s *CVService) GetAllSkills(ctx context.Context) ([]models.Skill, error) {
	return s.contentLoader.GetSkills(), nil
}

// GetAllProjects retourne tous les projets
func (s *CVService) GetAllProjects(ctx context.Context) ([]models.Project, error) {
	return s.contentLoader.GetProjects(), nil
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
