//go:build testing

package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"maicivy/internal/models"
)

// CreateTestExperience crée une expérience de test avec des valeurs par défaut
func CreateTestExperience() *models.Experience {
	return &models.Experience{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Title:        "Backend Developer",
		Company:      "TestCorp",
		Description:  "Test experience for unit tests",
		StartDate:    time.Now().AddDate(-2, 0, 0),
		Technologies: pq.StringArray{"go", "postgresql", "redis"},
		Tags:         pq.StringArray{"backend", "api"},
		Category:     "backend",
		Featured:     true,
	}
}

// CreateTestSkill crée une compétence de test
func CreateTestSkill() *models.Skill {
	return &models.Skill{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Name:            "Go",
		Level:           models.SkillLevelExpert,
		Category:        "backend",
		YearsExperience: 5,
		Tags:            pq.StringArray{"backend", "programming"},
	}
}

// CreateTestProject crée un projet de test
func CreateTestProject() *models.Project {
	return &models.Project{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Title:        "maicivy",
		Description:  "CV interactif avec IA",
		Technologies: pq.StringArray{"go", "react", "postgresql"},
		Category:     "fullstack",
		Featured:     true,
	}
}

// CreateTestVisitor crée un visiteur de test
func CreateTestVisitor(sessionID string) *models.Visitor {
	if sessionID == "" {
		sessionID = "test-session-" + uuid.New().String()
	}

	return &models.Visitor{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		SessionID:       sessionID,
		VisitCount:      1,
		FirstVisit:      time.Now(),
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeUnknown,
	}
}

// CreateTestGeneratedLetter crée une lettre générée de test
func CreateTestGeneratedLetter(sessionID, companyName string) *models.GeneratedLetter {
	return &models.GeneratedLetter{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		SessionID:   sessionID,
		CompanyName: companyName,
		JobTitle:    "Backend Developer",
		Theme:       "backend",
		Status:      "completed",
	}
}

// CreateTestGitHubToken crée un token GitHub de test
func CreateTestGitHubToken(sessionID string) *models.GitHubToken {
	return &models.GitHubToken{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		SessionID:   sessionID,
		AccessToken: "gho_test_token_12345",
	}
}

// CreateTestGitHubProfile crée un profil GitHub de test
func CreateTestGitHubProfile(sessionID string) *models.GitHubProfile {
	return &models.GitHubProfile{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		SessionID:    sessionID,
		GitHubID:     12345,
		Login:        "testuser",
		Name:         "Test User",
		Email:        "test@example.com",
		Bio:          "Test bio",
		PublicRepos:  10,
		PublicGists:  5,
		Followers:    50,
		Following:    30,
	}
}
