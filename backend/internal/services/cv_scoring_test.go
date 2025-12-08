// backend/internal/services/cv_scoring_test.go
package services

import (
	"testing"
	"time"

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
	experience := &Experience{
		Title:  "Backend Developer",
		Tags:   []string{"go", "postgresql", "redis"},
		Years:  3,
	}

	theme := &Theme{
		Name:     "backend",
		Keywords: []string{"go", "postgresql"},
		Weight:   1.0,
	}

	score := suite.service.ScoreExperience(experience, theme)

	// Assert score élevé (2 tags sur 3 matchent)
	assert.Greater(suite.T(), score, 0.6, "Score devrait être > 0.6 pour 2 tags matchés")
	assert.LessOrEqual(suite.T(), score, 1.0, "Score max est 1.0")
}

// Test scoring sans match
func (suite *CVScoringTestSuite) TestScoreExperience_NoMatch() {
	experience := &Experience{
		Title: "Designer",
		Tags:  []string{"photoshop", "illustrator"},
		Years: 2,
	}

	theme := &Theme{
		Name:     "backend",
		Keywords: []string{"go", "postgresql"},
		Weight:   1.0,
	}

	score := suite.service.ScoreExperience(experience, theme)

	// Assert score très bas ou 0
	assert.Equal(suite.T(), 0.0, score, "Score devrait être 0 sans match")
}

// Test tri par score décroissant
func (suite *CVScoringTestSuite) TestSortExperiencesByScore() {
	experiences := []*Experience{
		{Title: "Backend Dev", Tags: []string{"go"}, Score: 0.8},
		{Title: "Frontend Dev", Tags: []string{"react"}, Score: 0.3},
		{Title: "Full-Stack Dev", Tags: []string{"go", "react"}, Score: 0.9},
	}

	sorted := suite.service.SortByScore(experiences)

	// Assert ordre décroissant
	assert.Equal(suite.T(), "Full-Stack Dev", sorted[0].Title)
	assert.Equal(suite.T(), "Backend Dev", sorted[1].Title)
	assert.Equal(suite.T(), "Frontend Dev", sorted[2].Title)
}

// Test scoring avec expérience récente (bonus)
func (suite *CVScoringTestSuite) TestScoreExperience_RecentExperienceBonus() {
	recentExp := &Experience{
		Title:     "Senior Backend Dev",
		Tags:      []string{"go"},
		StartDate: time.Now().AddDate(-1, 0, 0), // 1 an
		EndDate:   nil,                           // Current
	}

	oldExp := &Experience{
		Title:     "Backend Dev",
		Tags:      []string{"go"},
		StartDate: time.Now().AddDate(-5, 0, 0), // 5 ans
		EndDate:   &[]time.Time{time.Now().AddDate(-4, 0, 0)}[0],
	}

	theme := &Theme{
		Name:     "backend",
		Keywords: []string{"go"},
		Weight:   1.0,
	}

	recentScore := suite.service.ScoreExperience(recentExp, theme)
	oldScore := suite.service.ScoreExperience(oldExp, theme)

	// Expérience récente devrait avoir un score supérieur
	assert.Greater(suite.T(), recentScore, oldScore, "Expérience récente devrait avoir meilleur score")
}

// Test avec thème inexistant
func (suite *CVScoringTestSuite) TestScoreExperience_InvalidTheme() {
	experience := &Experience{
		Title: "Developer",
		Tags:  []string{"python", "django"},
	}

	theme := &Theme{
		Name:     "",
		Keywords: []string{},
		Weight:   0.0,
	}

	score := suite.service.ScoreExperience(experience, theme)

	assert.Equal(suite.T(), 0.0, score, "Thème vide devrait donner score 0")
}

// Test table-driven pour multiple thèmes
func (suite *CVScoringTestSuite) TestScoreExperience_MultipleThemes() {
	experience := &Experience{
		Title: "Full-Stack Developer",
		Tags:  []string{"go", "react", "postgresql", "docker"},
		Years: 4,
	}

	testCases := []struct {
		name          string
		theme         *Theme
		expectedScore float64
		description   string
	}{
		{
			name: "Backend Theme",
			theme: &Theme{
				Name:     "backend",
				Keywords: []string{"go", "postgresql"},
				Weight:   1.0,
			},
			expectedScore: 0.5,
			description:   "2 tags sur 4 matchent",
		},
		{
			name: "Frontend Theme",
			theme: &Theme{
				Name:     "frontend",
				Keywords: []string{"react"},
				Weight:   1.0,
			},
			expectedScore: 0.25,
			description:   "1 tag sur 4 matche",
		},
		{
			name: "DevOps Theme",
			theme: &Theme{
				Name:     "devops",
				Keywords: []string{"docker", "kubernetes"},
				Weight:   1.0,
			},
			expectedScore: 0.25,
			description:   "1 tag sur 4 matche",
		},
		{
			name: "Full-Stack Theme",
			theme: &Theme{
				Name:     "fullstack",
				Keywords: []string{"go", "react", "postgresql"},
				Weight:   1.0,
			},
			expectedScore: 0.75,
			description:   "3 tags sur 4 matchent",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			score := suite.service.ScoreExperience(experience, tc.theme)
			assert.InDelta(t, tc.expectedScore, score, 0.1, tc.description)
		})
	}
}

// Test filtrage expériences par seuil minimum
func (suite *CVScoringTestSuite) TestFilterExperiencesByMinScore() {
	experiences := []*Experience{
		{Title: "Backend Dev", Tags: []string{"go"}, Score: 0.9},
		{Title: "Frontend Dev", Tags: []string{"react"}, Score: 0.2},
		{Title: "DevOps Eng", Tags: []string{"docker"}, Score: 0.7},
		{Title: "Designer", Tags: []string{"figma"}, Score: 0.1},
	}

	filtered := suite.service.FilterByMinScore(experiences, 0.5)

	assert.Len(suite.T(), filtered, 2, "Devrait retourner 2 expériences avec score >= 0.5")
	assert.Equal(suite.T(), "Backend Dev", filtered[0].Title)
	assert.Equal(suite.T(), "DevOps Eng", filtered[1].Title)
}

// Test limite nombre résultats
func (suite *CVScoringTestSuite) TestLimitExperiences() {
	experiences := []*Experience{
		{Title: "Exp 1", Score: 0.9},
		{Title: "Exp 2", Score: 0.8},
		{Title: "Exp 3", Score: 0.7},
		{Title: "Exp 4", Score: 0.6},
		{Title: "Exp 5", Score: 0.5},
	}

	limited := suite.service.Limit(experiences, 3)

	assert.Len(suite.T(), limited, 3, "Devrait retourner exactement 3 résultats")
	assert.Equal(suite.T(), "Exp 1", limited[0].Title)
	assert.Equal(suite.T(), "Exp 2", limited[1].Title)
	assert.Equal(suite.T(), "Exp 3", limited[2].Title)
}

// Lancer la suite
func TestCVScoringTestSuite(t *testing.T) {
	suite.Run(t, new(CVScoringTestSuite))
}

// Benchmark pour algorithme de scoring (performance)
func BenchmarkScoreExperience(b *testing.B) {
	service := NewCVScoringService()

	experience := &Experience{
		Title: "Full-Stack Developer",
		Tags:  []string{"go", "react", "postgresql", "docker", "kubernetes"},
		Years: 5,
	}

	theme := &Theme{
		Name:     "backend",
		Keywords: []string{"go", "postgresql", "redis", "mongodb"},
		Weight:   1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ScoreExperience(experience, theme)
	}
}
