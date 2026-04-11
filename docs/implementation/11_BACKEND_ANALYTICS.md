# 11. BACKEND_ANALYTICS

## 📋 Métadonnées

- **Phase:** 4
- **Priorité:** MOYENNE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 04. BACKEND_MIDDLEWARES.md
- **Temps estimé:** 4-5 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Implémenter un système complet d'analytics publiques en temps réel permettant de :
- Collecter et agréger les événements utilisateurs (page views, clicks, interactions)
- Fournir des métriques temps réel via WebSocket
- Exposer des statistiques agrégées via API REST
- Optimiser les performances avec Redis pour le caching et les agrégations
- Intégrer des métriques Prometheus pour le monitoring système
- Gérer la rétention des données avec archivage automatique

---

## 🏗️ Architecture

### Vue d'Ensemble

```
┌─────────────┐
│   Visiteur  │
└──────┬──────┘
       │ Events
       ▼
┌─────────────────────────────────────┐
│   Middleware Tracking                │
│   (capture automatique pageviews)   │
└──────────┬──────────────────────────┘
           │
           ▼
    ┌──────────────────┐
    │ Analytics Service │
    └──────┬───────────┘
           │
     ┌─────┴─────┐
     ▼           ▼
┌─────────┐  ┌──────────┐
│  Redis  │  │PostgreSQL│
│(Realtime)│  │ (Events) │
└────┬────┘  └─────┬────┘
     │             │
     │             ▼
     │      ┌────────────────┐
     │      │ Data Retention │
     │      │  (90j → 1an)   │
     │      └────────────────┘
     │
     ▼
┌────────────────┐
│  WebSocket     │
│  /ws/analytics │
└────┬───────────┘
     │
     ▼
┌─────────────┐
│  Dashboard  │
│   Frontend  │
└─────────────┘

┌──────────────┐
│  Prometheus  │
│  (metrics)   │
└──────────────┘
```

### Design Decisions

**1. Architecture Dual Storage (Redis + PostgreSQL)**
- **Redis** : Agrégations temps réel, compteurs, HyperLogLog (visiteurs uniques)
- **PostgreSQL** : Stockage permanent des événements bruts avec rétention

**2. WebSocket pour Temps Réel**
- Broadcast des métriques à tous les clients connectés
- Pub/Sub Redis pour communication multi-instances (scalabilité horizontale)
- Heartbeat mechanism pour détecter connexions mortes

**3. Métriques Prometheus**
- Intégration native pour monitoring applicatif
- Exposition des custom metrics business (visiteurs, lettres générées, etc.)

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# WebSocket
go get github.com/gofiber/contrib/websocket

# Prometheus metrics
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp

# Redis (déjà installé Phase 1)
go get github.com/redis/go-redis/v9

# GORM (déjà installé Phase 1)
go get gorm.io/gorm
```

### Services Externes

- **Redis** : Configuré en Phase 1 (persistence RDB/AOF activée)
- **PostgreSQL** : Configuré en Phase 1
- **Prometheus** : À configurer en Phase 6 (mais métriques préparées dès maintenant)

---

## 🔨 Implémentation

### Étape 1: Modèle de Données PostgreSQL

**Description:** Créer le modèle GORM pour stocker les événements analytics

**Code:**

```go
// backend/internal/models/analytics.go
package models

import (
	"time"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

// AnalyticsEvent représente un événement utilisateur capturé
type AnalyticsEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VisitorID uuid.UUID `gorm:"type:uuid;index;not null"` // FK vers visitors
	EventType string    `gorm:"type:varchar(50);not null;index"`
	EventData map[string]interface{} `gorm:"type:jsonb"` // Données custom de l'événement
	PageURL   string    `gorm:"type:text"`
	Referrer  string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime;index"`
}

// TableName override pour correspondre au schéma
func (AnalyticsEvent) TableName() string {
	return "analytics_events"
}

// BeforeCreate hook pour valider les données
func (e *AnalyticsEvent) BeforeCreate(tx *gorm.DB) error {
	if e.EventType == "" {
		return fmt.Errorf("event_type cannot be empty")
	}
	return nil
}
```

**Migration SQL:**

```sql
-- backend/migrations/000007_create_analytics_events.up.sql
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    page_url TEXT,
    referrer TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes pour performance
CREATE INDEX idx_analytics_events_visitor_id ON analytics_events(visitor_id);
CREATE INDEX idx_analytics_events_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at);
CREATE INDEX idx_analytics_events_type_created ON analytics_events(event_type, created_at);

-- Index GIN pour recherches JSONB
CREATE INDEX idx_analytics_events_data ON analytics_events USING GIN(event_data);
```

```sql
-- backend/migrations/000007_create_analytics_events.down.sql
DROP TABLE IF EXISTS analytics_events;
```

**Explications:**
- `event_type` : pageview, click, cv_theme_change, letter_generated, etc.
- `event_data` : JSONB flexible pour stocker des données custom par type d'événement
- Indexes multiples pour optimiser les requêtes d'agrégation par période/type

---

### Étape 2: Service Analytics Core

**Description:** Service central pour gérer collecte, agrégation et stockage

**Code:**

```go
// backend/internal/services/analytics.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/pkg/logger"
)

type AnalyticsService struct {
	db          *gorm.DB
	redis       *redis.Client
	log         *logger.Logger
	pubSubTopic string
}

func NewAnalyticsService(db *gorm.DB, rdb *redis.Client, log *logger.Logger) *AnalyticsService {
	return &AnalyticsService{
		db:          db,
		redis:       rdb,
		log:         log,
		pubSubTopic: "analytics:realtime",
	}
}

// TrackEvent enregistre un événement utilisateur
func (s *AnalyticsService) TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	// 1. Sauvegarder en PostgreSQL (événement brut)
	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		s.log.Error("Failed to save analytics event", "error", err)
		return fmt.Errorf("failed to save event: %w", err)
	}

	// 2. Mettre à jour agrégations Redis
	if err := s.updateRedisAggregations(ctx, event); err != nil {
		// Non bloquant : on log mais ne fail pas
		s.log.Warn("Failed to update Redis aggregations", "error", err)
	}

	// 3. Publier événement temps réel via Pub/Sub
	if err := s.publishRealtimeEvent(ctx, event); err != nil {
		s.log.Warn("Failed to publish realtime event", "error", err)
	}

	return nil
}

// updateRedisAggregations met à jour les compteurs Redis
func (s *AnalyticsService) updateRedisAggregations(ctx context.Context, event *models.AnalyticsEvent) error {
	pipe := s.redis.Pipeline()
	now := time.Now()

	// Clés de temps
	dayKey := fmt.Sprintf("analytics:stats:day:%s", now.Format("2006-01-02"))
	weekKey := fmt.Sprintf("analytics:stats:week:%s", now.Format("2006-W01"))
	monthKey := fmt.Sprintf("analytics:stats:month:%s", now.Format("2006-01"))

	// Incrémentation compteurs globaux
	pipe.Incr(ctx, fmt.Sprintf("%s:total_events", dayKey))
	pipe.Incr(ctx, fmt.Sprintf("%s:total_events", weekKey))
	pipe.Incr(ctx, fmt.Sprintf("%s:total_events", monthKey))

	// TTL sur les clés (cleanup auto)
	pipe.Expire(ctx, dayKey+":total_events", 90*24*time.Hour)   // 90 jours
	pipe.Expire(ctx, weekKey+":total_events", 365*24*time.Hour) // 1 an
	pipe.Expire(ctx, monthKey+":total_events", 365*24*time.Hour)

	// Compteur visiteurs uniques (HyperLogLog)
	pipe.PFAdd(ctx, "analytics:visitors:unique:day:"+now.Format("2006-01-02"), event.VisitorID.String())
	pipe.PFAdd(ctx, "analytics:visitors:unique:week:"+now.Format("2006-W01"), event.VisitorID.String())
	pipe.PFAdd(ctx, "analytics:visitors:unique:month:"+now.Format("2006-01"), event.VisitorID.String())

	// Événements spécifiques
	switch event.EventType {
	case "cv_theme_view":
		if theme, ok := event.EventData["theme"].(string); ok {
			// Top thèmes consultés (Sorted Set)
			pipe.ZIncrBy(ctx, "analytics:themes:top", 1, theme)
		}
	case "letter_generated":
		pipe.Incr(ctx, fmt.Sprintf("%s:letters_generated", dayKey))
		pipe.Incr(ctx, fmt.Sprintf("%s:letters_generated", weekKey))
		pipe.Incr(ctx, fmt.Sprintf("%s:letters_generated", monthKey))
	}

	_, err := pipe.Exec(ctx)
	return err
}

// publishRealtimeEvent publie via Redis Pub/Sub pour WebSocket
func (s *AnalyticsService) publishRealtimeEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	payload := map[string]interface{}{
		"type":       event.EventType,
		"visitor_id": event.VisitorID,
		"timestamp":  event.CreatedAt,
		"data":       event.EventData,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.redis.Publish(ctx, s.pubSubTopic, data).Err()
}

// GetRealtimeStats récupère les stats temps réel depuis Redis
func (s *AnalyticsService) GetRealtimeStats(ctx context.Context) (map[string]interface{}, error) {
	pipe := s.redis.Pipeline()

	// Visiteurs actuels (Set avec TTL 5 minutes)
	currentVisitorsCmd := pipe.SCard(ctx, "analytics:realtime:visitors")

	// Visiteurs uniques aujourd'hui (HyperLogLog)
	uniqueTodayCmd := pipe.PFCount(ctx, "analytics:visitors:unique:day:"+time.Now().Format("2006-01-02"))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"current_visitors": currentVisitorsCmd.Val(),
		"unique_today":     uniqueTodayCmd.Val(),
		"timestamp":        time.Now().Unix(),
	}

	return stats, nil
}

// GetStats récupère statistiques agrégées par période
func (s *AnalyticsService) GetStats(ctx context.Context, period string) (map[string]interface{}, error) {
	var periodKey string
	now := time.Now()

	switch period {
	case "day":
		periodKey = now.Format("2006-01-02")
	case "week":
		periodKey = now.Format("2006-W01")
	case "month":
		periodKey = now.Format("2006-01")
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	pipe := s.redis.Pipeline()

	totalEventsCmd := pipe.Get(ctx, fmt.Sprintf("analytics:stats:%s:%s:total_events", period, periodKey))
	lettersCmd := pipe.Get(ctx, fmt.Sprintf("analytics:stats:%s:%s:letters_generated", period, periodKey))
	uniqueVisitorsCmd := pipe.PFCount(ctx, fmt.Sprintf("analytics:visitors:unique:%s:%s", period, periodKey))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	totalEvents, _ := totalEventsCmd.Int64()
	lettersGenerated, _ := lettersCmd.Int64()

	stats := map[string]interface{}{
		"period":            period,
		"total_events":      totalEvents,
		"letters_generated": lettersGenerated,
		"unique_visitors":   uniqueVisitorsCmd.Val(),
	}

	return stats, nil
}

// GetTopThemes récupère les thèmes CV les plus consultés
func (s *AnalyticsService) GetTopThemes(ctx context.Context, limit int64) ([]map[string]interface{}, error) {
	// Sorted Set avec scores = nombre de vues
	themes, err := s.redis.ZRevRangeWithScores(ctx, "analytics:themes:top", 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(themes))
	for i, theme := range themes {
		result[i] = map[string]interface{}{
			"theme": theme.Member,
			"views": int64(theme.Score),
		}
	}

	return result, nil
}

// MarkVisitorActive marque un visiteur comme actif (temps réel)
func (s *AnalyticsService) MarkVisitorActive(ctx context.Context, visitorID uuid.UUID) error {
	// Ajouter au Set avec TTL 5 minutes
	key := "analytics:realtime:visitors"
	pipe := s.redis.Pipeline()
	pipe.SAdd(ctx, key, visitorID.String())
	pipe.Expire(ctx, key, 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// CleanupOldEvents nettoie les événements > 90 jours
func (s *AnalyticsService) CleanupOldEvents(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -90)

	result := s.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&models.AnalyticsEvent{})

	if result.Error != nil {
		return result.Error
	}

	s.log.Info("Cleaned up old analytics events", "deleted", result.RowsAffected)
	return nil
}
```

**Explications:**
- **TrackEvent** : Point d'entrée unique pour tous les événements
- **HyperLogLog** : Structure Redis probabiliste pour comptage visiteurs uniques (mémoire optimale)
- **Sorted Sets** : Classement automatique des top thèmes
- **Pipeline** : Batch Redis commands pour performance
- **TTL automatiques** : Pas besoin de cron pour cleanup Redis

---

### Étape 3: API REST Endpoints

**Description:** Endpoints pour récupérer les statistiques

**Code:**

```go
// backend/internal/api/analytics.go
package api

import (
	"github.com/gofiber/fiber/v2"
	"maicivy/internal/models"
	"maicivy/internal/services"
	"maicivy/pkg/logger"
)

type AnalyticsHandler struct {
	service *services.AnalyticsService
	log     *logger.Logger
}

func NewAnalyticsHandler(service *services.AnalyticsService, log *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes enregistre les routes analytics
func (h *AnalyticsHandler) RegisterRoutes(router fiber.Router) {
	analytics := router.Group("/analytics")

	analytics.Get("/realtime", h.GetRealtimeStats)
	analytics.Get("/stats", h.GetStats)
	analytics.Get("/themes", h.GetTopThemes)
	analytics.Get("/letters", h.GetLettersStats)
	analytics.Post("/event", h.TrackEvent)
}

// GetRealtimeStats récupère les métriques temps réel
// GET /api/analytics/realtime
func (h *AnalyticsHandler) GetRealtimeStats(c *fiber.Ctx) error {
	stats, err := h.service.GetRealtimeStats(c.Context())
	if err != nil {
		h.log.Error("Failed to get realtime stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetStats récupère statistiques agrégées
// GET /api/analytics/stats?period=day|week|month
func (h *AnalyticsHandler) GetStats(c *fiber.Ctx) error {
	period := c.Query("period", "day")

	if period != "day" && period != "week" && period != "month" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid period. Must be: day, week, or month",
		})
	}

	stats, err := h.service.GetStats(c.Context(), period)
	if err != nil {
		h.log.Error("Failed to get stats", "error", err, "period", period)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetTopThemes récupère les thèmes CV les plus consultés
// GET /api/analytics/themes?limit=5
func (h *AnalyticsHandler) GetTopThemes(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 5)
	if limit < 1 || limit > 20 {
		limit = 5
	}

	themes, err := h.service.GetTopThemes(c.Context(), int64(limit))
	if err != nil {
		h.log.Error("Failed to get top themes", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve themes",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    themes,
	})
}

// GetLettersStats récupère stats lettres générées
// GET /api/analytics/letters?period=day|week|month
func (h *AnalyticsHandler) GetLettersStats(c *fiber.Ctx) error {
	period := c.Query("period", "day")

	stats, err := h.service.GetStats(c.Context(), period)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"period":            stats["period"],
			"letters_generated": stats["letters_generated"],
		},
	})
}

// TrackEvent enregistre un événement custom
// POST /api/analytics/event
// Body: { "event_type": "custom_click", "event_data": { "button": "cta" } }
func (h *AnalyticsHandler) TrackEvent(c *fiber.Ctx) error {
	// Récupérer visitor_id depuis le contexte (ajouté par middleware tracking)
	visitorID, ok := c.Locals("visitor_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing visitor session",
		})
	}

	var req struct {
		EventType string                 `json:"event_type" validate:"required"`
		EventData map[string]interface{} `json:"event_data"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	event := &models.AnalyticsEvent{
		VisitorID: visitorID,
		EventType: req.EventType,
		EventData: req.EventData,
		PageURL:   c.Get("Referer"),
	}

	if err := h.service.TrackEvent(c.Context(), event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to track event",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Event tracked",
	})
}
```

**Explications:**
- API publique (pas d'authentification)
- Validation des paramètres (period, limit)
- visitor_id récupéré depuis Locals (ajouté par middleware tracking Phase 1)

---

### Étape 4: WebSocket Temps Réel

**Description:** WebSocket pour broadcast des métriques en temps réel

**Code:**

```go
// backend/internal/websocket/analytics.go
package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/services"
	"maicivy/pkg/logger"
)

type AnalyticsWSHandler struct {
	service   *services.AnalyticsService
	redis     *redis.Client
	log       *logger.Logger
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	broadcast chan []byte
	pubsub    *redis.PubSub
}

func NewAnalyticsWSHandler(service *services.AnalyticsService, rdb *redis.Client, log *logger.Logger) *AnalyticsWSHandler {
	handler := &AnalyticsWSHandler{
		service:   service,
		redis:     rdb,
		log:       log,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 256),
	}

	// Démarrer Pub/Sub listener
	go handler.listenRedisPubSub()
	// Démarrer broadcaster
	go handler.runBroadcaster()
	// Démarrer heartbeat
	go handler.runHeartbeat()

	return handler
}

// RegisterRoutes enregistre la route WebSocket
func (h *AnalyticsWSHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/ws/analytics", websocket.New(h.HandleConnection))
}

// HandleConnection gère une connexion WebSocket
func (h *AnalyticsWSHandler) HandleConnection(c *websocket.Conn) {
	// Ajouter client
	h.clientsMu.Lock()
	h.clients[c] = true
	h.clientsMu.Unlock()

	h.log.Info("WebSocket client connected", "remote_addr", c.RemoteAddr())

	// Envoyer stats initiales
	h.sendInitialStats(c)

	defer func() {
		// Retirer client à la déconnexion
		h.clientsMu.Lock()
		delete(h.clients, c)
		h.clientsMu.Unlock()
		c.Close()
		h.log.Info("WebSocket client disconnected", "remote_addr", c.RemoteAddr())
	}()

	// Boucle de lecture (pour garder connexion active + recevoir pings)
	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.log.Error("WebSocket read error", "error", err)
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
		h.handleClientMessage(c, message)
	}
}

// sendInitialStats envoie les stats initiales à un nouveau client
func (h *AnalyticsWSHandler) sendInitialStats(c *websocket.Conn) {
	stats, err := h.service.GetRealtimeStats(context.Background())
	if err != nil {
		h.log.Error("Failed to get initial stats", "error", err)
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"type": "initial_stats",
		"data": stats,
	})
	if err != nil {
		return
	}

	c.WriteMessage(websocket.TextMessage, data)
}

// listenRedisPubSub écoute les événements Redis Pub/Sub
func (h *AnalyticsWSHandler) listenRedisPubSub() {
	h.pubsub = h.redis.Subscribe(context.Background(), "analytics:realtime")
	defer h.pubsub.Close()

	ch := h.pubsub.Channel()
	for msg := range ch {
		// Transmettre au broadcaster
		h.broadcast <- []byte(msg.Payload)
	}
}

// runBroadcaster diffuse les messages à tous les clients
func (h *AnalyticsWSHandler) runBroadcaster() {
	for message := range h.broadcast {
		h.clientsMu.RLock()
		for client := range h.clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				h.log.Error("Failed to send message to client", "error", err)
				client.Close()
				h.clientsMu.Lock()
				delete(h.clients, client)
				h.clientsMu.Unlock()
			}
		}
		h.clientsMu.RUnlock()
	}
}

// runHeartbeat envoie périodiquement les stats à jour
func (h *AnalyticsWSHandler) runHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := h.service.GetRealtimeStats(context.Background())
		if err != nil {
			h.log.Error("Failed to get stats for heartbeat", "error", err)
			continue
		}

		data, err := json.Marshal(map[string]interface{}{
			"type": "heartbeat",
			"data": stats,
		})
		if err != nil {
			continue
		}

		h.broadcast <- data
	}
}

// handleClientMessage gère les messages reçus du client (optionnel)
func (h *AnalyticsWSHandler) handleClientMessage(c *websocket.Conn, message []byte) {
	var req struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(message, &req); err != nil {
		return
	}

	// Exemple: client demande refresh manuel
	if req.Type == "refresh_stats" {
		h.sendInitialStats(c)
	}
}

// GetConnectedClients retourne le nombre de clients connectés (pour debug)
func (h *AnalyticsWSHandler) GetConnectedClients() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}
```

**Explications:**
- **Pub/Sub Redis** : Communication entre instances backend (scalabilité horizontale)
- **Broadcast channel** : Pattern Go pour diffusion efficace
- **Heartbeat** : Envoi stats toutes les 5s même sans événement
- **Thread-safe** : Mutex pour accès concurrent à la map clients

---

### Étape 5: Métriques Prometheus

**Description:** Exposition de custom metrics pour monitoring

**Code:**

```go
// backend/internal/metrics/analytics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Compteur total visiteurs
	VisitorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "maicivy_visitors_total",
		Help: "Total number of unique visitors",
	})

	// Compteur lettres générées
	LettersGeneratedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "maicivy_letters_generated_total",
		Help: "Total number of letters generated",
	}, []string{"type"}) // type = motivation | anti_motivation

	// Gauge visiteurs actuels
	CurrentVisitors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "maicivy_current_visitors",
		Help: "Number of current active visitors",
	})

	// Histogram temps réponse API analytics
	AnalyticsRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "maicivy_analytics_request_duration_seconds",
		Help:    "Duration of analytics API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint", "method"})

	// Compteur événements par type
	EventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "maicivy_events_total",
		Help: "Total number of analytics events tracked",
	}, []string{"event_type"})

	// Gauge thèmes CV populaires
	ThemeViews = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "maicivy_cv_theme_views",
		Help: "Number of views per CV theme",
	}, []string{"theme"})
)

// IncrementEvent incrémente le compteur d'un type d'événement
func IncrementEvent(eventType string) {
	EventsTotal.WithLabelValues(eventType).Inc()
}

// IncrementLetter incrémente le compteur de lettres
func IncrementLetter(letterType string) {
	LettersGeneratedTotal.WithLabelValues(letterType).Inc()
}

// UpdateCurrentVisitors met à jour le gauge visiteurs actuels
func UpdateCurrentVisitors(count float64) {
	CurrentVisitors.Set(count)
}

// UpdateThemeViews met à jour les vues d'un thème
func UpdateThemeViews(theme string, count float64) {
	ThemeViews.WithLabelValues(theme).Set(count)
}
```

**Intégration dans le service:**

```go
// backend/internal/services/analytics.go (ajout)
import "maicivy/internal/metrics"

func (s *AnalyticsService) TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	// ... code existant ...

	// Incrémenter métrique Prometheus
	metrics.IncrementEvent(event.EventType)

	// Métriques spécifiques
	if event.EventType == "letter_generated" {
		if letterType, ok := event.EventData["type"].(string); ok {
			metrics.IncrementLetter(letterType)
		}
	}

	return nil
}

func (s *AnalyticsService) GetRealtimeStats(ctx context.Context) (map[string]interface{}, error) {
	// ... code existant ...

	// Mettre à jour Gauge Prometheus
	if currentVisitors, ok := stats["current_visitors"].(int64); ok {
		metrics.UpdateCurrentVisitors(float64(currentVisitors))
	}

	return stats, nil
}
```

**Route exposition Prometheus:**

```go
// backend/cmd/main.go (ajout)
import (
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// ... setup existant ...

	// Endpoint Prometheus metrics
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// ... reste du code ...
}
```

**Explications:**
- **Counter** : Métriques cumulatives (lettres générées, événements)
- **Gauge** : Métriques instantanées (visiteurs actuels)
- **Histogram** : Distribution temps de réponse
- **Labels** : Dimensions pour filtrer/grouper (type, endpoint)

---

### Étape 6: Middleware Auto-Tracking Pageviews

**Description:** Middleware pour capturer automatiquement les pageviews

**Code:**

```go
// backend/internal/middleware/analytics.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

func AnalyticsMiddleware(analyticsService *services.AnalyticsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer visitor_id depuis Locals (ajouté par tracking middleware)
		visitorID, ok := c.Locals("visitor_id").(uuid.UUID)
		if !ok {
			// Pas de session, skip analytics
			return c.Next()
		}

		// Marquer visiteur comme actif (pour compteur temps réel)
		go analyticsService.MarkVisitorActive(c.Context(), visitorID)

		// Capturer pageview seulement pour routes HTML (pas API)
		if !isAPIRoute(c.Path()) {
			event := &models.AnalyticsEvent{
				VisitorID: visitorID,
				EventType: "pageview",
				EventData: map[string]interface{}{
					"path":       c.Path(),
					"method":     c.Method(),
					"user_agent": c.Get("User-Agent"),
				},
				PageURL:  c.OriginalURL(),
				Referrer: c.Get("Referer"),
			}

			// Async tracking (non bloquant)
			go analyticsService.TrackEvent(c.Context(), event)
		}

		return c.Next()
	}
}

func isAPIRoute(path string) bool {
	return len(path) >= 4 && path[:4] == "/api"
}
```

**Intégration dans main.go:**

```go
// backend/cmd/main.go
func main() {
	// ... setup ...

	// Middlewares (ordre important)
	app.Use(middleware.TrackingMiddleware(db, redis)) // Phase 1
	app.Use(middleware.AnalyticsMiddleware(analyticsService)) // Nouveau
	app.Use(middleware.RateLimitMiddleware(redis))    // Phase 1

	// ... routes ...
}
```

---

### Étape 7: Job de Cleanup (Data Retention)

**Description:** Tâche cron pour nettoyer les anciennes données

**Code:**

```go
// backend/internal/jobs/analytics_cleanup.go
package jobs

import (
	"context"
	"time"

	"maicivy/internal/services"
	"maicivy/pkg/logger"
)

type AnalyticsCleanupJob struct {
	service *services.AnalyticsService
	log     *logger.Logger
}

func NewAnalyticsCleanupJob(service *services.AnalyticsService, log *logger.Logger) *AnalyticsCleanupJob {
	return &AnalyticsCleanupJob{
		service: service,
		log:     log,
	}
}

// Run exécute le job de nettoyage
func (j *AnalyticsCleanupJob) Run(ctx context.Context) error {
	j.log.Info("Starting analytics cleanup job")

	if err := j.service.CleanupOldEvents(ctx); err != nil {
		j.log.Error("Analytics cleanup failed", "error", err)
		return err
	}

	j.log.Info("Analytics cleanup completed successfully")
	return nil
}

// Start démarre le job en mode cron
func (j *AnalyticsCleanupJob) Start(ctx context.Context) {
	// Exécuter quotidiennement à 2h du matin
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Calculer délai jusqu'à 2h du matin
	now := time.Now()
	next2AM := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	timeUntil2AM := time.Until(next2AM)

	// Première exécution à 2h
	time.AfterFunc(timeUntil2AM, func() {
		j.Run(ctx)
	})

	// Puis toutes les 24h
	for {
		select {
		case <-ticker.C:
			j.Run(ctx)
		case <-ctx.Done():
			j.log.Info("Analytics cleanup job stopped")
			return
		}
	}
}
```

**Lancement dans main.go:**

```go
// backend/cmd/main.go
func main() {
	// ... setup ...

	// Démarrer cleanup job
	cleanupJob := jobs.NewAnalyticsCleanupJob(analyticsService, log)
	go cleanupJob.Start(context.Background())

	// ... reste ...
}
```

---

## 🧪 Tests

### Tests Unitaires Service

```go
// backend/internal/services/analytics_test.go
package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/models"
	"maicivy/internal/services"
	"maicivy/test/testutil"
)

func TestAnalyticsService_TrackEvent(t *testing.T) {
	db, redis, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	log := testutil.NewTestLogger()
	service := services.NewAnalyticsService(db, redis, log)

	// Créer visiteur test
	visitor := &models.Visitor{
		SessionID: uuid.New(),
	}
	require.NoError(t, db.Create(visitor).Error)

	// Créer événement
	event := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: "pageview",
		EventData: map[string]interface{}{
			"path": "/cv",
		},
		PageURL: "https://example.com/cv",
	}

	// Track event
	err := service.TrackEvent(context.Background(), event)
	require.NoError(t, err)

	// Vérifier PostgreSQL
	var savedEvent models.AnalyticsEvent
	err = db.Where("visitor_id = ?", visitor.ID).First(&savedEvent).Error
	require.NoError(t, err)
	assert.Equal(t, "pageview", savedEvent.EventType)

	// Vérifier Redis (compteur)
	dayKey := "analytics:stats:day:" + time.Now().Format("2006-01-02")
	count, err := redis.Get(context.Background(), dayKey+":total_events").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestAnalyticsService_GetTopThemes(t *testing.T) {
	db, redis, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	log := testutil.NewTestLogger()
	service := services.NewAnalyticsService(db, redis, log)

	ctx := context.Background()

	// Insérer données test dans Sorted Set
	redis.ZAdd(ctx, "analytics:themes:top",
		redis.Z{Score: 10, Member: "backend"},
		redis.Z{Score: 5, Member: "frontend"},
		redis.Z{Score: 3, Member: "devops"},
	)

	// Récupérer top 2
	themes, err := service.GetTopThemes(ctx, 2)
	require.NoError(t, err)
	require.Len(t, themes, 2)

	// Vérifier ordre (desc par score)
	assert.Equal(t, "backend", themes[0]["theme"])
	assert.Equal(t, int64(10), themes[0]["views"])
	assert.Equal(t, "frontend", themes[1]["theme"])
}

func TestAnalyticsService_CleanupOldEvents(t *testing.T) {
	db, redis, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	log := testutil.NewTestLogger()
	service := services.NewAnalyticsService(db, redis, log)

	// Créer visiteur
	visitor := &models.Visitor{SessionID: uuid.New()}
	require.NoError(t, db.Create(visitor).Error)

	// Créer événement ancien (95 jours)
	oldEvent := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: "old_event",
		CreatedAt: time.Now().AddDate(0, 0, -95),
	}
	require.NoError(t, db.Create(oldEvent).Error)

	// Créer événement récent
	recentEvent := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: "recent_event",
	}
	require.NoError(t, db.Create(recentEvent).Error)

	// Cleanup
	err := service.CleanupOldEvents(context.Background())
	require.NoError(t, err)

	// Vérifier : ancien supprimé, récent conservé
	var count int64
	db.Model(&models.AnalyticsEvent{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var remaining models.AnalyticsEvent
	db.First(&remaining)
	assert.Equal(t, "recent_event", remaining.EventType)
}
```

### Tests Integration API

```go
// backend/internal/api/analytics_test.go
package api_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/api"
	"maicivy/test/testutil"
)

func TestAnalyticsAPI_GetRealtimeStats(t *testing.T) {
	app, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/analytics/realtime", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	assert.True(t, body["success"].(bool))
	assert.NotNil(t, body["data"])
}

func TestAnalyticsAPI_GetStats_InvalidPeriod(t *testing.T) {
	app, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/analytics/stats?period=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
```

### Commandes Tests

```bash
# Tests unitaires
go test ./internal/services -v -cover

# Tests integration
go test ./internal/api -v -cover

# Coverage global
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## ⚠️ Points d'Attention

- **HyperLogLog Precision** : Erreur ~0.81% sur comptage unique (acceptable pour analytics)
- **Redis Memory** : Surveiller utilisation mémoire (Sorted Sets + HyperLogLog)
- **WebSocket Scalability** : Avec multi-instances, tous partagent via Redis Pub/Sub
- **Event Data JSONB** : Indexer avec parcimonie (GIN index coûteux)
- **Cleanup Job** : Éviter de lancer pendant heures de pointe (2h du matin optimal)
- **Rate Limiting Analytics** : Endpoint POST /event accessible publiquement → risque spam
- **Timezone** : Tous les timestamps en UTC, conversion côté frontend
- **Data Privacy** : Ne pas logger d'IP en clair (hash uniquement, via middleware tracking Phase 1)

**Pièges à éviter:**
- ❌ Oublier TTL sur clés Redis → fuite mémoire
- ❌ Bloquer requête HTTP pour analytics → toujours async
- ❌ Stocker événements sans limite → disk plein (cleanup essentiel)
- ❌ WebSocket sans heartbeat → connexions zombies

**Optimisations:**
- Batch inserts PostgreSQL (si volume élevé) avec buffering
- Utiliser Redis Pipeline pour réduire round-trips
- Index partiel PostgreSQL sur événements récents uniquement
- Compression JSONB pour event_data

---

## 📚 Ressources

**Redis Data Structures:**
- [HyperLogLog](https://redis.io/docs/data-types/probabilistic/hyperloglogs/) - Comptage unique
- [Sorted Sets](https://redis.io/docs/data-types/sorted-sets/) - Top N
- [Pub/Sub](https://redis.io/docs/interact/pubsub/) - Messaging temps réel

**Fiber WebSocket:**
- [gofiber/websocket](https://github.com/gofiber/contrib/tree/main/websocket)
- [WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455)

**Prometheus:**
- [Client Golang](https://github.com/prometheus/client_golang)
- [Metric Types](https://prometheus.io/docs/concepts/metric_types/)
- [Best Practices](https://prometheus.io/docs/practices/naming/)

**PostgreSQL:**
- [JSONB Indexing](https://www.postgresql.org/docs/current/datatype-json.html#JSON-INDEXING)
- [Partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html) - Pour très gros volumes

**Articles:**
- [Real-time Analytics at Scale](https://netflixtechblog.com/scaling-time-series-data-storage-part-i-ec2b6d44ba39)
- [HyperLogLog in Practice](https://research.neustar.biz/2012/10/25/sketch-of-the-day-hyperloglog-cornerstone-of-a-big-data-infrastructure/)

---

## ✅ Checklist de Complétion

- [ ] Migration PostgreSQL `analytics_events` créée et appliquée
- [ ] Service Analytics implémenté avec toutes méthodes
- [ ] Endpoints API REST testés (realtime, stats, themes, letters)
- [ ] WebSocket `/ws/analytics` fonctionnel avec broadcast
- [ ] Redis Pub/Sub configuré et testé
- [ ] Métriques Prometheus exposées sur `/metrics`
- [ ] Middleware auto-tracking pageviews intégré
- [ ] Job cleanup configuré et testé
- [ ] Tests unitaires > 80% coverage
- [ ] Tests integration API passants
- [ ] Documentation API (OpenAPI) mise à jour
- [ ] Logs structurés avec contexte approprié
- [ ] Review sécurité (validation inputs, rate limiting)
- [ ] Review performance (indexes DB, Redis pipelines)
- [ ] Test charge WebSocket (100+ connexions simultanées)
- [ ] Commit & Push

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
