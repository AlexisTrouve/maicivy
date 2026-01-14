package services

import (
	"maicivy/internal/models"
)

// LocalizationHelper provides utilities for handling i18n content
type LocalizationHelper struct{}

// NewLocalizationHelper creates a new localization helper
func NewLocalizationHelper() *LocalizationHelper {
	return &LocalizationHelper{}
}

// GetLocalizedField returns the appropriate field value based on language
// Falls back to French (default) if English translation is empty or language is not "en"
func (l *LocalizationHelper) GetLocalizedField(frValue, enValue, lang string) string {
	if lang == "en" && enValue != "" {
		return enValue
	}
	return frValue
}

// LocalizeExperience returns a localized copy of an experience
// This creates a new Experience with localized fields based on the requested language
func (l *LocalizationHelper) LocalizeExperience(exp models.Experience, lang string) models.Experience {
	if lang != "en" {
		return exp // Return as-is for French (default)
	}

	// Create a copy to avoid modifying the original
	localized := exp

	// Replace fields with English versions if available
	if exp.TitleEn != "" {
		localized.Title = exp.TitleEn
	}
	if exp.DescriptionEn != "" {
		localized.Description = exp.DescriptionEn
	}
	if exp.CatchphraseEn != "" {
		localized.Catchphrase = exp.CatchphraseEn
	}
	if exp.FunctionalDescriptionEn != "" {
		localized.FunctionalDescription = exp.FunctionalDescriptionEn
	}
	if exp.TechnicalDescriptionEn != "" {
		localized.TechnicalDescription = exp.TechnicalDescriptionEn
	}

	return localized
}

// LocalizeSkill returns a localized copy of a skill
func (l *LocalizationHelper) LocalizeSkill(skill models.Skill, lang string) models.Skill {
	if lang != "en" {
		return skill // Return as-is for French (default)
	}

	// Create a copy to avoid modifying the original
	localized := skill

	// Replace fields with English versions if available
	if skill.NameEn != "" {
		localized.Name = skill.NameEn
	}
	if skill.DescriptionEn != "" {
		localized.Description = skill.DescriptionEn
	}

	return localized
}

// LocalizeProject returns a localized copy of a project
func (l *LocalizationHelper) LocalizeProject(project models.Project, lang string) models.Project {
	if lang != "en" {
		return project // Return as-is for French (default)
	}

	// Create a copy to avoid modifying the original
	localized := project

	// Replace fields with English versions if available
	if project.TitleEn != "" {
		localized.Title = project.TitleEn
	}
	if project.DescriptionEn != "" {
		localized.Description = project.DescriptionEn
	}
	if project.CatchphraseEn != "" {
		localized.Catchphrase = project.CatchphraseEn
	}
	if project.FunctionalDescriptionEn != "" {
		localized.FunctionalDescription = project.FunctionalDescriptionEn
	}
	if project.TechnicalDescriptionEn != "" {
		localized.TechnicalDescription = project.TechnicalDescriptionEn
	}

	return localized
}

// IsValidLanguage checks if a language code is supported
func (l *LocalizationHelper) IsValidLanguage(lang string) bool {
	return lang == "fr" || lang == "en"
}

// GetDefaultLanguage returns the default language code
func (l *LocalizationHelper) GetDefaultLanguage() string {
	return "fr"
}

// NormalizeLanguage ensures the language is valid, returning default if not
func (l *LocalizationHelper) NormalizeLanguage(lang string) string {
	if l.IsValidLanguage(lang) {
		return lang
	}
	return l.GetDefaultLanguage()
}
