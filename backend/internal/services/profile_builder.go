package services

import (
	"context"
	"fmt"
	"strings"
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

	// 4. Récupérer les expériences détaillées (max 5 plus récentes)
	var experiences []models.Experience
	pb.db.Order("start_date DESC").Limit(5).Find(&experiences)

	experienceDetails := make([]models.ExperienceDetail, 0, len(experiences))
	for _, exp := range experiences {
		duration := formatDuration(exp.StartDate, exp.EndDate)

		// Extraire les highlights de la description (phrases séparées par des points)
		highlights := extractHighlights(exp.Description)

		experienceDetails = append(experienceDetails, models.ExperienceDetail{
			Title:       exp.Title,
			Company:     exp.Company,
			Duration:    duration,
			Description: exp.Description,
			Highlights:  highlights,
		})
	}

	// 5. Construire un résumé professionnel
	summary := pb.buildSummary(currentRole, yearsOfExperience, skillNames)

	profile := models.UserProfile{
		Name:        "Alexis Trouve",
		Address:     "17 rue principale",
		PostalCode:  "79100",
		City:        "Tourtenay",
		Email:       "alexistrouve.pro@gmail.com",
		Phone:       "+33 6 95 11 09 67",
		CurrentRole: currentRole,
		Skills:      skillNames,
		Experience:  yearsOfExperience,
		Experiences: experienceDetails,
		Summary:     summary,
	}

	log.Info().
		Str("name", profile.Name).
		Str("role", profile.CurrentRole).
		Int("skills_count", len(profile.Skills)).
		Int("experience_years", profile.Experience).
		Int("experiences_count", len(profile.Experiences)).
		Msg("User profile built from database")

	return profile
}

// formatDuration formate la durée d'une expérience
func formatDuration(start time.Time, end *time.Time) string {
	startYear := start.Year()
	if end == nil {
		return fmt.Sprintf("%d - présent", startYear)
	}
	return fmt.Sprintf("%d - %d", startYear, end.Year())
}

// extractHighlights extrait les points clés d'une description
func extractHighlights(description string) []string {
	if description == "" {
		return nil
	}

	// Séparer par les phrases (points suivis d'espace ou fin de chaîne)
	sentences := strings.Split(description, ". ")
	highlights := make([]string, 0)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 { // Ignorer les fragments trop courts
			// Nettoyer le point final si présent
			sentence = strings.TrimSuffix(sentence, ".")
			highlights = append(highlights, sentence)
		}
	}

	// Limiter à 3 highlights par expérience
	if len(highlights) > 3 {
		highlights = highlights[:3]
	}

	return highlights
}

// buildSummary construit un résumé professionnel
func (pb *ProfileBuilder) buildSummary(role string, years int, skills []string) string {
	topSkills := skills
	if len(topSkills) > 5 {
		topSkills = topSkills[:5]
	}

	return fmt.Sprintf(
		"%s avec %d ans d'expérience, spécialisé en %s",
		role,
		years,
		strings.Join(topSkills, ", "),
	)
}
