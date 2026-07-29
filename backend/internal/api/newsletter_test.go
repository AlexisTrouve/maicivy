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

// mimeWord : ASCII inchangé, non-ASCII encodé en RFC2047 (sinon les accents cassent l'en-tête).
func TestMimeWord(t *testing.T) {
	if got := mimeWord("Blog Etheryale"); got != "Blog Etheryale" {
		t.Errorf("ASCII ne doit pas être encodé, got %q", got)
	}
	if got := mimeWord("Resume accentue éàç"); !strings.HasPrefix(got, "=?UTF-8?") {
		t.Errorf("non-ASCII doit être RFC2047-encodé, got %q", got)
	}
}
