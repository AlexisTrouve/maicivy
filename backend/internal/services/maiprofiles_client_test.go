package services

import "testing"

// TestLangSuffix verrouille le repli de langue du client maiProFiles.
// CONTEXTE : maiProFiles ne contient du contenu QUE pour fr/en/ka. Le front expose fr/en/de/it/zh.
// AVANT : une langue non servie (de/it/zh) ne recevait PAS de param ?lang= → maiProFiles servait son
// défaut FRANÇAIS → un visiteur allemand voyait des fiches en français (incohérent avec l'UI/voix DE).
// MAINTENANT (décision B) : une langue non servie ET non vide replie sur l'ANGLAIS (neutre/international).
// EXCEPTION : lang="" reste le « défaut API » (FR) — utilisé par des appels internes non liés à une
// locale visiteur (stats numériques, profil du prompt système) qu'on ne veut pas modifier.
func TestLangSuffix(t *testing.T) {
	cases := []struct {
		name string
		path string
		lang string
		want string
	}{
		// Langues réellement servies par maiProFiles → passées telles quelles.
		{"fr servi tel quel", "/projects", "fr", "/projects?lang=fr"},
		{"en servi tel quel", "/projects", "en", "/projects?lang=en"},
		{"ka servi tel quel", "/projects", "ka", "/projects?lang=ka"},

		// Langues du front non servies par maiProFiles → repli ANGLAIS (et surtout PAS le défaut FR).
		{"de → fallback en", "/projects", "de", "/projects?lang=en"},
		{"it → fallback en", "/projects", "it", "/projects?lang=en"},
		{"zh → fallback en", "/projects", "zh", "/projects?lang=en"},

		// lang vide = défaut API (FR) pour les appels internes → aucun param ajouté.
		{"vide → défaut API (pas de param)", "/projects", "", "/projects"},

		// Séparateur correct quand le path porte déjà des query params.
		{"non servi + query existante → &", "/blog/posts?page=1&per_page=10", "de", "/blog/posts?page=1&per_page=10&lang=en"},
		{"servi + query existante → &", "/blog/posts?page=1", "fr", "/blog/posts?page=1&lang=fr"},
		{"vide + query existante → inchangé", "/blog/posts?page=1", "", "/blog/posts?page=1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := langSuffix(tc.path, tc.lang)
			if got != tc.want {
				t.Errorf("langSuffix(%q, %q) = %q, want %q", tc.path, tc.lang, got, tc.want)
			}
		})
	}
}
