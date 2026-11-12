package services

import (
	"sort"
	"time"
)

// ordonnerExperiencesCV range les expériences d'un CV dans l'ordre qu'un lecteur attend :
// chronologique inverse.
//
// QUOI : les postes EN COURS d'abord (pas de date de fin), puis par date de fin décroissante ;
// à fin égale (ou tous deux en cours), la plus récemment commencée d'abord.
//
// POURQUOI pas le score de thème : un CV se lit comme une chronologie. Trié par pertinence, le PDF
// de prod (thème ai, 28/10/2026) montrait une mission terminée en 03/2026 AVANT la mission en
// cours depuis 04/2026 — celle même que l'accroche met en avant. Le score reste calculé et
// renvoyé (le site l'affiche en pourcentage) ; il ne décide simplement plus de l'ordre.
//
// COMMENT : tri stable, pour qu'à dates identiques l'ordre du catalogue maiProFiles soit conservé
// d'un appel à l'autre (jamais un ordre aléatoire). Une date de fin absente vaut « +∞ ».
func ordonnerExperiencesCV(e []ScoredExperienceResponse) {
	fin := func(i int) time.Time {
		if e[i].EndDate == nil {
			return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		return *e[i].EndDate
	}
	sort.SliceStable(e, func(i, j int) bool {
		fi, fj := fin(i), fin(j)
		if !fi.Equal(fj) {
			return fi.After(fj)
		}
		return e[i].StartDate.After(e[j].StartDate)
	})
}
