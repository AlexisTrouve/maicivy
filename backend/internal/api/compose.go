package api

import (
	"github.com/gofiber/fiber/v2"

	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// ComposeHandler — vérification d'un draft pour l'écran /admin/compose.
//
// POURQUOI un endpoint distinct de /chat/stream : l'agent pousse le nouveau draft via son tool
// set_draft (le front l'affiche instantanément depuis l'event tool_call), puis le front demande ICI la
// traduction et la checklist. Découpler les deux évite de ralentir le tour de l'agent avec deux appels
// modèle supplémentaires, et garantit que la vérification part d'un contexte vierge — c'est ce qui la
// rend digne de confiance pour un opérateur qui ne lit pas la langue du message.
//
// Owner-only : même garde que le reste de /admin (cookie HMAC signé).
type ComposeHandler struct {
	svc           *services.ComposeService
	sessionSecret string
	// prospectQuota : plafond quotidien des vérifications d'un PROSPECTEUR (l'owner n'en a pas).
	// nil = non câblé ⇒ un prospecteur est refusé (échec fermé, cf. verify).
	prospectQuota *middleware.ProspectVerifyQuota
}

func NewComposeHandler(svc *services.ComposeService, sessionSecret string) *ComposeHandler {
	return &ComposeHandler{svc: svc, sessionSecret: sessionSecret}
}

// SetProspectVerifyQuota câble le plafond des vérifications des prospecteurs (cf. RegisterAdminSurface).
func (h *ComposeHandler) SetProspectVerifyQuota(q *middleware.ProspectVerifyQuota) {
	h.prospectQuota = q
}

func (h *ComposeHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/admin/compose")
	g.Post("/verify", h.verify)
}

// composeVerifyRequest : l'historique de conversation n'est VOLONTAIREMENT pas accepté. Le front n'a
// aucune raison de l'envoyer et le service ne doit jamais l'injecter dans un prompt (cf. ComposeService).
type composeVerifyRequest struct {
	Draft        string `json:"draft"`
	Signature    string `json:"signature"`
	RetainedRate int    `json:"retained_rate"`
	OperatorLang string `json:"operator_lang"`
	Announce     string `json:"announce"`
	// WantTitle : le front le met à vrai tant que l'atelier n'a pas de nom. Cf. ComposeVerifyInput —
	// nommer à chaque tour ferait changer le repère de l'opératrice sous ses yeux.
	WantTitle bool `json:"want_title"`
}

func (h *ComposeHandler) verify(c *fiber.Ctx) error {
	// Owner OU prospecteur (identité déposée par AdminRoleGuard, compte actif).
	prospect, isProspect := middleware.ProspectFromCtx(c)
	if !GetStatsAuth(c, h.sessionSecret) && !isProspect {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	}

	var req composeVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}
	if req.Draft == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "draft required",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Un message de prospecteur est signé de SON nom : la signature attendue est imposée côté serveur,
	// le front n'a pas à la connaître ni à pouvoir la falsifier.
	if isProspect {
		req.Signature = prospect.Name

		// Chaque vérification appelle le modèle aux frais du propriétaire : un prospecteur a un
		// plafond quotidien, consommé AVANT l'appel. Quota absent ou Redis en panne ⇒ refus franc
		// (503) plutôt qu'un accès illimité ; plafond atteint ⇒ 429 lisible par le front.
		ok, err := h.prospectQuota.Consume(c.Context(), prospect.ID)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "verification quota unavailable",
				"code":  "VERIFY_QUOTA_UNAVAILABLE",
			})
		}
		if !ok {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "daily verification limit reached",
				"code":  "PROSPECT_VERIFY_LIMIT",
			})
		}
	}

	res, err := h.svc.Verify(c.Context(), services.ComposeVerifyInput{
		Draft:        req.Draft,
		Signature:    req.Signature,
		RetainedRate: req.RetainedRate,
		OperatorLang: req.OperatorLang,
		Announce:     req.Announce,
		WantTitle:    req.WantTitle,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "verification failed",
			"code":  "VERIFY_ERROR",
		})
	}

	return c.JSON(res)
}
