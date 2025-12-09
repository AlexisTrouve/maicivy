// backend/internal/services/cv_scoring_test.go
package services

import (
	"testing"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Suite de tests pour le service de scoring CV
type CVScoringTestSuite struct {
	suite.Suite
	service *CVScoringService
}

// Setup avant chaque test
func (suite *CVScoringTestSuite) SetupTest() {
	suite.service = NewCVScoringService()
}

// Test scoring avec tags exacts
func (suite *CVScoringTestSuite) TestScoreExperience_ExactTagMatch() {
	experience := models.Experience{
		Title:        "Backend Developer",
		Company:      "Test Corp",
		Tags:         pq.StringArray{"go", "postgresql", "redis"},
		Technologies: pq.StringArray{"go", "postgresql"},
		Category:     "backend",
		StartDate:    time.Now().AddDate(-3, 0, 0),
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend Developer",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 1.0,
		},
	}

	score := suite.service.CalculateExperienceScore(experience, theme)

	// Assert score élevé (2 tags sur 2 matchent)
	assert.Greater(suite.T(), score, 0.5, "Score devrait être > 0.5 pour tags matchés")
	assert.LessOrEqual(suite.T(), score, 2.0, "Score max dépend du nombre de tags")
}

// Test scoring sans match
func (suite *CVScoringTestSuite) TestScoreExperience_NoMatch() {
	experience := models.Experience{
		Title:        "Designer",
		Company:      "Design Co",
		Tags:         pq.StringArray{"photoshop", "illustrator"},
		Technologies: pq.StringArray{},
		Category:     "design",
		StartDate:    time.Now().AddDate(-2, 0, 0),
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend Developer",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 1.0,
		},
	}

	score := suite.service.CalculateExperienceScore(experience, theme)

	// Assert score très bas ou 0
	assert.Equal(suite.T(), 0.0, score, "Score devrait être 0 sans match")
}

// Test scoring skills
func (suite *CVScoringTestSuite) TestScoreSkill_ExactMatch() {
	skill := models.Skill{
		Name:            "Go",
		Level:           models.SkillLevelExpert,
		Category:        "backend",
		YearsExperience: 5,
		Tags:            pq.StringArray{"backend", "programming"},
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend Developer",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":      1.0,
			"backend": 0.8,
		},
	}

	score := suite.service.CalculateSkillScore(skill, theme)

	// Assert score élevé avec bonus level expert et années
	assert.Greater(suite.T(), score, 0.5, "Score devrait être élevé pour expert avec 5+ ans")
}

// Test scoring projects
func (suite *CVScoringTestSuite) TestScoreProject_FeaturedBonus() {
	project := models.Project{
		Title:        "maicivy",
		Description:  "CV interactif",
		Technologies: pq.StringArray{"go", "react", "postgresql"},
		Category:     "fullstack",
		Featured:     true,
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend Developer",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 1.0,
		},
	}

	score := suite.service.CalculateProjectScore(project, theme)

	// Assert bonus featured
	assert.Greater(suite.T(), score, 0.3, "Score devrait inclure bonus featured")
}

// Test liste complète d'expériences
func (suite *CVScoringTestSuite) TestScoreExperiences_FilterAndSort() {
	experiences := []models.Experience{
		{
			Title:        "Backend Dev",
			Company:      "Company A",
			Tags:         pq.StringArray{"go", "postgresql"},
			Technologies: pq.StringArray{"go"},
			Category:     "backend",
			StartDate:    time.Now().AddDate(-3, 0, 0),
		},
		{
			Title:        "Frontend Dev",
			Company:      "Company B",
			Tags:         pq.StringArray{"react"},
			Technologies: pq.StringArray{"react"},
			Category:     "frontend",
			StartDate:    time.Now().AddDate(-2, 0, 0),
		},
		{
			Title:        "Full-Stack Dev",
			Company:      "Company C",
			Tags:         pq.StringArray{"go", "react"},
			Technologies: pq.StringArray{"go", "postgresql"},
			Category:     "fullstack",
			StartDate:    time.Now().AddDate(-1, 0, 0),
		},
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend Developer",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 0.9,
		},
	}

	scored := suite.service.ScoreExperiences(experiences, theme)

	// Devrait filtrer seulement celles avec score > 0
	assert.GreaterOrEqual(suite.T(), len(scored), 1, "Devrait avoir au moins une expérience pertinente")

	// Vérifier tri décroissant si plusieurs résultats
	if len(scored) > 1 {
		assert.GreaterOrEqual(suite.T(), scored[0].Score, scored[1].Score, "Devrait être trié par score décroissant")
	}
}

// Test avec thème inexistant (nil)
func (suite *CVScoringTestSuite) TestScoreExperience_NilTheme() {
	experience := models.Experience{
		Title:     "Developer",
		Company:   "Test",
		Tags:      pq.StringArray{"python", "django"},
		StartDate: time.Now().AddDate(-1, 0, 0),
	}

	score := suite.service.CalculateExperienceScore(experience, nil)

	assert.Equal(suite.T(), 0.0, score, "Thème nil devrait donner score 0")
}

// Test table-driven pour multiple thèmes
func (suite *CVScoringTestSuite) TestScoreExperience_MultipleThemes() {
	experience := models.Experience{
		Title:        "Full-Stack Developer",
		Company:      "TechCorp",
		Tags:         pq.StringArray{"fullstack", "web"},
		Technologies: pq.StringArray{"go", "react", "postgresql", "docker"},
		Category:     "fullstack",
		StartDate:    time.Now().AddDate(-4, 0, 0),
	}

	testCases := []struct {
		name        string
		theme       *config.CVTheme
		minScore    float64
		description string
	}{
		{
			name: "Backend Theme",
			theme: &config.CVTheme{
				ID:          "backend",
				Name:        "Backend",
				Description: "Backend dev",
				TagWeights: map[string]float64{
					"go":         1.0,
					"postgresql": 0.9,
				},
			},
			minScore:    0.3,
			description: "2 technologies matchent",
		},
		{
			name: "Frontend Theme",
			theme: &config.CVTheme{
				ID:          "frontend",
				Name:        "Frontend",
				Description: "Frontend dev",
				TagWeights: map[string]float64{
					"react": 1.0,
				},
			},
			minScore:    0.1,
			description: "1 technologie matche",
		},
		{
			name: "DevOps Theme",
			theme: &config.CVTheme{
				ID:          "devops",
				Name:        "DevOps",
				Description: "DevOps",
				TagWeights: map[string]float64{
					"docker":     1.0,
					"kubernetes": 0.9,
				},
			},
			minScore:    0.1,
			description: "1 technologie matche",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			score := suite.service.CalculateExperienceScore(experience, tc.theme)
			assert.GreaterOrEqual(t, score, tc.minScore, tc.description)
		})
	}
}

// Test scoring de skills
func (suite *CVScoringTestSuite) TestScoreSkills_FilterAndSort() {
	skills := []models.Skill{
		{
			Name:            "Go",
			Level:           models.SkillLevelExpert,
			Category:        "backend",
			YearsExperience: 5,
		},
		{
			Name:            "React",
			Level:           models.SkillLevelIntermediate,
			Category:        "frontend",
			YearsExperience: 2,
		},
		{
			Name:            "PostgreSQL",
			Level:           models.SkillLevelAdvanced,
			Category:        "database",
			YearsExperience: 4,
		},
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 0.9,
		},
	}

	scored := suite.service.ScoreSkills(skills, theme)

	// Devrait avoir au moins Go et PostgreSQL
	assert.GreaterOrEqual(suite.T(), len(scored), 2, "Devrait avoir au moins 2 skills pertinentes")

	// Vérifier tri décroissant
	if len(scored) > 1 {
		assert.GreaterOrEqual(suite.T(), scored[0].Score, scored[1].Score, "Devrait être trié par score décroissant")
	}
}

// Test scoring de projects
func (suite *CVScoringTestSuite) TestScoreProjects_FilterAndSort() {
	projects := []models.Project{
		{
			Title:        "Backend API",
			Technologies: pq.StringArray{"go", "postgresql", "redis"},
			Category:     "backend",
			Featured:     true,
		},
		{
			Title:        "Frontend App",
			Technologies: pq.StringArray{"react", "typescript"},
			Category:     "frontend",
			Featured:     false,
		},
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 0.9,
			"redis":      0.8,
		},
	}

	scored := suite.service.ScoreProjects(projects, theme)

	// Devrait avoir au moins le projet backend
	assert.GreaterOrEqual(suite.T(), len(scored), 1, "Devrait avoir au moins 1 projet pertinent")

	// Le premier devrait être le backend avec featured
	if len(scored) > 0 {
		assert.Equal(suite.T(), "Backend API", scored[0].Project.Title)
	}
}

// Lancer la suite
func TestCVScoringTestSuite(t *testing.T) {
	suite.Run(t, new(CVScoringTestSuite))
}

// Benchmark pour algorithme de scoring (performance)
func BenchmarkScoreExperience(b *testing.B) {
	service := NewCVScoringService()

	experience := models.Experience{
		Title:        "Full-Stack Developer",
		Company:      "BenchCorp",
		Tags:         pq.StringArray{"fullstack", "web", "api"},
		Technologies: pq.StringArray{"go", "react", "postgresql", "docker", "kubernetes"},
		Category:     "fullstack",
		StartDate:    time.Now().AddDate(-5, 0, 0),
	}

	theme := &config.CVTheme{
		ID:          "backend",
		Name:        "Backend",
		Description: "Backend development",
		TagWeights: map[string]float64{
			"go":         1.0,
			"postgresql": 0.9,
			"redis":      0.8,
			"mongodb":    0.7,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CalculateExperienceScore(experience, theme)
	}
}
