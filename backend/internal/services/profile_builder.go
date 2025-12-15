package services

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// ProfileBuilder construit le profil utilisateur à partir de la base de données
type ProfileBuilder struct {
	db *gorm.DB
}

// NewProfileBuilder crée une nouvelle instance de ProfileBuilder
func NewProfileBuilder(db *gorm.DB) *ProfileBuilder {
	return &ProfileBuilder{
		db: db,
	}
}

// BuildProfile construit le UserProfile à partir des données en BDD
func (pb *ProfileBuilder) BuildProfile(ctx context.Context) models.UserProfile {
	// 1. Récupérer l'expérience la plus récente pour le CurrentRole
	var latestExperience models.Experience
	result := pb.db.Order("start_date DESC").First(&latestExperience)

	currentRole := "Développeur Full-Stack" // Fallback
	if result.Error == nil {
		currentRole = latestExperience.Title
	}

	// 2. Récupérer les skills featured (max 10)
	var skills []models.Skill
	pb.db.Where("featured = ?", true).
		Order("years_experience DESC").
		Limit(10).
		Find(&skills)

	skillNames := make([]string, len(skills))
	for i, skill := range skills {
		skillNames[i] = skill.Name
	}

	// Fallback si pas de skills
	if len(skillNames) == 0 {
		skillNames = []string{"Go", "PostgreSQL", "Next.js", "TypeScript", "Docker"}
	}

	// 3. Calculer les années d'expérience
	var firstExperience, lastExperience models.Experience

	// Première expérience (la plus ancienne)
	result = pb.db.Order("start_date ASC").First(&firstExperience)

	yearsOfExperience := 5 // Fallback
	if result.Error == nil {
		// Dernière expérience (la plus récente ou en cours)
		pb.db.Order("start_date DESC").First(&lastExperience)

		startDate := firstExperience.StartDate
		endDate := time.Now()

		// Si lastExperience a une end_date, l'utiliser
		if lastExperience.EndDate != nil {
			endDate = *lastExperience.EndDate
		}

		yearsOfExperience = int(endDate.Sub(startDate).Hours() / (24 * 365))
		if yearsOfExperience < 1 {
			yearsOfExperience = 1
		}
	}

	profile := models.UserProfile{
		Name:        "Alexi",
		CurrentRole: currentRole,
		Skills:      skillNames,
		Experience:  yearsOfExperience,
	}

	log.Info().
		Str("name", profile.Name).
		Str("role", profile.CurrentRole).
		Int("skills_count", len(profile.Skills)).
		Int("experience_years", profile.Experience).
		Msg("User profile built from database")

	return profile
}
