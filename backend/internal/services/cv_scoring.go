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
	switch strings.ToLower(string(skill.Level)) {
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
// Retourne TOUS les projets, triés par score (les plus pertinents en premier)
func (s *CVScoringService) ScoreProjects(projects []models.Project, theme *config.CVTheme) []ScoredProject {
	scored := make([]ScoredProject, 0, len(projects))

	for _, project := range projects {
		score := s.CalculateProjectScore(project, theme)
		// Inclure tous les projets avec un score minimum de 0.1 pour les non-matchés
		if score == 0 {
			score = 0.1
		}
		scored = append(scored, ScoredProject{
			Project: project,
			Score:   score,
		})
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
	// Tri par score décroissant (bubble sort simple)
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
