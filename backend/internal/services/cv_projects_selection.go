package services

import (
	"strings"
)

// Sélection des projets affichés dans le PDF.
//
// CONTEXTE (mesuré le 2026-09-07 sur la prod) : le CV déroulait le catalogue entier — 79 projets,
// dont 73 en grille compacte, occupant 4 pages d'un PDF de 6. Les projets sont triés par score
// décroissant : cette queue est donc, par construction, ce qui correspond le MOINS à l'offre. On
// dépensait les deux tiers du document sur ce qui compte le moins.
//
// La règle retenue est un SEUIL sur le score, encadré par un plancher et un plafond. Le seuil
// seul ne suffit pas : un score n'est un signal que si le scoring discrimine. Quand il ne
// discrimine plus (tous les projets à la même valeur, ce qui était l'état réel en prod), le
// seuil laisse tout passer ou tout tomber — d'où les deux bornes, qui gardent le document
// utilisable dans les deux cas.
const (
	projetsDetailles = 6 // cartes détaillées en tête (inchangé)

	// seuilProjetCompact : score minimal pour figurer dans la grille compacte. Sous ce seuil,
	// le projet n'apparaît plus dans le PDF (il reste sur le site, qui n'a pas de pagination).
	seuilProjetCompact = 0.40

	// compactsMin : plancher. Si le scoring est en panne et note tout au plancher, le seuil
	// couperait TOUT — un CV sans aucun projet est pire qu'un CV trop long.
	compactsMin = 4

	// compactsMax : plafond. Il s'applique même quand tous les projets passent le seuil, ce qui
	// est le cas dégénéré d'un scoring à plat.
	// 18 → 6 → 4 le 2026-10-11 : avec 18, le CV faisait 3 pages, dont une aux trois quarts vide,
	// et noyait les projets forts. Avec 6, la 2e page se remplissait à 90 % : selon la longueur
	// des accroches, une rangée basculait sur une 3e page (mesuré en prod). 6 détaillés + 4
	// compacts laisse une marge ; les guides de recrutement visent 4 à 6 projets prouvés.
	compactsMax = 4

	// budgetStripATS : taille maximale du strip ATS du header, en caractères.
	//
	// Le strip est rogné par sa boîte (.header-stealth : 16px de haut, police 5px, soit trois
	// lignes). Au-delà, Chromium émet TOUJOURS le texte dans le flux PDF, mais en laissant
	// tomber des glyphes : le flux finit en "P j tT i O ( t l di d k )". Un strip plus long ne
	// donne donc pas plus de mots-clés à l'ATS, il lui donne de la bouillie.
	//
	// Mesuré le 2026-09-07 avec une sonde de jetons numérotés (T000xx, T001xx, …) : la
	// dernière entrée intacte tombe au caractère ~861. On garde une marge, parce que la limite
	// dépend de la largeur des glyphes, donc du texte : un strip de 842 caractères de mots
	// réels était déjà tronqué en fin de chaîne.
	// zh (mesuré le 2026-10-16, même sonde, préfixe chinois de buildStealthText) : jetons intacts
	// jusqu'à l'octet ~1075. Le budget est compté en OCTETS (len) : un idéogramme vaut 3 octets
	// pour ~2 largeurs latines, donc 780 reste sous la limite CJK, avec plus de marge qu'en FR.
	// ⚠️ Ce budget est lié à la HAUTEUR et à la POLICE de .header-stealth : changer l'une des
	// deux dans cv_base.html déplace la limite, et la sonde ci-dessus est là pour la remesurer.
	budgetStripATS = 780
)

// selectionnerProjets répartit les projets (déjà triés par score décroissant) en trois lots :
// les cartes détaillées, la grille compacte, et ce qui est écarté du PDF.
//
// COMMENT : les `projetsDetailles` premiers passent en détaillé. Parmi le reste, on garde ceux
// au-dessus du seuil, plafonnés à `compactsMax` ; si le seuil n'en laisse pas assez, on complète
// jusqu'à `compactsMin` en suivant l'ordre de score (donc les moins mauvais d'abord).
func selectionnerProjets(projets []ScoredProjectResponse) (detailles, compacts, ecartes []ScoredProjectResponse) {
	return selectionnerProjetsPlafond(projets, compactsMax)
}

// selectionnerProjetsPlafond : même règle que selectionnerProjets, avec le plafond de la grille
// compacte fixé par l'appelant.
//
// POURQUOI : le CV taillé doit tenir en 2 pages (cf. ajusterAuxPages). Quand il déborde, on
// retire des rangées de la grille compacte — les projets les moins pertinents du document —
// plutôt que de tronquer du contenu. Le plancher suit le plafond (min des deux) : sinon
// compactsMin remettrait les projets qu'on vient de retirer. L'ordre des scores n'est pas
// touché ; les projets retirés passent dans `ecartes`, donc leurs technos rejoignent le strip ATS.
func selectionnerProjetsPlafond(projets []ScoredProjectResponse, plafond int) (detailles, compacts, ecartes []ScoredProjectResponse) {
	if len(projets) <= projetsDetailles {
		return projets, nil, nil
	}
	detailles = projets[:projetsDetailles]
	candidats := projets[projetsDetailles:]

	garde := 0
	for _, p := range candidats {
		if p.Score < seuilProjetCompact || garde >= plafond {
			break // la liste est triée : le premier qui échoue clôt la sélection
		}
		garde++
	}
	if plancher := min(compactsMin, plafond); garde < plancher {
		garde = plancher
	}
	if garde > len(candidats) {
		garde = len(candidats)
	}

	return detailles, candidats[:garde], candidats[garde:]
}

// enrichirStripAvecEcartes complète le strip ATS avec les technologies des projets écartés qui
// n'apparaissent nulle part ailleurs dans le strip.
//
// POURQUOI seulement les technos : couper des projets retire leurs mots-clés du PDF. Mais ce
// qu'un ATS cherche, ce sont des technologies, pas des noms de projets — et le budget du strip
// est trop étroit (voir budgetStripATS) pour y verser 60 lignes. On récupère donc le seul
// contenu qui a une valeur de correspondance, et uniquement s'il est absent du reste.
//
// COMMENT : parcours dans l'ordre des scores (les écartés arrivent déjà triés), ajout d'une
// techno seulement si elle n'est ni déjà dans le strip ni déjà ajoutée, arrêt net dès que le
// budget est atteint. Un strip vide reste vide : GenerateCVPDF n'en injecte aucun, et en
// fabriquer un ici ajouterait du texte invisible là où l'appelant n'en voulait pas.
func enrichirStripAvecEcartes(strip string, ecartes []ScoredProjectResponse) string {
	if strip == "" || len(ecartes) == 0 {
		return strip
	}

	dejaVu := strings.ToLower(strip)
	var ajouts []string
	for _, p := range ecartes {
		for _, techno := range p.Technologies {
			t := strings.TrimSpace(techno)
			if t == "" || strings.Contains(dejaVu, strings.ToLower(t)) {
				continue
			}
			// +2 pour le séparateur ", " ; on s'arrête avant de dépasser, jamais après.
			if len(strip)+len(strings.Join(ajouts, ", "))+len(t)+2 > budgetStripATS {
				return assembleStrip(strip, ajouts)
			}
			ajouts = append(ajouts, t)
			dejaVu += " " + strings.ToLower(t)
		}
	}
	return assembleStrip(strip, ajouts)
}

// assembleStrip recolle le strip d'origine et les technos récupérées.
func assembleStrip(strip string, ajouts []string) string {
	if len(ajouts) == 0 {
		return strip
	}
	return strip + " " + strings.Join(ajouts, ", ") + "."
}
