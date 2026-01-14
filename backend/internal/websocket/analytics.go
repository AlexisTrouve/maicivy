package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// AnalyticsWSHandler gère les connexions WebSocket pour analytics temps réel
type AnalyticsWSHandler struct {
	service   *services.AnalyticsService
	redis     *redis.Client
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	broadcast chan []byte
	pubsub    *redis.PubSub
}

// NewAnalyticsWSHandler crée un nouveau handler WebSocket
func NewAnalyticsWSHandler(service *services.AnalyticsService, rdb *redis.Client) *AnalyticsWSHandler {
	handler := &AnalyticsWSHandler{
		service:   service,
		redis:     rdb,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 256),
	}

	// Démarrer les goroutines
	go handler.listenRedisPubSub()
	go handler.runBroadcaster()
	go handler.runHeartbeat()

	return handler
}

// RegisterRoutes enregistre la route WebSocket
func (h *AnalyticsWSHandler) RegisterRoutes(app *fiber.App) {
	// Middleware pour upgrade HTTP -> WebSocket
	app.Use("/ws/analytics", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Route WebSocket
	app.Get("/ws/analytics", websocket.New(h.HandleConnection))
}

// HandleConnection gère une connexion WebSocket
func (h *AnalyticsWSHandler) HandleConnection(c *websocket.Conn) {
	// Ajouter client
	h.clientsMu.Lock()
	h.clients[c] = true
	clientCount := len(h.clients)
	h.clientsMu.Unlock()

	log.Info().
		Str("remote_addr", c.RemoteAddr().String()).
		Int("total_clients", clientCount).
		Msg("WebSocket client connected")

	// Envoyer stats initiales
	h.sendInitialStats(c)

	defer func() {
		// Retirer client à la déconnexion
		h.clientsMu.Lock()
		delete(h.clients, c)
		clientCount := len(h.clients)
		h.clientsMu.Unlock()

		c.Close()

		log.Info().
			Str("remote_addr", c.RemoteAddr().String()).
			Int("total_clients", clientCount).
			Msg("WebSocket client disconnected")
	}()

	// Boucle de lecture (pour garder connexion active + recevoir pings)
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		// Echo pings (heartbeat client)
		if messageType == websocket.PingMessage {
			if err := c.WriteMessage(websocket.PongMessage, nil); err != nil {
				break
			}
		}

		// Gérer messages client (optionnel)
		if messageType == websocket.TextMessage {
			h.handleClientMessage(c, message)
		}
	}
}

// sendInitialStats envoie les stats initiales à un nouveau client
func (h *AnalyticsWSHandler) sendInitialStats(c *websocket.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := h.service.GetRealtimeStats(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get initial stats")
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"type": "initial_stats",
		"data": stats,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal initial stats")
		return
	}

	if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Error().Err(err).Msg("Failed to send initial stats")
	}
}

// listenRedisPubSub écoute les événements Redis Pub/Sub
func (h *AnalyticsWSHandler) listenRedisPubSub() {
	h.pubsub = h.redis.Subscribe(context.Background(), "analytics:realtime")
	defer h.pubsub.Close()

	log.Info().Msg("Started listening to Redis Pub/Sub for analytics events")

	ch := h.pubsub.Channel()
	for msg := range ch {
		// Transmettre au broadcaster
		h.broadcast <- []byte(msg.Payload)
	}

	log.Warn().Msg("Redis Pub/Sub listener stopped")
}

// runBroadcaster diffuse les messages à tous les clients
func (h *AnalyticsWSHandler) runBroadcaster() {
	log.Info().Msg("Started WebSocket broadcaster")

	for message := range h.broadcast {
		h.clientsMu.RLock()
		clientCount := len(h.clients)

		for client := range h.clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Error().
					Err(err).
					Str("remote_addr", client.RemoteAddr().String()).
					Msg("Failed to send message to client")

				// Fermer et retirer le client en cas d'erreur
				client.Close()
				h.clientsMu.RUnlock()
				h.clientsMu.Lock()
				delete(h.clients, client)
				h.clientsMu.Unlock()
				h.clientsMu.RLock()
			}
		}
		h.clientsMu.RUnlock()

		log.Debug().
			Int("clients", clientCount).
			Msg("Broadcasted message to all clients")
	}

	log.Warn().Msg("WebSocket broadcaster stopped")
}

// runHeartbeat envoie périodiquement les stats à jour
func (h *AnalyticsWSHandler) runHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Info().Msg("Started WebSocket heartbeat")

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		stats, err := h.service.GetRealtimeStats(ctx)
		cancel()

		if err != nil {
			log.Error().Err(err).Msg("Failed to get stats for heartbeat")
			continue
		}

		data, err := json.Marshal(map[string]interface{}{
			"type": "heartbeat",
			"data": stats,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal heartbeat stats")
			continue
		}

		// Envoyer via broadcast channel
		select {
		case h.broadcast <- data:
			log.Debug().Msg("Heartbeat sent")
		default:
			log.Warn().Msg("Broadcast channel full, skipping heartbeat")
		}
	}

	log.Warn().Msg("WebSocket heartbeat stopped")
}

// handleClientMessage gère les messages reçus du client
func (h *AnalyticsWSHandler) handleClientMessage(c *websocket.Conn, message []byte) {
	var req struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(message, &req); err != nil {
		log.Warn().Err(err).Msg("Failed to parse client message")
		return
	}

	log.Debug().
		Str("type", req.Type).
		Str("remote_addr", c.RemoteAddr().String()).
		Msg("Received message from client")

	// Traiter selon le type de message
	switch req.Type {
	case "refresh_stats":
		// Client demande un refresh manuel
		h.sendInitialStats(c)

	case "ping":
		// Répondre avec pong
		response, _ := json.Marshal(map[string]interface{}{
			"type": "pong",
			"time": time.Now().Unix(),
		})
		c.WriteMessage(websocket.TextMessage, response)

	default:
		log.Warn().
			Str("type", req.Type).
			Msg("Unknown message type from client")
	}
}

// GetConnectedClients retourne le nombre de clients connectés (pour debug/monitoring)
func (h *AnalyticsWSHandler) GetConnectedClients() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// Close ferme proprement le handler WebSocket
func (h *AnalyticsWSHandler) Close() error {
	log.Info().Msg("Closing WebSocket handler")

	// Fermer tous les clients
	h.clientsMu.Lock()
	for client := range h.clients {
		client.Close()
	}
	h.clients = make(map[*websocket.Conn]bool)
	h.clientsMu.Unlock()

	// Fermer broadcast channel
	close(h.broadcast)

	// Fermer pubsub
	if h.pubsub != nil {
		return h.pubsub.Close()
	}

	return nil
}
