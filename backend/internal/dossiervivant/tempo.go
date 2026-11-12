package dossiervivant

import (
	"fmt"
	"time"
)

// QUOI : le TEMPO d'un run — un rappel SOUPLE, jamais un arrêt. Quand le temps écoulé approche la
// cible (tempoApproach) ou la dépasse (tempoTarget), chaque appel de modèle suivant le sait dans son
// prompt et resserre son travail : réflexion brève, réponse au plus court, conclure. Le code, lui,
// ouvre moins de fiches (tempo.cap).
//
// POURQUOI pas un délai dur : décision d'Alexis — « la soluce n'est PAS un stop mais un rappel
// soft ». L'ancien context.WithTimeout de 100 s coupait un run en cours (`run.failed` raison
// `timeout`) : le visiteur perdait tout le travail déjà fait. Un run lent est un run moins bon, pas un
// run perdu. La cible (20 à 30 s) vise la patience d'un visiteur qui regarde l'écran.
//
// COMMENT : l'horloge est injectable (Options.Now) — les tests la pilotent. Le rappel part dans le
// MESSAGE utilisateur, pas dans la consigne système : la consigne reste identique d'un run à l'autre
// et garde le cache de prompt de la passerelle.
//
// La borne qui reste : le transport. Le serveur Fiber (WriteTimeout) et le front-door (ReadTimeout du
// client de relais) bornent tous deux une réponse à 120 s. Ce n'est pas un arrêt du run voulu par le
// produit mais la limite de la connexion ; à 20–30 s de cible, elle laisse une marge de ×4.

const (
	tempoApproach = 20 * time.Second // début de la fourchette cible : on commence à resserrer
	tempoTarget   = 30 * time.Second // fin de la fourchette : on conclut
)

// pace — où en est le run par rapport à sa cible.
type pace int

const (
	onTrack pace = iota
	approaching
	late
)

type tempo struct {
	start time.Time
	now   func() time.Time
}

func (t tempo) elapsed() time.Duration { return t.now().Sub(t.start) }

func (t tempo) pace() pace {
	switch e := t.elapsed(); {
	case e >= tempoTarget:
		return late
	case e >= tempoApproach:
		return approaching
	default:
		return onTrack
	}
}

// Les rappels, en anglais comme toutes les consignes (prompts.go). %d = secondes écoulées.
const (
	approachReminder = "\n<pace>This run has been going for %d s; the team aims to hand over the dossier within %d s. Keep this answer tight: think briefly and write only what the shape asks for, at the short end of every length limit.</pace>\n"
	lateReminder     = "\n<pace>This run is past its %d s target (%d s elapsed). Conclude now: think as little as possible and give the shortest answer that fits the shape.</pace>\n"
)

// reminderReserve — la place que le plus long rappel peut prendre dans un message (callsize.go).
var reminderReserve = estimateTokens(fmt.Sprintf(approachReminder, 99999, 99), fmt.Sprintf(lateReminder, 99, 99999))

// reminder — le rappel à joindre au message d'un appel qui part MAINTENANT ; vide tant que le run
// est dans les temps.
func (t tempo) reminder() string {
	secs := int(t.elapsed().Seconds())
	target := int(tempoTarget.Seconds())
	switch t.pace() {
	case approaching:
		return fmt.Sprintf(approachReminder, secs, target)
	case late:
		return fmt.Sprintf(lateReminder, target, secs)
	}
	return ""
}

// cap — combien d'éléments ouvrir : `full` dans les temps, `tight` dès qu'on approche de la cible.
// C'est le « moins de lectures » du rappel, tenu par le code (un modèle ne décide pas combien de
// fiches le code ouvre).
func (t tempo) cap(full, tight int) int {
	if t.pace() == onTrack {
		return full
	}
	return tight
}
