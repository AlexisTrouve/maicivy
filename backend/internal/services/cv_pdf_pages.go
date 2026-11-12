package services

import (
	"regexp"

	"github.com/rs/zerolog/log"
)

// pagesCVTaille : nombre de pages du CV taillé (POST /cv/generate) — celui de l'export standard,
// mesuré pour les six thèmes en fr et en.
const pagesCVTaille = 2

// rangeeCompacts : nombre de projets d'une rangée de la grille compacte (.other-projects-grid,
// deux colonnes). On retire par rangée entière : une demi-rangée ne libère aucune hauteur.
const rangeeCompacts = 2

// motifObjetPage reconnaît le dictionnaire d'UNE page (« /Type /Page »), pas celui de l'arbre
// des pages (« /Type /Pages ») : `\b` exige une frontière de mot après « Page ».
var motifObjetPage = regexp.MustCompile(`/Type\s*/Page\b`)

// compterPagesPDF rend le nombre de pages d'un PDF produit par htmlToPDF.
//
// QUOI : compte les objets page du fichier.
// POURQUOI sans bibliothèque PDF : le PDF vient TOUJOURS de Chromium (Skia/PDF), qui écrit ses
// dictionnaires en clair, sans flux d'objets compressés (vérifié sur les PDF de prod, 11/11/2026).
// Ajouter une dépendance pour lire un entier que l'on produit soi-même serait disproportionné.
// COMMENT : pas valable pour un PDF quelconque venu de l'extérieur — uniquement pour les nôtres.
func compterPagesPDF(pdf []byte) int {
	return len(motifObjetPage.FindAll(pdf, -1))
}

// ajusterAuxPages rend un CV d'au plus `pagesMax` pages en réduisant la grille compacte.
//
// QUOI : `rendre(plafond)` produit le PDF avec au plus `plafond` projets compacts. On part de
// compactsMax (rendu standard) et on retire une rangée à chaque débordement, jusqu'à zéro.
// POURQUOI la grille compacte : ce sont, par construction, les projets les MOINS pertinents
// pour l'offre (tri par score), et leurs technos restent dans le strip ATS. Le résumé, les
// expériences et les six cartes détaillées ne sont jamais touchés.
// COMMENT : si même sans grille le CV déborde (débordement venu d'ailleurs, par ex. la colonne
// des compétences), on rend le dernier rendu et on le journalise — un CV de trois pages vaut
// mieux qu'aucun CV, mais il ne doit pas passer inaperçu.
func ajusterAuxPages(pagesMax int, rendre func(plafondCompacts int) ([]byte, error)) ([]byte, error) {
	for plafond := compactsMax; ; plafond -= rangeeCompacts {
		plafond = max(plafond, 0)
		pdf, err := rendre(plafond)
		if err != nil {
			return nil, err
		}
		pages := compterPagesPDF(pdf)
		if pages <= pagesMax {
			return pdf, nil
		}
		if plafond == 0 {
			log.Warn().Int("pages", pages).Int("pages_max", pagesMax).
				Msg("[cv-pdf] le CV déborde même sans grille compacte — rendu tel quel")
			return pdf, nil
		}
	}
}
