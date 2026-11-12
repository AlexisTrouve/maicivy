package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/rs/zerolog/log"
)

// maxDetailedProjects est le nombre max de projets affichés en détail dans le CV/PDF
const maxDetailedProjects = 6

// CVService gère la logique métier du CV
type CVService struct {
	contentLoader  ContentProvider // source des données CV (maiprofiles)
	redis          *redis.Client
	scoringService *CVScoringService
	l10nHelper     *LocalizationHelper
	llmScoring     *LLMScoringService // nil si non configuré
}

// NewCVService crée une nouvelle instance
func NewCVService(contentLoader ContentProvider, redisClient *redis.Client, llmScoring *LLMScoringService) *CVService {
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
	Summary     string                     `json:"summary,omitempty"`  // résumé dynamique généré par LLM
	Location    string                     `json:"location,omitempty"` // ville extraite de l'offre (stealth ATS)
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
			// Le thème vient TOUJOURS de la config, jamais du cache : NameI18n porte `json:"-"`
			// et disparaissait au décodage → le CV chinois retombait sur le titre anglais pendant
			// les 5 min du TTL. Ça garantit aussi qu'un thème modifié s'applique dès le déploiement.
			response.Theme = *theme
			return &response, nil
		}
	}

	// 3. Cache miss - récupérer depuis le content loader (contenu déjà dans la bonne langue)
	experiences := s.contentLoader.GetExperiences(lang)
	skills := s.contentLoader.GetSkills(lang)
	projects := s.contentLoader.GetProjects(lang)

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

	// Ordre chronologique inverse, pas par score (cf. cv_experience_order.go).
	ordonnerExperiencesCV(filteredExperiences)

	filteredSkills := make([]ScoredSkillResponse, 0, len(scoredSkills))
	for _, ss := range scoredSkills {
		localizedSkill := s.l10nHelper.LocalizeSkill(ss.Skill, lang)
		filteredSkills = append(filteredSkills, ScoredSkillResponse{
			Skill: localizedSkill,
			Score: ss.Score,
		})
	}

	filteredProjects := s.classerProjets(ctx, theme, scoredProjects, lang)

	// 6. Construire réponse
	response := &AdaptiveCVResponse{
		Theme:       *theme,
		Experiences: filteredExperiences,
		Skills:      filteredSkills,
		Projects:    filteredProjects,
		GeneratedAt: time.Now(),
	}

	// 7. Mettre en cache (TTL 5 min — aligné sur le TTL du content provider, plancher de fraîcheur).
	// Court exprès : un changement de contenu/traduction apparaît en ~5 min sans recalcul par requête
	// (le scoring LLM reste caché 6h à part). Avant : 1h → contenu figé trop longtemps.
	jsonData, err := json.Marshal(response)
	if err == nil {
		s.redis.Set(ctx, cacheKey, jsonData, 5*time.Minute)
	}

	return response, nil
}

// classerProjets : pertinence (LLM, sinon poids de tags) × preuve × activité, puis tri décroissant.
// Cf. cv_project_ranking.go pour la formule et llm_scoring.go pour la pertinence.
//
// POURQUOI la pertinence se calcule sur le catalogue FRANÇAIS quelle que soit la langue du CV :
// les scores sont indexés par identifiant, donc valables dans toutes les langues. Un seul appel
// LLM par thème au lieu d'un par langue, et un CV anglais classé exactement comme le français.
func (s *CVService) classerProjets(ctx context.Context, theme *config.CVTheme, scored []ScoredProject, lang string) []ScoredProjectResponse {
	// Repli : poids de tags du thème, normalisés (ScoreProjects les a déjà calculés).
	tags := make(map[string]float64, len(scored))
	for _, sp := range scored {
		tags[sp.Project.SourceID] = sp.Score
	}
	pertinences := pertinencesDeRepli(tags)

	if s.llmScoring != nil {
		llm, err := s.llmScoring.ScoreProjectsForTheme(ctx, s.contentLoader.GetProjects("fr"), theme)
		if err != nil {
			// Repli journalisé, JAMAIS muet : ce repli a déjà tourné des mois sans que personne
			// ne le sache (budget de sortie trop court, cf. llm_scoring.go).
			log.Warn().Err(err).Str("theme", theme.ID).Int("projets", len(scored)).
				Msg("pertinence LLM des projets indisponible, repli sur les poids de tags")
		} else {
			for id, v := range llm {
				pertinences[id] = float64(v) / 100.0
			}
		}
	}

	out := make([]ScoredProjectResponse, 0, len(scored))
	for _, sp := range scored {
		out = append(out, ScoredProjectResponse{
			Project: s.l10nHelper.LocalizeProject(sp.Project, lang),
			Score:   scoreFinalProjet(pertinences[sp.Project.SourceID], sp.Project),
		})
	}
	// Tri stable : à égalité, l'ordre du catalogue — jamais un ordre aléatoire d'un appel à l'autre.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// GetAllExperiences retourne toutes les expériences dans la langue demandée.
func (s *CVService) GetAllExperiences(ctx context.Context, lang string) ([]models.Experience, error) {
	return s.contentLoader.GetExperiences(lang), nil
}

// GetAllSkills retourne toutes les compétences dans la langue demandée.
func (s *CVService) GetAllSkills(ctx context.Context, lang string) ([]models.Skill, error) {
	return s.contentLoader.GetSkills(lang), nil
}

// GetAllProjects retourne tous les projets dans la langue demandée.
func (s *CVService) GetAllProjects(ctx context.Context, lang string) ([]models.Project, error) {
	return s.contentLoader.GetProjects(lang), nil
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
