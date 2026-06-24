package services

import (
	"testing"

	"maicivy/internal/models"
)

// TestNormalizeLanguage_UnsupportedFallsBackToEnglish verrouille le principe de repli du CV (décision B) :
// la langue de l'utilisateur est servie SI maiProFiles a du contenu natif pour elle (fr/en), SINON on
// sert l'ANGLAIS — jamais le français. AVANT : tout ce qui n'était pas fr/en retombait sur "fr" (défaut),
// si bien qu'un visiteur de/it/zh voyait un CV en français malgré une UI dans sa langue.
func TestNormalizeLanguage_UnsupportedFallsBackToEnglish(t *testing.T) {
	l := NewLocalizationHelper()
	cases := map[string]string{
		// Langues avec contenu CV natif → servies telles quelles.
		"fr": "fr",
		"en": "en",
		// Langues du front sans contenu CV natif → repli ANGLAIS (pas français).
		"de": "en",
		"it": "en",
		"zh": "en",
		// Valeurs vides / inconnues → anglais par défaut.
		"":   "en",
		"xx": "en",
	}
	for in, want := range cases {
		if got := l.NormalizeLanguage(in); got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLocalizeExperience_GermanVisitorServedInEnglish prouve le bout-en-bout : un visiteur DE (langue
// non servie) reçoit les champs EN d'une expérience, pas les champs FR. La langue est d'abord normalisée
// (de→en) PUIS localisée — exactement l'ordre de CVService.GetAdaptiveCV.
func TestLocalizeExperience_GermanVisitorServedInEnglish(t *testing.T) {
	l := NewLocalizationHelper()
	exp := models.Experience{
		Title:         "Titre FR",
		TitleEn:       "EN Title",
		Description:   "description fr",
		DescriptionEn: "english description",
	}

	got := l.LocalizeExperience(exp, l.NormalizeLanguage("de"))

	if got.Title != "EN Title" {
		t.Errorf("Title : got %q, want %q (un visiteur DE doit voir l'anglais)", got.Title, "EN Title")
	}
	if got.Description != "english description" {
		t.Errorf("Description : got %q, want %q", got.Description, "english description")
	}
}
