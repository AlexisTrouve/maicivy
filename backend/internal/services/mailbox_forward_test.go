package services_test

import (
	"io"
	"mime/quotedprintable"
	"strings"
	"testing"
	"time"

	"maicivy/internal/services"
)

// corpsDecode sépare les en-têtes du corps et décode le corps en quoted-printable.
//
// POURQUOI les tests passent par ici plutôt que de chercher une sous-chaîne dans le message brut :
// depuis l'encodage QP, « Transféré » s'écrit « Transf=C3=A9r=C3=A9 » sur le fil, et une longue
// ligne est coupée par des retours souples. Assérer sur les octets transmis reviendrait à tester
// l'encodage ; ce qu'on veut vérifier, c'est ce que la destinataire LIT.
func corpsDecode(t *testing.T, msg string) string {
	t.Helper()
	_, corps, ok := strings.Cut(msg, "\r\n\r\n")
	if !ok {
		t.Fatalf("message sans séparation en-têtes/corps:\n%s", msg)
	}
	dec, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(corps)))
	if err != nil {
		t.Fatalf("corps non décodable en quoted-printable: %v", err)
	}
	return string(dec)
}

func TestRenderMailboxForward(t *testing.T) {
	in := services.MailboxForwardInput{
		FromAddress: "notifications@malt.fr",
		Subject:     "Nouvelle mission disponible",
		BodyText:    "Un client recherche un développeur Go.",
		Platform:    "malt",
		ReceivedAt:  time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))

	// En-têtes : lus tels quels sur le fil.
	for _, s := range []string{
		"From: Mailbox Etheryale <mailbox@etheryale.com>",
		"To: dest@example.com",
		"Reply-To: notifications@malt.fr",
		"Subject: [malt] Nouvelle mission disponible",
		"Content-Type: text/plain",
	} {
		if !strings.Contains(msg, s) {
			t.Errorf("en-tête manquant: %q\n---\n%s", s, msg)
		}
	}

	// Corps : vérifié APRÈS décodage — sur le fil, les accents sont échappés par le quoted-printable.
	corps := corpsDecode(t, msg)
	for _, s := range []string{
		"Transféré automatiquement depuis malt",
		"Expéditeur original : notifications@malt.fr",
		"Un client recherche un développeur Go.",
	} {
		if !strings.Contains(corps, s) {
			t.Errorf("corps décodé ne contient pas: %q\n---\n%s", s, corps)
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

	corps := corpsDecode(t, msg)
	if !strings.Contains(corps, "REFUSE this offer on Malt") {
		t.Errorf("le corps du transfert doit contenir la consigne, got:\n%s", corps)
	}
	if !strings.Contains(corps, "20/100") {
		t.Errorf("la consigne doit contenir le score, got:\n%s", corps)
	}
	if !strings.Contains(corps, "Corps du mail original.") {
		t.Errorf("le corps original doit rester présent malgré la consigne, got:\n%s", corps)
	}
	// Le sujet (avant encodage RFC2047) doit porter le tag [TO REFUSE] EN PLUS de [malt].
	if !strings.Contains(msg, "[malt][TO REFUSE] Mission hors profil") {
		t.Errorf("le sujet doit être préfixé par [TO REFUSE], got:\n%s", msg)
	}
}

// RÉGRESSION — panne de production du 2026-08-01 : AUCUNE opportunité Malt n'arrivait à la
// destinataire. Le serveur SMTP rejetait chaque transfert avec
//
//	550 maximum allowed line length is 998 octets, got 3019
//
// Le corps était écrit tel quel sous un Content-Type sans Content-Transfer-Encoding : l'encodage par
// défaut est alors `7bit`, qui plafonne une ligne à 998 octets (RFC 5321 §4.5.3.1.6). Or les
// notifications d'offre Malt tiennent un paragraphe entier — URL de suivi comprise — sur une seule
// ligne. Symptôme cruel : les mails COURTS (newsletters) passaient, les vraies offres non.
//
// Ce test tient sur l'invariant du protocole, pas sur un encodage particulier : quelle que soit la
// façon dont on s'y prend, aucune ligne transmise ne doit dépasser 998 octets.
func TestRenderMailboxForward_AucuneLigneNeDepasse998(t *testing.T) {
	// Reproduit la forme réelle : un paragraphe long d'un seul tenant, sans espace de repli possible
	// à la fin (une URL) — le cas où un simple retour à la ligne sur les espaces ne suffirait pas.
	longue := strings.Repeat("Un client recherche un développeur Go pour une mission longue. ", 60) +
		"https://www.malt.fr/projects/" + strings.Repeat("a", 900)

	in := services.MailboxForwardInput{
		FromAddress: "notifications@malt.fr",
		Subject:     "Nouvelle opportunité avec Galadrim",
		BodyText:    longue,
		Platform:    "malt",
		ReceivedAt:  time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC),
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))

	for i, ligne := range strings.Split(msg, "\r\n") {
		if len(ligne) > 998 {
			t.Fatalf("ligne %d longue de %d octets — le serveur SMTP rejettera le message (max 998)", i+1, len(ligne))
		}
	}

	// Le contenu doit survivre à l'encodage : un transfert illisible ne vaut pas mieux qu'un transfert
	// refusé. On vérifie sur le texte DÉCODÉ, tel que la destinataire le verra.
	if got := corpsDecode(t, msg); !strings.Contains(got, longue) {
		t.Errorf("le corps original doit être restitué intact après décodage")
	}
}

// L'encodage doit être ANNONCÉ, sinon le client mail affiche le quoted-printable brut
// (« Transf=C3=A9r=C3=A9 ») au lieu du texte.
func TestRenderMailboxForward_DeclareLEncodage(t *testing.T) {
	in := services.MailboxForwardInput{
		FromAddress: "notifications@malt.fr",
		Subject:     "Mission",
		BodyText:    "Corps accentué : déjà vu.",
		Platform:    "malt",
		ReceivedAt:  time.Now(),
	}
	msg := string(services.RenderMailboxForward("mailbox@etheryale.com", "Mailbox Etheryale", "dest@example.com", in))
	if !strings.Contains(msg, "Content-Transfer-Encoding: quoted-printable\r\n") {
		t.Errorf("en-tête Content-Transfer-Encoding manquant, got:\n%s", msg)
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
