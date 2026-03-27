package api

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// ChatHandler gère les endpoints de la feature chat portfolio
type ChatHandler struct {
	chatService *services.ChatService
	ownerAPIKey string
}

// NewChatHandler crée un ChatHandler
func NewChatHandler(chatService *services.ChatService, ownerAPIKey string) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		ownerAPIKey: ownerAPIKey,
	}
}

// ChatRequest body du POST /api/v1/chat/stream
type ChatRequest struct {
	Message string                 `json:"message"`
	History []services.ChatMessage `json:"history"`
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

	if len(req.Message) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message cannot be empty",
			"code":  "VALIDATION_ERROR",
		})
	}

	// Sélection modèle : owner → Opus, visiteur → Haiku
	model := "claude-haiku-4-5-20251001"
	if isOwner, _ := c.Locals("is_owner").(bool); isOwner {
		model = "claude-opus-4-6"
	}

	// Headers SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Désactive le buffering nginx pour le SSE

	// Channel de communication avec le ChatService
	eventCh := make(chan services.ChatEvent, 20)

	// Lancer la génération en goroutine
	go h.chatService.Chat(c.Context(), req.Message, req.History, model, eventCh)

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

	return nil
}
