package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// AdminChatHandler — persistance des conversations de l'agent admin (mémoire durable owner-only).
// POURQUOI : le streaming de l'agent réutilise /chat (owner cookie → Opus + tools maiProFiles) ; ce
// handler ne fait QUE le CRUD des conversations en base. Le front charge une conversation, streame les
// nouveaux tours via /chat, puis PUT la conversation mise à jour ici.
// Accès : owner OU prospecteur actif (onglet Agent), chacun cloisonné sur SES conversations via
// draftScope — même périmètre que les ateliers Composer (cf. compose_drafts.go).
type AdminChatHandler struct {
	db            *gorm.DB
	sessionSecret string
}

func NewAdminChatHandler(db *gorm.DB, sessionSecret string) *AdminChatHandler {
	return &AdminChatHandler{db: db, sessionSecret: sessionSecret}
}

func (h *AdminChatHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/admin/chat")
	g.Get("/conversations", h.list)
	g.Post("/conversations", h.create)
	g.Get("/conversations/:id", h.get)
	g.Put("/conversations/:id", h.save)
	g.Delete("/conversations/:id", h.del)
}

// scope authentifie (owner ou prospecteur actif, posé par AdminRoleGuard) et renvoie le périmètre.
// POURQUOI réutiliser draftScope : une seule règle de propriété pour tout ce qu'un prospecteur crée ;
// une conversation d'un autre compte est INTROUVABLE (404), comme un atelier. 401 sinon.
func (h *AdminChatHandler) scope(c *fiber.Ctx) (draftScope, bool) {
	if GetStatsAuth(c, h.sessionSecret) { // réutilise VerifyAdminCookie (cf. admin_stats.go)
		return draftScope{}, true
	}
	if p, ok := middleware.ProspectFromCtx(c); ok {
		id := p.ID
		return draftScope{prospectID: &id}, true
	}
	_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	return draftScope{}, false
}

func (h *AdminChatHandler) notFound(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
}

// list — résumés des conversations (sans messages), plus récentes d'abord.
func (h *AdminChatHandler) list(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}
	type row struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	rows := []row{}
	if h.db != nil {
		sc.apply(h.db.Model(&models.ChatConversation{})).
			Select("id, title, updated_at").Order("updated_at desc").Limit(100).Scan(&rows)
	}
	return c.JSON(fiber.Map{"conversations": rows})
}

// create — nouvelle conversation vide.
func (h *AdminChatHandler) create(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}
	conv := models.ChatConversation{ID: uuid.NewString(), Title: "Nouvelle conversation", Messages: "[]", ProspectID: sc.prospectID}
	if err := h.db.Create(&conv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"id": conv.ID, "title": conv.Title})
}

// get — conversation complète (messages parsés en JSON, pas en string).
func (h *AdminChatHandler) get(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}
	var conv models.ChatConversation
	if err := sc.apply(h.db).First(&conv, "id = ?", c.Params("id")).Error; err != nil {
		return h.notFound(c)
	}
	msgs := conv.Messages
	if msgs == "" {
		msgs = "[]"
	}
	return c.JSON(fiber.Map{"id": conv.ID, "title": conv.Title, "messages": json.RawMessage(msgs)})
}

// save — met à jour titre et/ou messages (le front persiste après chaque tour).
func (h *AdminChatHandler) save(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}
	var body struct {
		Title    string          `json:"title"`
		Messages json.RawMessage `json:"messages"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if body.Title != "" {
		if len(body.Title) > 200 {
			body.Title = body.Title[:200]
		}
		updates["title"] = body.Title
	}
	if len(body.Messages) > 0 {
		updates["messages"] = string(body.Messages)
	}
	res := sc.apply(h.db.Model(&models.ChatConversation{})).Where("id = ?", c.Params("id")).Updates(updates)
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 { // hors périmètre ou inexistante : indiscernables, à dessein
		return h.notFound(c)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// del — supprime une conversation.
func (h *AdminChatHandler) del(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}
	if sc.apply(h.db).Delete(&models.ChatConversation{}, "id = ?", c.Params("id")).RowsAffected == 0 {
		return h.notFound(c)
	}
	return c.JSON(fiber.Map{"ok": true})
}
