package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/models"
)

// QUOI : verrouille la calibration de VOIX partagée entre le générateur de messages plateforme et
// l'agent de l'écran /admin/compose.
//
// POURQUOI : l'écran compose a d'abord été branché sur le prompt système du chat — calibré pour
// répondre à des questions sur le CV, pas pour écrire à la voix d'Alexi. Résultat rapporté en usage
// réel le 2026-07-31 : « tu as complètement cassé la voix ». Le bloc de style (few-shot de sa prose
// réelle, règles de traduction vers le registre pro, mots interdits, interdiction des tirets) existait
// pourtant déjà dans prompts_platform_message.go — il n'avait simplement jamais été reporté.
//
// Il a donc UNE seule source de vérité désormais, et ces tests interdisent qu'un des deux chemins la
// perde en silence : le symptôme (« ça ne sonne plus comme moi ») est invisible pour un opérateur qui
// ne lit pas le français, et coûteux à diagnostiquer.

// marqueurs présents dans le bloc de voix. Chacun porte une contrainte distincte : perdre l'un d'eux
// dégrade le rendu d'une façon différente.
var voixMarqueurs = []string{
	"EXEMPLES DE MON VRAI STYLE D'ÉCRITURE",  // few-shot de la prose réelle
	"MOTS ET FORMULES STRICTEMENT INTERDITS", // liste des tics à bannir
	"passionné",                              // un des mots interdits — la liste doit être là
	"RÈGLES ABSOLUES",                        // contraintes de forme
	"Jamais de tirets",                       // la plus visible dans le rendu final
}

func TestVoixPlatform_ContientLesContraintes(t *testing.T) {
	for _, m := range voixMarqueurs {
		assert.Contains(t, platformVoiceFR, m, "contrainte de voix perdue à l'extraction")
	}
}

// Non-régression du générateur d'origine : l'extraction du bloc ne doit rien lui avoir retiré.
func TestVoixPlatform_GenerateurOrigineIntact(t *testing.T) {
	pb := &PromptBuilder{userProfile: models.UserProfile{Name: "Alexis Trouve"}}

	prompt := pb.BuildPlatformMessagePrompt(models.PlatformMessageRequest{
		Mission: "Recherche freelance IA pour industrialiser des POC.", Platform: "malt", Lang: "fr",
	})

	for _, m := range voixMarqueurs {
		assert.Contains(t, prompt, m)
	}
	// Les sections interpolées doivent toujours être là : le découpage ne doit pas avoir cassé le
	// template autour du bloc extrait.
	assert.Contains(t, prompt, "PROFIL :")
	assert.Contains(t, prompt, "ANNONCE :")
	assert.Contains(t, prompt, "STRUCTURE :")
	assert.Contains(t, prompt, "Recherche freelance IA")
	assert.NotContains(t, prompt, "%!", "verbe de format non consommé — le template est désaligné")
}

// L'agent compose doit recevoir la MÊME voix, plus ses contraintes propres.
func TestVoixCompose_ReprendLaVoixEtSesRegles(t *testing.T) {
	guidance := composeVoiceGuidance()

	for _, m := range voixMarqueurs {
		assert.Contains(t, guidance, m, "l'agent compose n'a pas la calibration de voix")
	}

	// Règle permanente : l'ADRESSE du portfolio doit figurer dans le message.
	//
	// Ce n'est PLUS « cite maicivy » (décision d'Alexi, 2026-08-01) : le projet maicivy n'a aucune
	// raison d'être cité dans tous les messages — l'agent dispose désormais de l'inventaire complet
	// et doit choisir le projet le plus proche du besoin. C'est le LIEN qui convertit, pas le nom.
	assert.Contains(t, strings.ToLower(guidance), "maicivy.etheryale.com",
		"l'adresse du portfolio doit être exigée explicitement")

	// L'agent écrit via un tool, pas dans sa réponse texte : sans ça il rend le message dans le chat
	// et le draft reste vide.
	assert.Contains(t, guidance, "set_draft")
}

// Le brief des captures d'écran. L'opératrice colle un screenshot du formulaire de la plateforme et
// l'agent répond directement — il n'y a pas de tool, la vision est native.
//
// Les deux règles verrouillées ici sont celles dont l'absence se paierait chez un client :
//   - INTERDICTION D'INVENTER : l'opératrice ne relit pas le français, une réponse plausible mais
//     fausse partirait telle quelle. « Demande à Alexis » est toujours préférable.
//   - VALEURS EN FRANÇAIS, EXPLICATION DANS SA LANGUE : c'est la plateforme qui lit les réponses, pas
//     elle. Si tout sort dans sa langue, elle colle de l'anglais chez un client français.
func TestVoixCompose_BriefDesCaptures(t *testing.T) {
	guidance := strings.ToLower(composeVoiceGuidance())

	assert.Contains(t, guidance, "capture",
		"rien n'explique à l'agent quoi faire d'une image")
	assert.Contains(t, guidance, "invente",
		"sans interdiction explicite, l'agent remplira un champ avec du plausible et du faux")
	assert.Contains(t, guidance, "champ",
		"l'agent doit dire QUEL champ reçoit QUOI, pas répondre en vrac")
}

// Le chat PUBLIC ne doit pas hériter de ces instructions : un visiteur qui pose une question sur le CV
// n'a rien à faire avec des consignes de rédaction de messages de prospection.
func TestVoixCompose_AbsenteDuChatPublic(t *testing.T) {
	s := &ChatService{}

	// buildTools est le seul point testable sans réseau ; on vérifie au moins que la garde existe au
	// niveau des capabilities, la voix suivant le même drapeau.
	require.NotContains(t, toolNames(t, s, ChatCapabilities{}), "set_draft")
}

// --- Structure de vente ------------------------------------------------------
//
// POURQUOI ces assertions existent : avant elles, l'ouverture sur le problème du client et l'appel à
// l'action unique étaient ÉMERGENTS — observés sur les messages réels, jamais demandés. Un changement
// de modèle pouvait les faire disparaître sans qu'aucun test ne bronche, et personne ne s'en serait
// aperçu avant plusieurs candidatures sans réponse.
//
// Elles ne vérifient pas que le modèle OBÉIT (invérifiable ici) — elles vérifient que la consigne
// EXISTE. C'est ce qui empêche qu'on la supprime par inadvertance en remaniant le prompt.
func TestComposeVoiceGuidance_PorteLaStructureDeVente(t *testing.T) {
	g := composeVoiceGuidance()

	for _, cas := range []struct{ marqueur, pourquoi string }{
		{"OUVRE SUR LEUR PROBLÈME", "ouvrir sur le client et non sur soi est la pratique n°1 du domaine"},
		{"PROUVE QUE TU AS LU L'ANNONCE", "la personnalisation est le facteur qui double le taux de réponse"},
		{"TERMINE PAR UNE OUVERTURE, PAS PAR UNE INJONCTION", "le call fermé n'a rien converti en 2026"},
		{"pas une liste de technos", "un client achète un résultat, pas un inventaire de stack"},
	} {
		assert.Contains(t, g, cas.marqueur, "consigne de structure absente — %s", cas.pourquoi)
	}
}

// GARDE-FOU le plus important de cette section : exiger un détail précis SANS interdire d'inventer
// pousserait le modèle à en fabriquer un quand l'annonce est vague. Ce serait pire que l'absence —
// un détail faux se voit immédiatement, et l'opératrice ne lit pas le français pour l'attraper.
func TestComposeVoiceGuidance_PreuveDeLectureInterditLInvention(t *testing.T) {
	g := composeVoiceGuidance()
	assert.Contains(t, g, "RÉELLEMENT dans l'annonce")
	assert.Contains(t, g, "n'en invente pas",
		"la preuve de lecture doit être bornée par une interdiction d'inventer")
}

// Le comptage est explicitement DÉCLASSÉ face à la preuve de réalité.
//
// POURQUOI cette précision compte : les seuls chiffres dont l'agent dispose sont des métriques
// d'EFFORT (tests, modules, étapes) — elles mesurent mon activité, pas un résultat pour le client.
// Une consigne du genre « donne un chiffre » aurait donc pousse vers la vanité, faute de mieux.
// Mesuré sur les 79 fiches : un seul mini dit « en production » alors que 39 projets le sont, d'où
// la marque [production] ajoutée à l'inventaire — c'est elle que le message doit reprendre.
func TestComposeVoiceGuidance_DeclasseLeComptage(t *testing.T) {
	g := composeVoiceGuidance()
	assert.Contains(t, g, "prouve l'effort, pas le résultat")
	assert.Contains(t, g, "[production]",
		"l'agent doit savoir que la marque de production est la preuve à reprendre")
}

// L'agent doit LIRE la fiche du projet avant d'écrire, pas se contenter du résumé d'inventaire.
//
// CONSTAT qui motive cette consigne (2026-08-02) : sur une génération réelle, l'agent n'a appelé que
// set_draft et set_rate. Il a rédigé tout le message depuis le mini seul — 100 caractères — alors que
// description.portfolio en contient ~2 800 : le problème réellement traité, les vrais utilisateurs,
// ce que ça a changé pour eux. Exactement la matière qui manquait au message.
//
// EFFET DE BORD dont il faut se souvenir : l'inventaire des projets a rendu l'agent MOINS curieux.
// Avant lui, l'agent ne connaissait aucun projet et devait chercher ; depuis, il en sait assez pour
// écrire quelque chose de plausible sans jamais ouvrir une fiche. Ajouter du contexte peut réduire
// l'investigation — il faut alors la redemander explicitement.
func TestComposeVoiceGuidance_ExigeDeLireLaFiche(t *testing.T) {
	g := composeVoiceGuidance()
	assert.Contains(t, g, "get_project",
		"rien ne dit à l'agent d'aller chercher la fiche : il écrira depuis le résumé")
	assert.Contains(t, g, "AVANT D'ÉCRIRE, LIS LA FICHE")
	assert.Contains(t, g, "sert à CHOISIR, pas à décrire",
		"le rôle de l'inventaire doit être explicite, sinon il sert de source de rédaction")
}

// Régressions du constat Malt 2026 (14 envois, 1 échange, décroché sur « pas de lien public ») :
// le prompt exige la page du projet cité quand elle existe, interdit d'inventer une adresse, et
// nomme le moule d'ouverture qui se répétait d'un message à l'autre.
func TestComposeVoice_LienProfondEtFinDuMoule(t *testing.T) {
	g := composeVoiceGuidance()
	assert.Contains(t, g, "https://maicivy.etheryale.com/fr/portfolio/<identifiant>")
	assert.Contains(t, g, "[page]")
	assert.Contains(t, g, "N'INVENTE JAMAIS une adresse")
	assert.Contains(t, g, "casse (ou coince) rarement")
	assert.Contains(t, g, "J'ai construit <projet>, en production")
}

// Retour d'Alexi du 2026-10-16 : « on envoie 3 lignes et demi dans un style que j'aime pas. Je
// propose une strat moins impérative dans ce contexte et plus de lien pour référence. »
// Trois consignes, trois verrous : densité (2-3 projets, 250-350 mots), ouverture au lieu du call
// fermé, 3 à 5 liens de référence. Et le générateur /messages/generate GARDE son format court : la
// longueur a quitté le bloc de voix partagé pour cette raison.
func TestComposeVoice_DenseOuvertEtReference(t *testing.T) {
	g := composeVoiceGuidance()
	assert.Contains(t, g, "2 OU 3 PROJETS", "densité : plusieurs preuves")
	assert.Contains(t, g, "250 à 350 mots")
	assert.NotContains(t, g, "120 à 180 mots", "l'ancien plafond court contredirait la densité")
	assert.Contains(t, g, "3 À 5 LIENS DE RÉFÉRENCE")
	assert.Contains(t, g, "https://maicivy.etheryale.com/fr/gitstats")
	assert.NotContains(t, g, "Un ou deux liens au plus")
	assert.NotContains(t, g, "laquelle on répond oui")
	// Le call fermé n'apparaît plus que comme exemple À NE PAS faire.
	assert.Equal(t, 1, strings.Count(g, "Disponible pour un call cette semaine"))
	assert.Contains(t, g, "Pas de « Disponible pour un call cette semaine ? »")

	// Le plafond mécanique doit laisser passer un message conforme au prompt.
	assert.GreaterOrEqual(t, defaultMaxDraftRunes, 2600)

	// Le générateur plateforme reste court.
	pb := &PromptBuilder{userProfile: models.UserProfile{Name: "Alexis Trouve"}}
	court := pb.BuildPlatformMessagePrompt(models.PlatformMessageRequest{
		Mission: "Recherche freelance IA.", Platform: "malt", Lang: "fr",
	})
	assert.Contains(t, court, "120 à 180 mots")
	assert.NotContains(t, court, "250 à 350 mots")
}
