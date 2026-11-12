package dossiervivant

import (
	"errors"
	"fmt"
	"strings"
)

// QUOI : le plafond d'entrée d'UN appel de modèle, et les aides qui font TENIR chaque prompt dessous.
//
// POURQUOI par appel et pas par run : au-delà de 100 000 tokens d'entrée, claude-haiku-5-5 passe à
// un autre tarif (×5). C'est une propriété de chaque requête, pas de leur somme. L'ancien budget de
// 90 000 tokens PAR RUN confondait les deux : il arrêtait des runs entiers (`run.failed` raison
// `budget`) alors qu'aucun de leurs appels n'approchait la borne. Décision d'Alexis : plus aucun arrêt
// sur la consommation d'un run.
//
// COMMENT la propriété est garantie, sans arrêter le run :
//  1. complete (mind.go), seul chemin vers la passerelle, refuse d'émettre un appel dont l'entrée
//     estimée dépasse maxCallInput. C'est la garantie : aucun appel trop gros ne PART ;
//  2. les seuls prompts de taille non bornée sont construits pour tenir AVANT d'arriver là — ce
//     sont ceux qui portent des données externes (maiProFiles) :
//     - l'index (filtre, références) : lignes ajoutées tant qu'elles tiennent (fitLines) ;
//     - le brief d'un projet (verdict, leçon) : un brief qui ne tient pas n'est PAS jugé — note
//       visible dans la colonne de l'agent — et jamais rogné (règle des briefs, brief.go) ;
//     - les preuves relues par le Rédacteur (apport) : les preuves dont le brief ne tient plus
//       passent dans « aussi retenus » (writer.go › closingFor).
//     Tous les autres prompts sont bornés par construction (besoin ≤ 2 000 caractères, sorties de
//     modèle ramenées sous leur borne par fit.go).
//
// L'estimation : octets / 3. Un caractère chinois pèse 3 octets pour environ un token, le français
// tourne autour de 4 caractères par token : octets/3 majore les deux. Pour un texte très chargé en
// symboles (code, JSON) ce n'est plus un majorant strict, d'où la marge de 20 % sous la borne.

const maxCallInput = 80_000

// ErrCallTooLarge — un appel aurait dépassé maxCallInput. Ne doit jamais se produire : les prompts
// sont ajustés en amont (cf. en-tête). S'il se produit, c'est un bug — l'appel ne part pas.
var ErrCallTooLarge = errors.New("dossier vivant: appel au-delà du plafond d'entrée")

// estimateTokens — majorant pratique du nombre de tokens d'un texte (cf. en-tête).
func estimateTokens(texts ...string) int {
	n := 0
	for _, t := range texts {
		n += len(t)
	}
	return n/3 + 1
}

// room — tokens estimés laissés au message d'une tâche `t` : le plafond, moins la consigne système
// et le plus long rappel de tempo qui pourra s'y ajouter (tempo.go).
func (r *run) room(t task) int {
	return maxCallInput - estimateTokens(systemPrompt(t, r.lang)) - reminderReserve
}

// fits — le message `prompt` de la tâche `t` tient-il sous le plafond ?
func (r *run) fits(t task, prompt string) bool { return estimateTokens(prompt) <= r.room(t) }

// fitLines — les lignes, dans l'ordre, tant qu'elles tiennent dans `room` tokens estimés. Rend le
// texte et le nombre de lignes laissées de côté (journalisé par l'appelant).
func fitLines(lines []string, room int) (string, int) {
	var b strings.Builder
	for i, l := range lines {
		if estimateTokens(b.String(), l) > room {
			return b.String(), len(lines) - i
		}
		b.WriteString(l)
	}
	return b.String(), 0
}

// checkCallSize — la garantie finale, posée par complete juste avant l'émission.
func checkCallSize(agent AgentID, k call) error {
	if in := estimateTokens(k.system, k.prompt); in > maxCallInput {
		return fmt.Errorf("%w: %s/%s, %d tokens estimés (plafond %d)", ErrCallTooLarge, agent, k.task, in, maxCallInput)
	}
	return nil
}
