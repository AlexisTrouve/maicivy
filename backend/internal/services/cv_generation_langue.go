package services

import (
	"strings"
	"unicode"
)

// Garde de LANGUE sur ce que le modèle rédige pour le CV taillé (job_title, summary, catchphrase).
//
// CONTEXTE (mesuré sur la prod le 11/11/2026) : un CV demandé en anglais sortait avec un résumé et
// trois accroches en français. La consigne de langue du prompt (consigneLangueSortie) est la
// première défense ; celle-ci est la seconde, côté réponse : un modèle peut ignorer une consigne,
// et ce texte part sous le nom d'Alexis chez un recruteur.
//
// COMMENT : détection par MOTS-OUTILS (articles, prépositions, pronoms), propres à une langue.
// Ce sont les mots qu'une phrase ne peut pas éviter et qu'aucun nom de technologie ne contient :
// « Python, Docker, Node.js » ne vote pour rien, « Agents IA qui livrent du code » vote 2 fois
// pour le français. On n'écarte qu'avec une preuve POSITIVE (seuil + majorité) : un titre fait de
// noms propres et de technos (« Senior AI Engineer — MCP ») reste, dans toutes les langues.

// seuilMarqueursLangue : nombre minimal de mots-outils d'une autre langue pour conclure. Un seul
// peut être un emprunt ; deux, dans un texte de 80 caractères, sont une phrase.
const seuilMarqueursLangue = 2

// marqueursLangue : mots-outils propres à chaque langue latine du CV. Les mots ambigus entre fr
// et en sont EXCLUS (« on », « a », « as », « plus », « son », « or », « an », « via ») — un
// marqueur partagé ferait voter les deux langues. Les élisions françaises (d', l', qu', j')
// arrivent ici comme mots d'une lettre, parce que le découpage se fait sur l'apostrophe ; « s »,
// « t », « c », « n » n'y sont pas : l'anglais les produit aussi (it's, don't).
var marqueursLangue = map[string]map[string]bool{
	"fr": ensembleDeMots("le", "la", "les", "des", "du", "de", "et", "est", "sont", "une", "un", "qui", "que",
		"pour", "avec", "dans", "sur", "par", "pas", "je", "nous", "vous", "au", "aux", "ce", "cette", "ces",
		"mon", "ma", "mes", "sa", "ses", "leur", "leurs", "où", "mais", "ou", "en", "à", "chez", "sans", "sous",
		"vers", "d", "l", "qu", "j"),
	"en": ensembleDeMots("the", "and", "of", "to", "for", "with", "in", "is", "are", "that", "which", "who",
		"i", "my", "by", "from", "at", "it", "this", "these", "those", "we", "you", "your", "not", "than",
		"into", "its", "our", "their", "be", "been", "have", "has", "was", "were", "will", "can", "while",
		"through", "but", "what", "where", "when"),
}

func ensembleDeMots(mots ...string) map[string]bool {
	m := make(map[string]bool, len(mots))
	for _, w := range mots {
		m[w] = true
	}
	return m
}

// empreinteLangue compte, pour un texte, les mots-outils de chaque langue latine et les
// idéogrammes (zh).
func empreinteLangue(texte string) (latin map[string]int, ideogrammes int) {
	latin = map[string]int{}
	for _, r := range texte {
		if unicode.Is(unicode.Han, r) {
			ideogrammes++
		}
	}
	// Découpe sur tout ce qui n'est pas une lettre : ponctuation, chiffres, apostrophes.
	mots := strings.FieldsFunc(strings.ToLower(texte), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, w := range mots {
		for lang, ens := range marqueursLangue {
			if ens[w] {
				latin[lang]++
			}
		}
	}
	return latin, ideogrammes
}

// redigeHorsLangue : vrai si `texte` est, avec une preuve positive, rédigé dans une AUTRE langue
// que `lang` (fr, en ou zh). Un texte sans preuve (vide, noms propres, technos) n'est jamais
// écarté.
func redigeHorsLangue(texte, lang string) bool {
	latin, ideogrammes := empreinteLangue(texte)
	if lang == "zh" {
		// Un texte chinois porte des idéogrammes ; sans aucun, deux mots-outils latins suffisent.
		return ideogrammes == 0 && (latin["fr"] >= seuilMarqueursLangue || latin["en"] >= seuilMarqueursLangue)
	}
	if ideogrammes >= seuilMarqueursLangue {
		return true
	}
	for autre, n := range latin {
		if autre != lang && n >= seuilMarqueursLangue && n > latin[lang] {
			return true
		}
	}
	return false
}

// gardeLangueRedaction retire de la réponse du modèle chaque texte rédigé hors de la langue du CV,
// et rend la liste des champs écartés (pour le journal).
//
// POURQUOI retirer plutôt que garder : chaque champ a un repli DANS la bonne langue — l'accroche
// maiProFiles de l'expérience (déjà localisée), le titre par défaut (titreCVTailleParDefaut) ;
// le résumé, lui, disparaît, et un CV sans résumé reste un CV, alors qu'un paragraphe français en
// tête d'un CV anglais le disqualifie.
func gardeLangueRedaction(r *llmGenerationResponse, lang string) []string {
	var ecartes []string
	if redigeHorsLangue(r.JobTitle, lang) {
		ecartes = append(ecartes, "job_title")
		r.JobTitle = ""
	}
	if redigeHorsLangue(r.Summary, lang) {
		ecartes = append(ecartes, "summary")
		r.Summary = ""
	}
	for i := range r.Experiences {
		if redigeHorsLangue(r.Experiences[i].Catchphrase, lang) {
			ecartes = append(ecartes, "catchphrase:"+r.Experiences[i].Slug)
			r.Experiences[i].Catchphrase = ""
		}
	}
	return ecartes
}

// titreCVTailleParDefaut : titre du bandeau quand le modèle n'en fournit pas (ou qu'il a été
// écarté par la garde), dans la langue du CV. Avant : « CV Dynamique » en dur, donc du français
// sur un CV anglais.
var titreCVTailleParDefaut = map[string]string{
	"fr": "CV sur mesure",
	"en": "Tailored CV",
	"zh": "定制简历",
}
