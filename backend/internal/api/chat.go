package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// chatCooldownDuration : TTL du cooldown posé après chaque message. Volontairement TRÈS court (pas
// 2 minutes comme CV/lettres) — un chat conversationnel doit encaisser des messages rapprochés. Sert
// uniquement à absorber un double-clic/replay accidentel ; le verrou in-flight (anti-TOCTOU, déjà
// dans AIRateLimit) couvre déjà le cas des requêtes concurrentes.
const chatCooldownDuration = 2 * time.Second

// ChatHandler gère les endpoints de la feature chat portfolio
type ChatHandler struct {
	chatService *services.ChatService
	redis       *redis.Client
	ownerAPIKey string
	// ownerModel / visitorModel : modèles des deux tiers (cf. config.OwnerModel / config.ClaudeModel).
	// Plus de littéral en dur ici — ils l'étaient dans trois handlers, ce qui rendait toute montée de
	// version partielle et invisible.
	ownerModel   string
	visitorModel string
	// Ancrage tarifaire de l'écran compose (cf. config.ComposeRate*). Transmis au service uniquement
	// sur un tour compose : le chat public n'a aucune raison de connaître les tarifs pratiqués.
	rateCurrent int
	rateTarget  int
	rateFloor   int // plancher dur de set_rate (cf. config.ComposeRateFloor), owner comme prospecteur
	// needs : range le besoin exprimé par un visiteur pour lui préparer un CV adapté (cf.
	// cv_context.go). nil = l'outil note_visitor_need n'est jamais offert.
	needs NeedRecorder
}

// SetNeedRecorder branche la préparation du CV adapté au besoin du visiteur.
func (h *ChatHandler) SetNeedRecorder(r NeedRecorder) { h.needs = r }

// noteNeedHook : le rappel de l'outil note_visitor_need, ou nil s'il ne doit pas être offert.
//
// QUI y a droit : un VISITEUR du chat public, à session signée. Ni l'owner (ses conversations ne sont
// pas des besoins de recrutement), ni un prospecteur (son onglet Agent passe aussi par ici, il
// travaille pour d'autres), ni le Composer. Sans session signée, le besoin n'aurait aucun cookie
// auquel se rattacher : rien n'est offert.
// COMMENT : visiteur (session + IP) capturé ICI, pendant la requête — le rappel s'exécute dans la
// goroutine du chat, après que Fiber a pu recycler le contexte.
func (h *ChatHandler) noteNeedHook(c *fiber.Ctx, isOwner bool, prospect *middleware.ProspectIdentity, caps services.ChatCapabilities) func(summary, lang string) {
	if h.needs == nil || isOwner || prospect != nil || caps.Compose {
		return nil
	}
	v, ok := visitorOf(c)
	if !ok {
		return nil
	}
	return func(summary, lang string) { recordNeed(h.needs, v, services.NeedSourceChat, summary, lang) }
}

// NewChatHandler crée un ChatHandler
func NewChatHandler(chatService *services.ChatService, redisClient *redis.Client, ownerAPIKey, ownerModel, visitorModel string, rateCurrent, rateTarget, rateFloor int) *ChatHandler {
	return &ChatHandler{
		chatService:  chatService,
		redis:        redisClient,
		ownerAPIKey:  ownerAPIKey,
		ownerModel:   ownerModel,
		visitorModel: visitorModel,
		rateCurrent:  rateCurrent,
		rateTarget:   rateTarget,
		rateFloor:    rateFloor,
	}
}

// ChatRequest body du POST /api/v1/chat/stream
type ChatRequest struct {
	Message string                 `json:"message"`
	History []services.ChatMessage `json:"history"`
	// Images : captures jointes au message COURANT (glisser-déposer, sélecteur de fichier, ou collage
	// presse-papier côté écran compose). Refusées hors mode compose — voir StreamChat.
	Images []services.ChatImage `json:"images,omitempty"`
	// Compose : l'appelant est l'écran /admin/compose et demande les tools de rédaction de draft.
	// N'est honoré que si la requête est owner (cf. StreamChat) — un visiteur qui poste compose:true
	// est traité comme un visiteur normal.
	Compose bool `json:"compose,omitempty"`
}

// StreamChat traite le POST /api/v1/chat/stream et stream la réponse via SSE
func (h *ChatHandler) StreamChat(c *fiber.Ctx) error {
	var req ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Un message vide est acceptable S'IL porte une capture : « regarde ça » est un geste légitime,
	// l'image EST le message. Sans ni texte ni image, il n'y a rien à traiter.
	if len(req.Message) == 0 && len(req.Images) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message cannot be empty",
			"code":  "VALIDATION_ERROR",
		})
	}

	isOwner, _ := c.Locals("is_owner").(bool)
	var prospect *middleware.ProspectIdentity
	if p, ok := middleware.ProspectFromCtx(c); ok {
		prospect = &p
	}
	model, caps := h.turnPolicy(isOwner, prospect, req.Compose)
	caps.NoteNeed = h.noteNeedHook(c, isOwner, prospect, caps)

	// Images : REFUSÉES hors mode compose, et validées avant tout appel modèle.
	// POURQUOI un refus explicite plutôt qu'un filtrage silencieux : écarter une capture en douce
	// ferait répondre l'agent à côté sans que personne comprenne pourquoi — l'opératrice croirait le
	// modèle idiot. Le code d'erreur est stable, le frontend le traduit.
	if len(req.Images) > 0 {
		if !caps.Compose {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "images not allowed",
				"code":  "image_not_allowed",
			})
		}
		if err := services.ValidateChatImages(req.Images); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid image",
				"code":  err.Error(),
			})
		}
	}

	// Headers SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Désactive le buffering nginx pour le SSE

	// Channel de communication avec le ChatService
	eventCh := make(chan services.ChatEvent, 20)

	// Lancer la génération en goroutine
	go h.chatService.Chat(c.Context(), req.Message, req.Images, req.History, model, caps, eventCh)

	// Streamer les events SSE au client
	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		for event := range eventCh {
			data, err := json.Marshal(event)
			if err != nil {
				log.Error().Err(err).Msg("ChatHandler: failed to marshal SSE event")
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				// Client a coupé la connexion — sortir proprement
				log.Debug().Msg("ChatHandler: client disconnected")
				return
			}
			w.Flush()
		}
	})

	// SetBodyStreamWriter est SYNCHRONE (bloque jusqu'à la fin du stream) : à ce point le tour de
	// chat est terminé (succès ou erreur SSE mi-stream — les deux ont consommé du token Claude, donc
	// comptent pareil, pas de distinction). Alimente le budget dédié chat (KeyPrefix "chat") posé par
	// chatRateLimitMW — sans cet appel le quota/circuit-breaker global ne bougerait jamais pour le chat.
	h.incrementRateLimit(c)

	return nil
}

// turnPolicy : modèle et capabilities d'un tour, selon le rôle de l'appelant.
//
// QUOI : owner → modèle owner ; prospecteur ou visiteur → modèle visiteur, JAMAIS le modèle owner.
// Les tools de rédaction (Compose) ne sont accordés qu'à l'écran compose ET à un rôle authentifié
// (owner via cookie admin / X-Owner-Key, prospecteur via cookie signé + compte actif — tous deux
// posés dans les locals par AIRateLimit).
//
// POURQUOI une fonction pure : /chat/stream est partagé par le chat public, le Composer et l'onglet
// Agent (owner ET prospecteur). Isoler la décision la rend testable sans SSE ni Claude. L'Agent d'un
// prospecteur (compose=false) reçoit des capabilities NULLES : exactement le toolset public du
// portfolio, aucun ancrage tarifaire, aucun tool qui écrit.
//
// COMMENT (prospect) : en Composer, plancher tarifaire identique à l'owner et message signé au nom du
// prospecteur (caps.Prospect). Les compteurs prospecteur / breaker global sont gérés en amont par
// AIRateLimit et incrementRateLimit, identiques pour Composer et Agent.
func (h *ChatHandler) turnPolicy(isOwner bool, prospect *middleware.ProspectIdentity, compose bool) (string, services.ChatCapabilities) {
	model := h.visitorModel
	if isOwner {
		model = h.ownerModel
	}
	caps := services.ChatCapabilities{Compose: compose && (isOwner || prospect != nil)}
	if caps.Compose {
		caps.RateCurrent, caps.RateTarget, caps.RateFloor = h.rateCurrent, h.rateTarget, h.rateFloor
		if prospect != nil && !isOwner {
			caps.Prospect = &services.ComposeProspect{Name: prospect.Name, Ref: prospect.Ref}
		}
	}
	return model, caps
}

// incrementRateLimit incrémente les compteurs de rate-limit du budget chat après un tour terminé.
// no-op si redis absent (contexte de test) ; IncrementAIRateLimit ignore lui-même l'owner (bypass).
func (h *ChatHandler) incrementRateLimit(c *fiber.Ctx) {
	if h.redis == nil {
		return
	}
	if err := middleware.IncrementAIRateLimit(c, h.redis, chatCooldownDuration); err != nil {
		log.Error().Err(err).Msg("ChatHandler: failed to increment chat rate limit")
	}
}
