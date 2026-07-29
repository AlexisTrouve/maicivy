package services

import (
	"fmt"
	"strings"
	"time"
)

// MailboxForwardInput — données minimales d'un mail capté nécessaires pour composer son transfert.
type MailboxForwardInput struct {
	FromAddress string // expéditeur original (ex: notifications@malt.fr)
	Subject     string
	BodyText    string
	Platform    string // label allowlist (ex: "malt")
	ReceivedAt  time.Time
	// MaltActionTag / MaltActionNote : consigne d'action pour la destinataire, sur TOUTE vraie
	// opportunité (cf. maltAction() dans mailbox_service.go, seul appelant qui décide quand les
	// remplir). Tag = libellé de tri injecté dans le sujet ("TO REFUSE" / "TO REVIEW"), Note =
	// bandeau en tête de corps. Les deux vides = rien n'est ajouté (non-opportunité ou filtre non
	// configuré).
	//
	// POURQUOI une consigne et pas un simple avertissement : sur Malt, une offre laissée EN ATTENTE
	// bloque les offres suivantes, et la plateforme n'a aucune API (cf. MaltReminderService) — seule
	// une action manuelle de la destinataire la débloque. Un texte qui dit « pas la peine de
	// postuler » invite à l'inaction et sature donc la file. On demande une action explicite.
	//
	// On ne bloque toujours JAMAIS le transfert : la consigne oriente, elle ne décide pas.
	MaltActionTag  string
	MaltActionNote string
}

// RenderMailboxForward compose le message RFC822 de transfert d'un mail capté vers l'adresse fixe
// MAILBOX_FORWARD_TO. Sujet préfixé par la plateforme (tri visuel en boîte de réception), Reply-To
// pointé sur l'expéditeur ORIGINAL (répondre au transfert répond directement à Malt, pas à la boîte
// de capture) — texte brut, pas de HTML : c'est un relai interne, pas une newsletter à soigner.
func RenderMailboxForward(fromEmail, fromName, to string, in MailboxForwardInput) []byte {
	tag := fmt.Sprintf("[%s]", in.Platform)
	if in.MaltActionTag != "" {
		tag = fmt.Sprintf("[%s][%s]", in.Platform, in.MaltActionTag)
	}
	subject := fmt.Sprintf("%s %s", tag, in.Subject)

	var warningBlock string
	if in.MaltActionNote != "" {
		warningBlock = fmt.Sprintf("⚠️  %s\r\n\r\n", in.MaltActionNote)
	}

	text := fmt.Sprintf(
		"Transféré automatiquement depuis %s\r\nExpéditeur original : %s\r\nReçu le : %s\r\n\r\n%s---\r\n\r\n%s\r\n",
		in.Platform, in.FromAddress, in.ReceivedAt.Format(time.RFC1123Z), warningBlock, in.BodyText,
	)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", MimeWord(fromName), fromEmail)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Reply-To: %s\r\n", in.FromAddress)
	fmt.Fprintf(&b, "Subject: %s\r\n", MimeWord(subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@etheryale.com>\r\n", RandHex16())
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(text)
	return []byte(b.String())
}
