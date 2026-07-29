package api

// Envoi de la newsletter à la publication d'un article (phase 2).
//
// FLUX : un article publié -> on cherche les abonnés dont les topics matchent son project_name
// (ou topics vide = tout) -> on rend un email multipart (texte + HTML) avec un en-tête
// List-Unsubscribe one-click (levier majeur de délivrabilité Gmail) -> on l'envoie via le relai SMTP
// (exim VPS57, qui DKIM-signe etheryale.com). Anti-doublon par slug dans Redis : un article n'envoie
// qu'UNE fois, même si CreatePost est rejoué.
//
// ENVOI : on relaie vers l'exim de VPS57 (la seule IP autorisée dans le SPF + qui a la clé DKIM).
// Config par env (défauts = prod). Pas d'auth SMTP : le relai fait confiance au subnet Tailscale.

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// newsletterPost = le minimum d'un article pour composer l'email.
type newsletterPost struct {
	Title       string
	Summary     string
	Slug        string
	ProjectName string
	Locale      string
}

// sendNewsletterAsync lance l'envoi en arrière-plan : la publication ne doit jamais bloquer/échouer
// à cause de la newsletter.
func (h *BlogHandler) sendNewsletterAsync(p newsletterPost) {
	go func() {
		if err := h.sendNewsletter(p); err != nil {
			log.Warn().Err(err).Str("slug", p.Slug).Msg("newsletter: envoi échoué")
		}
	}()
}

// sendNewsletter : dédup -> match abonnés -> rend + envoie chaque email -> marque le slug envoyé.
func (h *BlogHandler) sendNewsletter(p newsletterPost) error {
	if h.db == nil || p.Slug == "" {
		return nil
	}
	ctx := context.Background()
	const sentKey = "newsletter:sent"

	// Anti-doublon : si ce slug a déjà été envoyé, on s'arrête (idempotent sur les rejeux de publish).
	if h.redis != nil {
		if already, _ := h.redis.SIsMember(ctx, sentKey, p.Slug).Result(); already {
			log.Info().Str("slug", p.Slug).Msg("newsletter: déjà envoyée, skip")
			return nil
		}
	}

	// Match : abonné à TOUT (topics vide) OU project_name de l'article dans ses topics.
	var subs []models.BlogSubscriber
	if err := h.db.Where("cardinality(topics) = 0 OR ? = ANY(topics)", p.ProjectName).
		Find(&subs).Error; err != nil {
		return fmt.Errorf("match abonnés: %w", err)
	}

	from := getenvDefault("BLOG_FROM_EMAIL", "blog-emiter@etheryale.com")
	fromName := getenvDefault("BLOG_FROM_NAME", "Blog Etheryale")
	baseURL := strings.TrimRight(getenvDefault("BLOG_PUBLIC_URL", "https://maicivy.etheryale.com"), "/")
	addr := getenvDefault("NEWSLETTER_SMTP_HOST", "100.108.169.92") + ":" + getenvDefault("NEWSLETTER_SMTP_PORT", "25")

	sent := 0
	for _, s := range subs {
		msg := renderNewsletter(from, fromName, s.Email, s.UnsubscribeToken, p, baseURL)
		// Un échec sur un destinataire ne doit pas couper l'envoi aux autres.
		if err := services.SendViaRelay(addr, from, s.Email, msg); err != nil {
			log.Warn().Err(err).Str("to", s.Email).Msg("newsletter: échec pour un destinataire")
			continue
		}
		sent++
	}
	log.Info().Int("envoyes", sent).Int("abonnes", len(subs)).Str("slug", p.Slug).Msg("newsletter envoyée")

	// Marque le slug envoyé même si 0 abonné (évite de re-matcher à chaque rejeu).
	if h.redis != nil {
		h.redis.SAdd(ctx, sentKey, p.Slug)
	}
	return nil
}

// renderNewsletter compose le message RFC822 complet : en-têtes (dont List-Unsubscribe one-click)
// + corps multipart/alternative (texte puis HTML — l'ordre compte, le client prend le dernier qu'il sait lire).
func renderNewsletter(fromEmail, fromName, to, unsubToken string, p newsletterPost, baseURL string) []byte {
	locale := p.Locale
	if locale == "" {
		locale = "fr"
	}
	baseURL = strings.TrimRight(baseURL, "/") // évite les doubles slashs si baseURL finit par "/"
	articleURL := fmt.Sprintf("%s/%s/blog/%s", baseURL, locale, p.Slug)
	unsubURL := fmt.Sprintf("%s/api/v1/blog/unsubscribe?token=%s", baseURL, unsubToken)
	boundary := "nl_" + services.RandHex16()

	text := fmt.Sprintf("%s\r\n\r\n%s\r\n\r\nLire l'article : %s\r\n\r\n---\r\nSe désabonner : %s\r\n",
		p.Title, p.Summary, articleURL, unsubURL)

	htmlBody := fmt.Sprintf(`<!doctype html><html><body style="font-family:system-ui,sans-serif;max-width:600px;margin:0 auto;padding:24px;color:#111">
<h1 style="font-size:22px;line-height:1.3">%s</h1>
<p style="color:#444;font-size:15px;line-height:1.6">%s</p>
<p style="margin:24px 0"><a href="%s" style="display:inline-block;background:#2563eb;color:#fff;padding:11px 20px;border-radius:8px;text-decoration:none;font-weight:600">Lire l'article</a></p>
<hr style="border:none;border-top:1px solid #eee;margin:32px 0 16px">
<p style="font-size:12px;color:#999">Tu reçois ceci car tu t'es abonné au blog. <a href="%s" style="color:#999">Se désabonner</a>.</p>
</body></html>`, html.EscapeString(p.Title), html.EscapeString(p.Summary), articleURL, unsubURL)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", mimeWord(fromName), fromEmail)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mimeWord(p.Title))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@etheryale.com>\r\n", services.RandHex16())
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	// One-click unsubscribe (RFC 8058) — Gmail le réclame pour le bulk et ça aide fort l'inbox.
	fmt.Fprintf(&b, "List-Unsubscribe: <%s>\r\n", unsubURL)
	fmt.Fprintf(&b, "List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, text)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, htmlBody)
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}

// mimeWord — wrapper local pour garder le nom testé par newsletter_test.go ; logique dans services.MimeWord.
func mimeWord(s string) string {
	return services.MimeWord(s)
}

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
