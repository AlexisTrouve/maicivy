package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// MailboxHandler — panel admin de consultation des mails captés par ingestion IMAP (Malt, puis autres
// plateformes) + retry manuel de transfert. Owner-only (cookie maicivy_admin, même garde que le reste
// de /admin). service peut être nil (IMAP non configuré) : liste/détail restent utilisables (données
// déjà en DB d'un déploiement où le service tournait), seul /forward renvoie alors 503 explicite —
// c'est le handler qui reste toujours enregistré, seul le job de polling dépend de la config.
type MailboxHandler struct {
	db            *gorm.DB
	sessionSecret string
	service       *services.MailboxService
	translator    services.MailboxTranslator // nil = feature désactivée (credentials Anthropic absentes)
}

func NewMailboxHandler(db *gorm.DB, sessionSecret string, service *services.MailboxService, translator services.MailboxTranslator) *MailboxHandler {
	return &MailboxHandler{db: db, sessionSecret: sessionSecret, service: service, translator: translator}
}

func (h *MailboxHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/mailbox", h.list)
	api.Get("/admin/mailbox/:id", h.detail)
	api.Post("/admin/mailbox/:id/read", h.setRead)
	api.Post("/admin/mailbox/:id/forward", h.retryForward)
	api.Get("/admin/mailbox/:id/translation", h.getTranslation)
	api.Post("/admin/mailbox/:id/translation", h.translateNow)
}

// owner : 401 si le cookie admin n'est pas valide. Renvoie false + écrit la réponse 401.
func (h *MailboxHandler) owner(c *fiber.Ctx) bool {
	if GetStatsAuth(c, h.sessionSecret) { // réutilise VerifyAdminCookie (cf. admin_stats.go)
		return true
	}
	_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	return false
}

// mailboxEmailSummary — résumé liste, SANS body_text NI relevance_cot (potentiellement volumineux,
// réservés au détail — même logique que body_text).
type mailboxEmailSummary struct {
	ID              string     `json:"id"`
	FromAddress     string     `json:"from_address"`
	FromDomain      string     `json:"from_domain"`
	Platform        string     `json:"platform"`
	Subject         string     `json:"subject"`
	ReceivedAt      time.Time  `json:"received_at"`
	Read            bool       `json:"read"`
	ForwardedAt     *time.Time `json:"forwarded_at,omitempty"`
	ForwardError    string     `json:"forward_error,omitempty"`
	IsOpportunity   bool       `json:"is_opportunity"`
	RelevanceScore  *int       `json:"relevance_score,omitempty"`
	RelevanceReason string     `json:"relevance_reason,omitempty"`
	RelevanceLink   string     `json:"relevance_link,omitempty"`
	ForwardBlocked  bool       `json:"forward_blocked"`
}

// list — GET /admin/mailbox?page=&per_page=&platform=&unread=true
func (h *MailboxHandler) list(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&models.MailboxEmail{})
	if platform := c.Query("platform", ""); platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if c.Query("unread", "") == "true" {
		q = q.Where("read = ?", false)
	}

	var total int64
	q.Count(&total)

	var rows []models.MailboxEmail
	q.Order("received_at desc").Offset((page - 1) * perPage).Limit(perPage).Find(&rows)

	summaries := make([]mailboxEmailSummary, 0, len(rows))
	for _, e := range rows {
		summaries = append(summaries, mailboxEmailSummary{
			ID: e.ID.String(), FromAddress: e.FromAddress, FromDomain: e.FromDomain, Platform: e.Platform,
			Subject: e.Subject, ReceivedAt: e.ReceivedAt, Read: e.Read,
			ForwardedAt: e.ForwardedAt, ForwardError: e.ForwardError,
			IsOpportunity: e.IsOpportunity, RelevanceScore: e.RelevanceScore, RelevanceReason: e.RelevanceReason,
			RelevanceLink: e.RelevanceLink, ForwardBlocked: e.ForwardBlocked,
		})
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages < 1 {
		totalPages = 1
	}

	return c.JSON(fiber.Map{
		"emails": summaries, "total": total, "page": page, "per_page": perPage, "total_pages": totalPages,
	})
}

// detail — GET /admin/mailbox/:id (body_text inclus). Marque le mail lu — convention client mail :
// ouvrir un mail le marque lu automatiquement.
func (h *MailboxHandler) detail(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	var email models.MailboxEmail
	if err := h.db.First(&email, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if !email.Read {
		h.db.Model(&email).Update("read", true)
		email.Read = true
	}
	// Nettoie le corps À LA CONSULTATION uniquement (cf. services.StripHTMLNoise) — un mail sans
	// partie text/plain (fallback HTML brut, cf. ImapFetcher.extractPlainText) stocke des balises/CSS
	// bruts en base ; jamais montrés tels quels dans le panel. BodyText reste brut en DB (source de
	// vérité), seule cette copie en mémoire de la réponse est nettoyée.
	email.BodyText = services.StripHTMLNoise(email.BodyText)
	return c.JSON(email)
}

// setRead — POST /admin/mailbox/:id/read {"read": bool} — toggle explicite (permet re-marquer non lu,
// contrairement au marquage auto de detail()).
func (h *MailboxHandler) setRead(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	var body struct {
		Read bool `json:"read"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.db.Model(&models.MailboxEmail{}).Where("id = ?", c.Params("id")).
		Update("read", body.Read).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// retryForward — POST /admin/mailbox/:id/forward. 503 explicite si le service (IMAP/SMTP) n'est pas
// configuré — pas un échec de transfert, une indisponibilité de fonctionnalité.
func (h *MailboxHandler) retryForward(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	if h.service == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "mailbox service not configured"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.service.RetryForward(c.Context(), id); err != nil {
		return c.JSON(fiber.Map{"ok": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// validTranslationLangs — langues cibles supportées (locales UI next-intl autres que fr, la langue
// source). Une valeur hors de cette liste est rejetée en 400 plutôt que de laisser passer un code
// arbitraire jusqu'au prompt LLM.
var validTranslationLangs = map[string]bool{"en": true, "de": true, "it": true, "zh": true}

func mailboxTranslationLang(c *fiber.Ctx) (string, bool) {
	lang := c.Query("lang")
	return lang, validTranslationLangs[lang]
}

// getTranslation — GET /admin/mailbox/:id/translation?lang=xx. CACHE-ONLY : ne déclenche jamais de
// traduction (jamais d'appel LLM juste en ouvrant un mail dans le panel) — 404 si rien en cache.
func (h *MailboxHandler) getTranslation(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	lang, ok := mailboxTranslationLang(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or missing lang"})
	}
	var email models.MailboxEmail
	if err := h.db.First(&email, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	t, err := services.GetOrTranslateMailboxEmail(c.Context(), h.db, h.translator, &email, lang, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if t == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not translated yet"})
	}
	return c.JSON(fiber.Map{"subject": t.Subject, "body": t.Body})
}

// translateNow — POST /admin/mailbox/:id/translation?lang=xx (clic explicite "Traduire"). Traduit et
// met en cache si absent, sert le cache directement sinon (jamais retraduit deux fois). 503 explicite
// si le service de traduction n'est pas configuré (credentials Anthropic absentes).
func (h *MailboxHandler) translateNow(c *fiber.Ctx) error {
	if !h.owner(c) {
		return nil
	}
	lang, ok := mailboxTranslationLang(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or missing lang"})
	}
	var email models.MailboxEmail
	if err := h.db.First(&email, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	t, err := services.GetOrTranslateMailboxEmail(c.Context(), h.db, h.translator, &email, lang, true)
	if err != nil {
		if errors.Is(err, services.ErrMailboxTranslationNotConfigured) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "translation service not configured"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"subject": t.Subject, "body": t.Body})
}
