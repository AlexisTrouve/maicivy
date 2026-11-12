package services

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// motifTiretCadratin : un tiret cadratin et les espaces qui l'entourent, avec ou sans espaces
// (« prod — une », « prod —une », « prod—une »).
var motifTiretCadratin = regexp.MustCompile(`\s*—\s*`)

// tiretsEnPhrases transforme les tirets cadratins du résumé rédigé par le modèle en fins de phrase.
//
// QUOI : « en prod — une équipe » devient « en prod. Une équipe » ; un tiret final devient un point.
// POURQUOI : le tiret cadratin passe mal dans certains parseurs ATS et se lit comme un tic d'IA ;
// on le remplace par une fin de phrase. L'ancien remplacement brut laissait une MINUSCULE après le
// point (« en prod. une équipe ») — une faute visible en tête du CV, sur le texte le plus lu.
// COMMENT : chaque tiret (espaces compris) devient « . » ; la lettre qui suit passe en majuscule
// (unicode.ToUpper : « é » → « É ») ; un chiffre ou une ponctuation qui suit reste tel quel.
func tiretsEnPhrases(s string) string {
	parts := motifTiretCadratin.Split(s, -1)
	if len(parts) == 1 {
		return s
	}
	var b strings.Builder
	b.WriteString(parts[0])
	for _, p := range parts[1:] {
		if p == "" { // tiret final : on termine la phrase
			b.WriteString(".")
			continue
		}
		b.WriteString(". ")
		r, n := utf8.DecodeRuneInString(p)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(p[n:])
	}
	return b.String()
}
