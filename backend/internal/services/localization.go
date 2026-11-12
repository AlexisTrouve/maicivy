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

// IsValidLanguage indique si le CV a du contenu NATIF pour cette langue.
// QUOI : fr/en/zh. POURQUOI : content_provider fetche chacune de ces langues depuis maiProFiles et garde un
// cache par langue (champs principaux déjà dans la langue) ; maiProFiles sert du vrai chinois via ?lang=zh.
// de/it n'ont pas de contenu natif → repli anglais. Doit rester aligné sur providerLangs.
func (l *LocalizationHelper) IsValidLanguage(lang string) bool {
	return lang == "fr" || lang == "en" || lang == "zh"
}

// GetDefaultLanguage retourne la langue servie quand celle de l'utilisateur n'a PAS de contenu natif.
// POURQUOI "en" et non "fr" : un visiteur de/it (ou langue inconnue) ne doit jamais recevoir un CV en
// FRANÇAIS — l'anglais est le défaut neutre/international. Le français reste servi à un visiteur fr (natif).
// (Symétrique du repli côté chat : maiprofiles_client.fallbackLang.)
func (l *LocalizationHelper) GetDefaultLanguage() string {
	return "en"
}

// NormalizeLanguage mappe la locale de l'utilisateur vers la langue de contenu à servir :
// fr/en/zh → natif (servi tel quel) ; toute autre (de/it, vide, inconnue) → anglais (GetDefaultLanguage).
func (l *LocalizationHelper) NormalizeLanguage(lang string) string {
	if l.IsValidLanguage(lang) {
		return lang
	}
	return l.GetDefaultLanguage()
}

// PDFLanguages : langues pour lesquelles un PDF de CV est produit tel quel (export standard,
// catalogue, CV tailoré).
// POURQUOI une liste unique : le gabarit (getLabels), le texte ATS (buildStealthText) et le prompt
// tailoré ont chacun une version pour ces langues ; validation de l'export et catalogue s'y
// alignent au lieu de répéter fr/en en dur. Ajouter une langue ici exige ces trois versions.
var PDFLanguages = []string{"fr", "en", "zh"}

// IsPDFLanguage : vrai si `lang` a un PDF de CV natif (cf. PDFLanguages).
func IsPDFLanguage(lang string) bool {
	for _, l := range PDFLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// TailoredLanguage : langue du CV TAILORÉ (POST /cv/generate, JSON comme PDF).
// QUOI : fr/en/zh natifs ; toute autre (de/it, vide, inconnue) → anglais, jamais le français.
// POURQUOI : zh a désormais libellés, texte ATS et consigne de prompt chinois — le PDF zh est
// entièrement chinois (rendu CJK vérifié : polices Noto CJK dans l'image Docker).
func (l *LocalizationHelper) TailoredLanguage(lang string) string {
	if IsPDFLanguage(lang) {
		return lang
	}
	return l.GetDefaultLanguage()
}
