package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"maicivy/internal/models"
)

// motivationPromptVersion retourne la version active depuis l'env var PROMPT_VERSION.
// Valeurs : "v1" (original), "v2" (actif par défaut)
// Switch sans rebuild : changer PROMPT_VERSION dans .env + docker compose restart backend
func motivationPromptVersion() string {
	if v := os.Getenv("PROMPT_VERSION"); v != "" {
		return v
	}
	return "v2"
}

type PromptBuilder struct {
	userProfile models.UserProfile
	projects    []models.Project // projets perso — injectés dans le prompt
}

func NewPromptBuilder(profile models.UserProfile, projects []models.Project) *PromptBuilder {
	return &PromptBuilder{userProfile: profile, projects: projects}
}

// BuildMotivationPrompt : prompt pour lettre de motivation professionnelle
// jobOffer est optionnel — si fourni, le prompt inclut l'offre pour une lettre tailorée
func (pb *PromptBuilder) BuildMotivationPrompt(company models.CompanyInfo, lang string, jobOffer ...string) string {
	offer := ""
	if len(jobOffer) > 0 {
		offer = jobOffer[0]
	}
	if lang == "en" {
		return pb.buildMotivationPromptEN(company, offer)
	}
	return pb.buildMotivationPromptFR(company, offer)
}

// buildMotivationPromptFR : routeur de version — délègue à la version active (PROMPT_VERSION env)
func (pb *PromptBuilder) buildMotivationPromptFR(company models.CompanyInfo, jobOffer string) string {
	if motivationPromptVersion() == "v1" {
		return pb.buildMotivationPromptFR_v1(company, jobOffer)
	}
	return pb.buildMotivationPromptFR_v2(company, jobOffer)
}

// buildProjectsSection construit la section projets pour le prompt
// Format compact : titre | techs | catchphrase — chaque projet = 1 ligne
func (pb *PromptBuilder) buildProjectsSection() string {
	if len(pb.projects) == 0 {
		return "Aucun projet disponible."
	}

	var sb strings.Builder
	for _, p := range pb.projects {
		techs := strings.Join(p.Technologies, ", ")
		status := ""
		if p.InProgress {
			status = " [en cours]"
		}
		sb.WriteString(fmt.Sprintf("• %s%s [%s] — %s\n", p.Title, status, techs, p.Catchphrase))
	}
	return sb.String()
}

// buildExperiencesSection construit la section des expériences pour le prompt
func (pb *PromptBuilder) buildExperiencesSection() string {
	if len(pb.userProfile.Experiences) == 0 {
		return "Aucune expérience détaillée disponible."
	}

	var sb strings.Builder
	for i, exp := range pb.userProfile.Experiences {
		sb.WriteString(fmt.Sprintf("%d. %s @ %s (%s)\n", i+1, exp.Title, exp.Company, exp.Duration))
		if len(exp.Highlights) > 0 {
			for _, h := range exp.Highlights {
				sb.WriteString(fmt.Sprintf("   • %s\n", h))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildMotivationPromptEN : routeur de version — délègue à la version active (PROMPT_VERSION env)
func (pb *PromptBuilder) buildMotivationPromptEN(company models.CompanyInfo, jobOffer string) string {
	if motivationPromptVersion() == "v1" {
		return pb.buildMotivationPromptEN_v1(company, jobOffer)
	}
	return pb.buildMotivationPromptEN_v2(company, jobOffer)
}

// BuildAntiMotivationPrompt : prompt pour lettre d'anti-motivation humoristique
// Implémentation dans prompts_anti.go
func (pb *PromptBuilder) BuildAntiMotivationPrompt(company models.CompanyInfo, lang string) string {
	if lang == "en" {
		return pb.buildAntiMotivationPromptEN(company)
	}
	return pb.buildAntiMotivationPromptFR(company)
}

// Ex: "Tourtenay, le 5 janvier 2026"
func formatFrenchDate(t time.Time) string {
	frenchMonths := map[time.Month]string{
		time.January:   "janvier",
		time.February:  "février",
		time.March:     "mars",
		time.April:     "avril",
		time.May:       "mai",
		time.June:      "juin",
		time.July:      "juillet",
		time.August:    "août",
		time.September: "septembre",
		time.October:   "octobre",
		time.November:  "novembre",
		time.December:  "décembre",
	}

	return fmt.Sprintf("Tourtenay, le %d %s %d",
		t.Day(),
		frenchMonths[t.Month()],
		t.Year(),
	)
}

// formatEnglishDate formate une date au format anglais pour les lettres
// Ex: "Tourtenay, January 5, 2026"
func formatEnglishDate(t time.Time) string {
	return fmt.Sprintf("Tourtenay, %s %d, %d",
		t.Month().String(),
		t.Day(),
		t.Year(),
	)
}
