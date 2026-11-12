package api

import (
	"errors"
	"html"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"maicivy/internal/services"
)

// Lien « ✓ 标记为已处理 » des mails transférés (cf. services/mailbox_reminder.go).
//
// PUBLIC, sans cookie : la destinataire clique depuis sa messagerie (QQ), sans session ni compte. La
// seule autorisation est la signature HMAC de l'id dans l'URL — elle ne permet qu'UN geste : marquer
// CE mail traité. Rien n'est lu ni renvoyé du mail lui-même.
//
// POURQUOI deux temps (GET = page de confirmation, POST = marquage) : les messageries et leurs
// antivirus OUVRENT les liens d'un mail pour les analyser. Un GET qui marquerait directement ferait
// passer chaque offre pour traitée dès sa réception — et couperait les rappels en silence, soit
// exactement la panne qu'ils corrigent. Un scanner suit un lien, il ne soumet pas un formulaire.
//
// Pages en chinois + anglais, en dur ici : ce sont des pages serveur hors du front next-intl, qui
// n'ont qu'une lectrice, et dont le texte tient en deux phrases.

// RegisterAckRoutes enregistre les deux routes publiques du lien « traité ».
func (h *MailboxHandler) RegisterAckRoutes(api fiber.Router) {
	api.Get("/mailbox/ack/:id", h.ackPage)
	api.Post("/mailbox/ack/:id", h.ackConfirm)
}

// ackVerifie extrait l'id et vérifie la signature. Écrit la réponse d'erreur et renvoie ok=false sinon.
func (h *MailboxHandler) ackVerifie(c *fiber.Ctx) (uuid.UUID, string, bool) {
	id, err := uuid.Parse(c.Params("id"))
	sig := c.Query("sig")
	if err != nil || !services.VerifyMailboxAck(id.String(), sig, h.sessionSecret) {
		_ = ackHTML(c, fiber.StatusForbidden, "✕ 链接无效 / Invalid link", "")
		return uuid.Nil, "", false
	}
	return id, sig, true
}

// ackPage — GET : page de confirmation, un seul bouton qui POSTe vers la même URL. Ne modifie rien.
func (h *MailboxHandler) ackPage(c *fiber.Ctx) error {
	id, sig, ok := h.ackVerifie(c)
	if !ok {
		return nil
	}
	action := "/api/v1/mailbox/ack/" + id.String() + "?sig=" + sig
	form := `<form method="post" action="` + html.EscapeString(action) + `">` +
		`<button type="submit" data-testid="ack-confirm">✓ 确认已处理 / Confirm done</button></form>`
	return ackHTML(c, fiber.StatusOK, "已在 Malt 上投递或拒绝了吗？ / Applied or refused on Malt?", form)
}

// ackConfirm — POST : marque le mail traité (idempotent) et arrête ses rappels.
func (h *MailboxHandler) ackConfirm(c *fiber.Ctx) error {
	id, _, ok := h.ackVerifie(c)
	if !ok {
		return nil
	}
	deja, err := services.MarkHandled(h.db, id, time.Now())
	if errors.Is(err, services.ErrMailboxNotFound) {
		return ackHTML(c, fiber.StatusNotFound, "✕ 未找到该邮件 / Not found", "")
	}
	if err != nil {
		return ackHTML(c, fiber.StatusInternalServerError, "✕ 出错了，请再试一次 / Error, please retry", "")
	}
	if deja {
		return ackHTML(c, fiber.StatusOK, "✓ 此前已标记为已处理 / Already marked done", "")
	}
	return ackHTML(c, fiber.StatusOK, "✓ 已标记为已处理，不再提醒 / Done — no more reminders", "")
}

// ackHTML rend une page minimale autonome (aucune ressource externe). titre est échappé ; corps est
// du HTML construit par l'appelant à partir de valeurs déjà échappées.
func ackHTML(c *fiber.Ctx, status int, titre, corps string) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	c.Set("X-Robots-Tag", "noindex")
	return c.Status(status).SendString(`<!doctype html><html><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex">` +
		`<title>maicivy</title><style>body{font-family:system-ui,sans-serif;background:#0f172a;color:#e2e8f0;` +
		`display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}` +
		`main{text-align:center;padding:2rem}h1{font-size:1.4rem}` +
		`button{font-size:1.2rem;padding:.9rem 1.6rem;border-radius:.6rem;border:0;background:#d4a843;color:#0f172a;cursor:pointer}` +
		`</style></head><body><main><h1 data-testid="ack-title">` + html.EscapeString(titre) + `</h1>` + corps +
		`</main></body></html>`)
}
