package services

import (
	"fmt"
	"strings"
)

// Inventaire des projets injecté dans le prompt système du chat.
//
// POURQUOI ce bloc existe : sans lui, l'agent ne connaît AUCUN projet au démarrage. Une question
// aussi banale que « qu'est-ce qu'il a fait en Rust ? » imposait un aller-retour d'outil avant de
// pouvoir seulement énumérer ce qui existe — et l'agent ne pouvait rien PROPOSER de lui-même,
// puisqu'il ignorait le terrain. Avec l'inventaire en contexte, il sait quoi mettre en avant et ne
// va chercher le détail que du projet qu'il va réellement développer.
//
// POURQUOI un rendu TEXTE et pas le JSON de l'API : mesuré sur les 79 projets réels, le JSON de
// /projects/index pèse ~9 600 tokens contre ~5 600 pour ce rendu — 4 000 tokens d'accolades, de
// guillemets et de noms de clés répétés 79 fois, payés à CHAQUE tour de conversation.

// indexMiniMax : garde-fou de longueur sur une ligne d'inventaire.
//
// Les `mini` rédigés côté maiProFiles tiennent aujourd'hui en ~102 caractères (médiane) et ne sont
// jamais tronqués. Cette borne protège d'une dérive éditoriale future — une fiche dont la « ligne »
// deviendrait un paragraphe ne doit pas gonfler le prompt de tout le monde en silence.
const indexMiniMax = 200

// renderProjectsIndex rend l'inventaire en lignes compactes, une par projet.
//
// Format : `id | Nom — mini`. Rien d'autre (choix d'Alexi, et sa conception le justifie).
//
// POURQUOI ni la stack, ni la catégorie, ni le statut : le `mini` est rédigé côté maiProFiles comme
// « quoi + techno structurante + ce qui distingue » — la technologie y est DÉJÀ. Réémettre `stack`
// à côté doublonnait l'information tout en pesant 35 % du bloc (mesuré sur les 79 fiches réelles),
// payés à chaque tour de conversation. L'agent qui a besoin du détail lit la fiche.
//
// L'ID vient EN PREMIER parce que c'est ce que l'agent doit repasser pour demander le détail : le
// mettre en tête évite qu'il fabrique un identifiant à partir du nom affiché.
//
// Retourne une chaîne VIDE si l'index est vide ou indisponible. L'appelant n'ajoute alors aucun
// en-tête : un titre « Projets » suivi de rien ferait conclure à l'agent qu'il n'y a pas de projet,
// et il répondrait « il n'a rien fait » avec aplomb.
func renderProjectsIndex(entries []MPFProjectIndexEntry) string {
	return renderProjectsIndexAvecPages(entries, nil)
}

// marquePage : suffixe des projets qui ont une page publique sur le site (/fr/portfolio/<id>).
// POURQUOI dans l'inventaire : le Composer doit lier la page du projet cité plutôt que l'accueil
// (cf. checkProjectLink). L'agent ne peut pas savoir seul qu'une page existe, et en inventer une
// produit un lien mort. Une marque courte, sur ~25 lignes : quelques dizaines de tokens par tour.
const marquePage = " [page]"

// renderProjectsIndexAvecPages : comme renderProjectsIndex, en marquant [page] les projets dont
// l'identifiant figure dans pages (nil = aucune marque).
func renderProjectsIndexAvecPages(entries []MPFProjectIndexEntry, pages map[string]bool) string {
	lignes := make([]string, 0, len(entries))
	for _, e := range entries {
		mini := strings.TrimSpace(e.Mini)
		// Sans `mini`, la ligne n'apprend rien qui permette de CHOISIR : on l'écarte plutôt que
		// d'occuper des tokens avec un nom nu.
		if mini == "" {
			continue
		}
		if r := []rune(mini); len(r) > indexMiniMax {
			mini = strings.TrimSpace(string(r[:indexMiniMax]))
		}
		// Marque de PRODUCTION — le seul statut rendu, et la seule preuve de réalité disponible.
		//
		// POURQUOI ce champ a été réintégré : mesuré sur les 79 fiches, UN SEUL mini dit « en
		// production » alors que 39 projets le sont. La métadonnée le sait, la prose ne le dit pas.
		// Sans elle, l'agent ne dispose que de comptages (modules, tests, étapes) pour étayer — or un
		// comptage prouve l'effort, pas que la chose tourne pour quelqu'un. C'est la différence entre
		// « j'ai construit ça » et « ça tourne », et c'est la seconde qui enlève le risque au client.
		//
		// POURQUOI seulement « production » : `active`, `wip` et `R&D` n'apportent aucune preuve — et
		// « wip » en dirait du mal. Les rendre coûterait des tokens sur chaque message pour desservir.
		marque := ""
		if e.Status == "production" {
			marque = " [production]"
		}
		if pages[e.ID] {
			marque += marquePage
		}
		lignes = append(lignes, fmt.Sprintf("%s | %s%s — %s", e.ID, nomProjetCourt(e.Name), marque, mini))
	}

	if len(lignes) == 0 {
		return ""
	}
	return strings.Join(lignes, "\n")
}

// separateursAccroche : séparateurs qui, ENTOURÉS D'ESPACES, introduisent une accroche derrière le
// nom d'un projet. L'espace est déterminant : `wechat-homework-bot` ou `e-commerce` portent un tiret
// qui fait partie du nom, alors que « Aria – Assistant IA desktop » porte une description.
var separateursAccroche = []string{" — ", " – ", " - "}

// nomProjetCourt réduit un nom à son identité, en retirant le sous-titre qui le suit.
//
// POURQUOI — le partage des rôles, et lui seul : le NOM sert à DÉSIGNER le projet devant
// l'utilisateur, le MINI (gardé complet) sert à le DÉCRIRE. Le mini est le champ rédigé pour ça côté
// maiProFiles ; le sous-titre du nom fait le même travail en moins bien et en double emploi.
//
// Mesuré sur les 79 fiches live : 69 noms portent un sous-titre, pesant 3 434 caractères (~980
// tokens) sur un bloc qui repart à CHAQUE tour.
//
// ⚠️ NUANCE, corrigée après vérification croisée avec la session maiProFiles : ces 69 sous-titres ne
// sont PAS « repris mot pour mot par le mini » — c'est ce que disait la première version de ce
// commentaire, et c'était faux. La redondance TEXTUELLE (préfixe d'au moins deux mots commun avec le
// début du mini) ne concerne que 32 fiches, soit 712 caractères. Le reste est de la description non
// dupliquée. La coupe reste justifiée par le partage des rôles ci-dessus, pas par la redondance.
//
// Ne renvoie jamais une chaîne vide : un nom qui ne serait QUE de l'accroche est laissé tel quel,
// mieux vaut un nom bavard qu'un projet anonyme dans une liste de 79.
func nomProjetCourt(nom string) string {
	for _, sep := range separateursAccroche {
		if i := strings.Index(nom, sep); i > 0 {
			if court := strings.TrimSpace(nom[:i]); court != "" {
				return court
			}
		}
	}
	return nom
}
