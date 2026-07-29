package services_test

import (
	"strings"
	"testing"
	"time"

	"maicivy/internal/services"
)

func TestRenderMailboxForward(t *testing.T) {
	in := services.MailboxForwardInput{
		FromAddress: "notifications@malt.fr",
		Subject:     "Nouvelle mission disponible",
		BodyText:    "Un client recherche un développeur Go.",
		Platform:    "malt",
		ReceivedAt:  time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))

	must := []string{
		"From: Mailbox Etheryale <mailbox@etheryale.com>",
		"To: dest@example.com",
		"Reply-To: notifications@malt.fr",
		"Subject: [malt] Nouvelle mission disponible",
		"Content-Type: text/plain",
		"Transféré automatiquement depuis malt",
		"Expéditeur original : notifications@malt.fr",
		"Un client recherche un développeur Go.",
	}
	for _, s := range must {
		if !strings.Contains(msg, s) {
			t.Errorf("email de transfert ne contient pas: %q\n---\n%s", s, msg)
		}
	}
}

// MaltActionTag/Note non-vides → tag injecté dans le sujet EN PLUS de [malt] + bandeau en tête de corps,
// mais ne bloque jamais rien (c'est une consigne, cf. décision produit dans mailbox_service.go).
func TestRenderMailboxForward_WithMaltAction(t *testing.T) {
	in := services.MailboxForwardInput{
		FromAddress:    "notifications@malt.fr",
		Subject:        "Mission hors profil",
		BodyText:       "Corps du mail original.",
		Platform:       "malt",
		ReceivedAt:     time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		MaltActionTag:  "TO REFUSE",
		MaltActionNote: "REFUSE this offer on Malt. Low relevance (20/100). Reason: out of profile",
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))

	if !strings.Contains(msg, "REFUSE this offer on Malt") {
		t.Errorf("le corps du transfert doit contenir la consigne, got:\n%s", msg)
	}
	if !strings.Contains(msg, "20/100") {
		t.Errorf("la consigne doit contenir le score, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Corps du mail original.") {
		t.Errorf("le corps original doit rester présent malgré la consigne, got:\n%s", msg)
	}
	// Le sujet (avant encodage RFC2047) doit porter le tag [TO REFUSE] EN PLUS de [malt].
	if !strings.Contains(msg, "[malt][TO REFUSE] Mission hors profil") {
		t.Errorf("le sujet doit être préfixé par [TO REFUSE], got:\n%s", msg)
	}
}

// Sujet/nom accentués → RFC2047 (mimeWord), sinon le sujet cassera dans certains clients mail.
func TestRenderMailboxForward_EncodesNonASCII(t *testing.T) {
	in := services.MailboxForwardInput{
		FromAddress: "notifications@malt.fr",
		Subject:     "Mission à Paris — développeur écosystème",
		Platform:    "malt",
		ReceivedAt:  time.Now(),
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))
	if !strings.Contains(msg, "Subject: =?UTF-8?") {
		t.Errorf("sujet non-ASCII doit être RFC2047-encodé, got:\n%s", msg)
	}
}
