package services

import "testing"

// Le résumé du CV taillé remplace les tirets cadratins du modèle par des fins de phrase. Une fin de
// phrase suivie d'une minuscule (« en prod. une équipe ») se lit comme une faute sur le CV : la
// lettre qui suit doit passer en majuscule, y compris accentuée.
func TestTiretsEnPhrases(t *testing.T) {
	cas := []struct{ nom, entree, attendu string }{
		{"tiret entouré d'espaces", "Agents en prod — une équipe de cinq.", "Agents en prod. Une équipe de cinq."},
		{"lettre accentuée", "Orchestration — état tenu par le code.", "Orchestration. État tenu par le code."},
		{"tiret collé", "Go en prod—une API.", "Go en prod. Une API."},
		{"tiret final", "Livré en prod —", "Livré en prod."},
		{"plusieurs tirets", "A — b — c", "A. B. C"},
		{"chiffre après", "Freelance — 6 ans d'agents.", "Freelance. 6 ans d'agents."},
		{"sans tiret", "Rien à changer, ici - non plus.", "Rien à changer, ici - non plus."},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := tiretsEnPhrases(c.entree); got != c.attendu {
				t.Errorf("tiretsEnPhrases(%q) = %q, attendu %q", c.entree, got, c.attendu)
			}
		})
	}
}
