package api

import (
	"strings"
	"testing"
)

// L'email newsletter doit contenir les bons en-têtes (dont List-Unsubscribe one-click), le bon lien
// d'article, le lien de désinscription, et les deux parties (texte + HTML).
func TestRenderNewsletter(t *testing.T) {
	p := newsletterPost{Title: "Mon Article", Summary: "Resume court", Slug: "mon-article", ProjectName: "Drifterra", Locale: "fr"}
	msg := string(renderNewsletter("blog-emiter@etheryale.com", "Blog Etheryale", "lecteur@example.com", "tok123", p, "https://maicivy.etheryale.com/"))

	must := []string{
		"From: Blog Etheryale <blog-emiter@etheryale.com>",
		"To: lecteur@example.com",
		"Subject: Mon Article",
		"List-Unsubscribe: <https://maicivy.etheryale.com/api/v1/blog/unsubscribe?token=tok123>",
		"List-Unsubscribe-Post: List-Unsubscribe=One-Click",
		"multipart/alternative",
		"Content-Type: text/plain",
		"Content-Type: text/html",
		"https://maicivy.etheryale.com/fr/blog/mon-article", // lien article (locale fr)
	}
	for _, s := range must {
		if !strings.Contains(msg, s) {
			t.Errorf("email ne contient pas: %q", s)
		}
	}
}

// MÊME DÉFAUT que la panne mailbox du 2026-08-01 (cf. QuotedPrintable dans services/email_compose.go) :
// sans Content-Transfer-Encoding, l'encodage par défaut est `7bit` et toute ligne de plus de 998
// octets fait rejeter le message par le serveur destinataire (550).
//
// La newsletter y est PLUS exposée que le transfert mailbox : sa partie HTML met chaque paragraphe
// sur une seule ligne, résumé de l'article compris. Un résumé un peu long suffit à dépasser le seuil,
// et l'envoi échouerait alors pour TOUS les abonnés à la fois.
func TestRenderNewsletter_AucuneLigneNeDepasse998(t *testing.T) {
	// Résumé long d'un seul tenant : la forme exacte qui casse, et qui n'a rien d'improbable pour
	// une accroche d'article.
	longSummary := strings.Repeat("Un résumé d'article suffisamment long pour dépasser la limite. ", 40)
	p := newsletterPost{Title: "Mon Article", Summary: longSummary, Slug: "mon-article", ProjectName: "Drifterra", Locale: "fr"}
	msg := string(renderNewsletter("blog-emiter@etheryale.com", "Blog Etheryale", "lecteur@example.com", "tok123", p, "https://maicivy.etheryale.com/"))

	for i, ligne := range strings.Split(msg, "\r\n") {
		if len(ligne) > 998 {
			t.Fatalf("ligne %d longue de %d octets — le serveur SMTP rejettera le message (max 998)", i+1, len(ligne))
		}
	}

	// Les DEUX parties doivent déclarer leur encodage : une partie non déclarée s'afficherait en
	// quoted-printable brut chez le lecteur.
	if n := strings.Count(msg, "Content-Transfer-Encoding: quoted-printable"); n != 2 {
		t.Errorf("les 2 parties (texte + HTML) doivent déclarer leur encodage, trouvé %d fois:\n%s", n, msg)
	}
}

// mimeWord : ASCII inchangé, non-ASCII encodé en RFC2047 (sinon les accents cassent l'en-tête).
func TestMimeWord(t *testing.T) {
	if got := mimeWord("Blog Etheryale"); got != "Blog Etheryale" {
		t.Errorf("ASCII ne doit pas être encodé, got %q", got)
	}
	if got := mimeWord("Resume accentue éàç"); !strings.HasPrefix(got, "=?UTF-8?") {
		t.Errorf("non-ASCII doit être RFC2047-encodé, got %q", got)
	}
}
