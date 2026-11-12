package services

import (
	"math"
	"strings"

	"maicivy/internal/models"
)

// Classement final des projets du CV (refonte du 2026-10-11).
//
// QUOI : score final = pertinence × (base + poids_preuve·preuve + poids_activité·activité).
//
// POURQUOI la pertinence MULTIPLIE au lieu de s'additionner : un projet hors sujet ne doit jamais
// être sauvé par sa taille. Un énorme projet sans rapport (pertinence 0,2) reste sous un petit
// projet pertinent (0,8) ; à pertinence comparable, celui qui est en production, de taille réelle
// et vivant passe devant. C'est exactement ce qui manquait : le modèle classait par mots-clés et
// ne savait pas qu'un projet tournait en prod.
//
// POURQUOI dans le CODE et pas dans le prompt : la preuve se mesure (statut, lignes, tests,
// commits). La confier au modèle la rendait non déterministe et invérifiable ; ici elle est
// testée et identique d'un appel à l'autre.
//
// COMMENT les poids : la base (0,55) garantit qu'un projet très pertinent sans preuve ni activité
// garde plus de la moitié de sa note ; preuve et activité ajoutent au plus 0,45 à eux deux. La
// preuve pèse le double de l'activité — elle dit que la chose existe, l'activité seulement qu'on y
// travaille.
const (
	poidsBase     = 0.55
	poidsPreuve   = 0.30
	poidsActivite = 0.15

	// Composition de la preuve (somme = 1).
	partProduction = 0.5
	partLignes     = 0.3
	partTests      = 0.2

	// Bornes log des échelles : en dessous du plancher = 0, au plafond = 1. Calées sur le
	// catalogue réel (le plus gros projet ~135 k lignes, ~5,4 k tests) : un outil de 1 000 lignes
	// pèse 0, un système de 100 k lignes ~0,97.
	lignesPlancher = 1_000.0
	lignesPlafond  = 120_000.0
	testsPlancher  = 10.0
	testsPlafond   = 3_000.0

	// facteurRetire : un projet retiré n'est pas effacé (il reste sur le site), mais il ne doit
	// plus prendre la place d'un projet vivant sur le CV. Mesuré : ClaudeDuo, retiré, sortait 1er.
	facteurRetire = 0.25
)

// statutsRetires : statuts maiProFiles d'un projet qui ne tourne plus.
var statutsRetires = map[string]bool{"retired": true, "archived": true, "deprecated": true, "abandoned": true}

// echelleLog ramène v sur 0..1 entre plancher et plafond, en échelle logarithmique (passer de 1 k
// à 10 k lignes compte autant que de 10 k à 100 k), saturée aux bornes.
func echelleLog(v, plancher, plafond float64) float64 {
	if v <= plancher {
		return 0
	}
	if v >= plafond {
		return 1
	}
	return math.Log(v/plancher) / math.Log(plafond/plancher)
}

// preuveProjet : 0..1 — ce qui prouve que le projet existe vraiment (production, ampleur, tests).
func preuveProjet(p models.Project) float64 {
	prod := 0.0
	if strings.EqualFold(p.Status, "production") {
		prod = 1
	}
	return partProduction*prod +
		partLignes*echelleLog(float64(p.LOC), lignesPlancher, lignesPlafond) +
		partTests*echelleLog(float64(p.TestCount), testsPlancher, testsPlafond)
}

// scoreFinalProjet : pertinence (0..1) combinée à la preuve et à l'activité du projet.
func scoreFinalProjet(pertinence float64, p models.Project) float64 {
	activite := math.Max(0, math.Min(1, p.Activity))
	s := pertinence * (poidsBase + poidsPreuve*preuveProjet(p) + poidsActivite*activite)
	if statutsRetires[strings.ToLower(p.Status)] {
		s *= facteurRetire
	}
	return s
}

// pertinencesDeRepli : quand le LLM est indisponible, la pertinence vient des poids de tags du
// thème, NORMALISÉS sur le meilleur projet (les scores de tags bruts plafonnent vers 0,3 et
// écraseraient toute la combinaison). Le repli est journalisé par l'appelant.
func pertinencesDeRepli(scoresTags map[string]float64) map[string]float64 {
	max := 0.0
	for _, v := range scoresTags {
		if v > max {
			max = v
		}
	}
	out := make(map[string]float64, len(scoresTags))
	for id, v := range scoresTags {
		if max > 0 {
			out[id] = v / max
		}
	}
	return out
}
