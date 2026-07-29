package api

// Follow par email du blog — phase 1 : CAPTURE des abonnés (l'envoi viendra en phase 2).
// Granularité par TOPIC = project_name d'un post (ex: "Drifterra") → un abonné ne reçoit que ce
// qu'il a choisi ; topics vide = tout. Les topics suivables sont dérivés EN LIVE des posts publiés
// (maiProFiles), donc un nouveau projet apparaît tout seul, sans toucher au code.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/validation"
)

// subscribeRequest — body de POST /api/v1/blog/subscribe.
type subscribeRequest struct {
	Email  string   `json:"email"`
	Topics []string `json:"topics"` // project_name à suivre ; vide = tous les articles
}

// Subscribe enregistre (ou met à jour) un abonné email avec ses topics choisis.
// POST /api/v1/blog/subscribe — PUBLIC.
func (h *BlogHandler) Subscribe(c *fiber.Ctx) error {
	var req subscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Normaliser + valider l'email (réutilise le validateur maison).
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validation.ValidateEmail(email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_email"})
	}

	topics := pq.StringArray(cleanTopics(req.Topics))

	// Upsert par email. POURQUOI pas un INSERT brut : un re-submit du formulaire (email déjà connu)
	// ne doit pas faire une 500 sur la contrainte unique — il met simplement à jour les topics choisis.
	var existing models.BlogSubscriber
	err := h.db.Where("email = ?", email).First(&existing).Error
	switch {
	case err == nil:
		existing.Topics = topics
		if e := h.db.Save(&existing).Error; e != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save_failed"})
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		sub := models.BlogSubscriber{Email: email, Topics: topics, UnsubscribeToken: newToken()}
		if e := h.db.Create(&sub).Error; e != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save_failed"})
		}
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db_error"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Inscription enregistrée"})
}

// Unsubscribe désinscrit un abonné via son token. PUBLIC.
// GET (clic sur le lien dans l'email) ET POST (List-Unsubscribe one-click de Gmail) → même effet.
func (h *BlogHandler) Unsubscribe(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		token = c.FormValue("token")
	}
	if token == "" {
		return c.Status(fiber.StatusBadRequest).SendString("token requis")
	}
	// Hard delete (Unscoped) : on retire vraiment l'abonné. POURQUOI pas un soft-delete : l'index
	// unique sur email bloquerait un futur réabonnement si la ligne soft-deletée gardait l'email.
	h.db.Unscoped().Where("unsubscribe_token = ?", token).Delete(&models.BlogSubscriber{})
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(`<!doctype html><html><body style="font-family:system-ui,sans-serif;text-align:center;padding:48px;color:#111"><h2>Désinscrit.</h2><p style="color:#666">Tu ne recevras plus les nouveaux articles. Tu peux te réabonner quand tu veux depuis le blog.</p></body></html>`)
}

// GetTopics expose les topics suivables = les project_name distincts des articles publiés.
// GET /api/v1/blog/topics — PUBLIC.
func (h *BlogHandler) GetTopics(c *fiber.Ctx) error {
	resp, err := h.mpfClient.GetBlogPosts(context.Background(), 1, 100, "")
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "upstream_failed"})
	}
	return c.JSON(fiber.Map{"topics": distinctProjectNames(resp.Posts)})
}

// distinctProjectNames extrait les project_name uniques (non vides), triés. Pur → testable.
func distinctProjectNames(posts []models.BlogPost) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range posts {
		name := strings.TrimSpace(p.ProjectName)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// cleanTopics : trim + drop des vides + dédup, en préservant l'ordre. Pur → testable.
func cleanTopics(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// newToken génère un jeton aléatoire (lien de désinscription, utilisé en phase 2).
func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
