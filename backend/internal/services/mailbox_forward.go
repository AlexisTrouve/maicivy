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
	// remplir). Tag = libellé de tri injecté dans le sujet (« 87 ✓ 优先处理 », « 62 ? 待评估 », « 20 ✕ 建议拒绝 »), Note =
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

	// AckURL : lien signé « ✓ 标记为已处理 » (cf. MailboxAckURL). Un clic marque l'offre traitée et arrête
	// les rappels. Vide = pas de lien (non-opportunité, ou secret de signature non configuré).
	AckURL string
	// Reminder / ReminderTotal : 0 = transfert initial ; n ≥ 1 = n-ième rappel sur ReminderTotal.
	// Le rappel est le MÊME mail que le transfert, sujet et bandeau préfixés — la destinataire
	// retrouve l'offre entière sans aller chercher l'original dans sa boîte.
	Reminder      int
	ReminderTotal int
}

// RenderMailboxForward compose le message RFC822 de transfert d'un mail capté vers l'adresse fixe
// MAILBOX_FORWARD_TO. Sujet préfixé par la plateforme (tri visuel en boîte de réception), Reply-To
// pointé sur l'expéditeur ORIGINAL (répondre au transfert répond directement à Malt, pas à la boîte
// de capture) — texte brut, pas de HTML : c'est un relai interne, pas une newsletter à soigner.
func RenderMailboxForward(fromEmail, fromName, to string, in MailboxForwardInput) []byte {
	// Sujet : [plateforme][rappel n/N][score glyphe priorité] sujet original. Chaque segment n'apparaît
	// que s'il a un contenu — un mail non-opportunité reste « [malt] sujet ».
	tag := fmt.Sprintf("[%s]", in.Platform)
	if in.Reminder > 0 {
		tag += fmt.Sprintf("[提醒 %d/%d]", in.Reminder, in.ReminderTotal)
	}
	if in.MaltActionTag != "" {
		tag += fmt.Sprintf("[%s]", in.MaltActionTag)
	}
	subject := fmt.Sprintf("%s %s", tag, in.Subject)

	// Bandeau de tête : rappel (s'il y a lieu), consigne d'action, lien « traité ». Repères en
	// chinois, consigne en anglais (cf. maltAction pour le pourquoi de l'anglais).
	var warningBlock string
	if in.Reminder > 0 {
		warningBlock += fmt.Sprintf(
			"⏰ 提醒 %d/%d — Reminder: this offer is still open. Apply or refuse it on Malt, then click the ✓ link below.\r\n\r\n",
			in.Reminder, in.ReminderTotal)
	}
	if in.MaltActionNote != "" {
		warningBlock += fmt.Sprintf("⚠️  %s\r\n\r\n", in.MaltActionNote)
	}
	if in.AckURL != "" {
		warningBlock += fmt.Sprintf("✓ 标记为已处理 / Done (applied OR refused) — click to stop the reminders:\r\n%s\r\n\r\n", in.AckURL)
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
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n")
	// Encodage du corps — voir QuotedPrintable (email_compose.go) pour le pourquoi détaillé. Résumé :
	// sans lui, les notifications d'offre Malt (un paragraphe entier sur une seule ligne, URL de
	// suivi comprise) dépassaient les 998 octets autorisés et étaient rejetées par le serveur
	// destinataire — aucune opportunité n'arrivait, alors que les newsletters, plus courtes, passaient.
	fmt.Fprintf(&b, "Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	b.WriteString(QuotedPrintable(text))
	return []byte(b.String())
}
