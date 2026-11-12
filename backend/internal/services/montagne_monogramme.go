package services

import (
	"strings"
	"unicode"
)

// monogramme — les deux lettres qui tiennent lieu d'icône tant qu'Eikon n'a pas produit la vraie
// (champ `icon_url` du nœud, vide pour l'instant).
//
// POURQUOI côté serveur : la règle est testée une fois, ici, et le front (SVG de la liste comme
// dessin du canvas) affiche la même chose partout. Déterministe : le nom seul décide.
//
// COMMENT, sur le nom court (sans sous-titre ni parenthèse) :
//  1. plusieurs mots → initiales des deux premiers mots porteurs (« Anthropic Vendor Proxy » →
//     « AV ») ; les mots-outils (to, of, the…) sont sautés (« Video to MP3 » → « VM ») ;
//  2. un seul mot en CamelCase → ses deux premières capitales, la première lettre comptant comme
//     capitale (« GroveEngine » → « GE », « ladderKeg » → « LK ») ;
//  3. sinon → initiale majuscule + lettre suivante minuscule (« Drifterra » → « Dr »).
func monogramme(nom string) string {
	court := nomIdentite(nom)
	if i := strings.Index(court, "("); i > 0 {
		court = strings.TrimSpace(court[:i])
	}
	mots := strings.FieldsFunc(court, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var porteurs []string
	for _, m := range mots {
		if !motsOutils[strings.ToLower(m)] {
			porteurs = append(porteurs, m)
		}
	}
	if len(porteurs) == 0 {
		porteurs = mots
	}
	if len(porteurs) == 0 {
		return "?"
	}

	if len(porteurs) >= 2 {
		return strings.ToUpper(string(premiereLettre(porteurs[0])) + string(premiereLettre(porteurs[1])))
	}

	runes := []rune(porteurs[0])
	capitales := []rune{unicode.ToUpper(runes[0])}
	for _, r := range runes[1:] {
		if unicode.IsUpper(r) {
			capitales = append(capitales, r)
			break
		}
	}
	if len(capitales) == 2 {
		return string(capitales)
	}
	if len(runes) == 1 {
		return string(capitales)
	}
	return string(capitales[0]) + strings.ToLower(string(runes[1]))
}

// nomIdentite — le nom affiché sous un nœud : nomProjetCourt (coupe sur un tiret entouré
// d'espaces), puis coupe aussi sur le premier « : » HORS parenthèses.
//
// POURQUOI une règle de plus que le chat : mesuré sur les fiches réelles, une dizaine de noms
// portent leur sous-titre après deux-points (« Theomen: Procedural geological world generator »,
// « AVEG: Self-hosted… »). Sous un nœud, seule l'identité tient. Le chat garde nomProjetCourt
// tel quel : ce n'est pas le même affichage, et son comportement est verrouillé ailleurs.
// COMMENT : on ignore un « : » à l'intérieur de parenthèses — « DAOS (Dawi: Age of Stone): … »
// garde « DAOS (Dawi: Age of Stone) ». Jamais de chaîne vide.
func nomIdentite(nom string) string {
	court := nomProjetCourt(nom)
	profondeur := 0
	for i, r := range court {
		switch r {
		case '(':
			profondeur++
		case ')':
			if profondeur > 0 {
				profondeur--
			}
		case ':':
			if profondeur == 0 && i > 0 && strings.HasPrefix(court[i:], ": ") {
				if tete := strings.TrimSpace(court[:i]); tete != "" {
					return tete
				}
			}
		}
	}
	return court
}

// motsOutils — mots qui ne portent pas l'identité d'un nom.
var motsOutils = map[string]bool{"to": true, "of": true, "the": true, "and": true, "for": true, "a": true, "de": true, "la": true, "le": true}

func premiereLettre(mot string) rune { return []rune(mot)[0] }
