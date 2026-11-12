package middleware

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"maicivy/internal/metrics"
	"maicivy/internal/models"
)

// analyticsSink : les deux opérations dont le middleware a besoin. Interface étroite (implémentée
// par *services.AnalyticsService) pour qu'un test puisse vérifier le CONTEXTE transmis.
type analyticsSink interface {
	MarkVisitorActive(ctx context.Context, visitorID uuid.UUID) error
	TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error
}

// delaiEcritureAnalytics : borne des écritures de fond (Redis/Postgres). Elles ne retiennent
// aucune requête ; le délai empêche seulement une goroutine de pendre indéfiniment.
const delaiEcritureAnalytics = 3 * time.Second

// contexteDetache : contexte des écritures analytics lancées en goroutine.
//
// POURQUOI : ces goroutines survivent à la requête. Leur passer c.Context() (le *fasthttp.RequestCtx)
// est FAUX : fasthttp recycle cet objet dès que le handler a rendu, et un appel ultérieur à son
// Done() déréférence un pointeur nil → panic qui TUE le process (une goroutine sans recover).
// Mesuré en prod le 2026-10-11 à 14:16 UTC : AnalyticsService.MarkVisitorActive → ZScore →
// fasthttp.(*RequestCtx).Done → SIGSEGV, backend redémarré, /cv à 30 s+ le temps de rechauffer.
// COMMENT : contexte racine indépendant de la requête, borné par un délai.
func contexteDetache() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), delaiEcritureAnalytics)
}

// Analytics middleware pour tracking automatique des pageviews et métriques
type Analytics struct {
	service analyticsSink
}

// NewAnalytics crée un nouveau middleware analytics
func NewAnalytics(service analyticsSink) *Analytics {
	return &Analytics{
		service: service,
	}
}

// Handler retourne le middleware handler
func (a *Analytics) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer visitor_id depuis le contexte (ajouté par tracking middleware)
		visitorID, ok := c.Locals("visitor_id").(uuid.UUID)
		if !ok {
			// Pas de session, skip analytics mais continuer
			return c.Next()
		}

		// Marquer visiteur comme actif (pour compteur temps réel)
		// Non bloquant, en goroutine — JAMAIS avec c.Context() (cf. contexteDetache).
		go func() {
			ctx, cancel := contexteDetache()
			defer cancel()
			if err := a.service.MarkVisitorActive(ctx, visitorID); err != nil {
				log.Warn().Err(err).Msg("Failed to mark visitor active")
			}
		}()

		// Capturer pageview pour routes trackables
		path := c.Path()
		if !isAPIRoute(path) && !isWebSocketRoute(path) && !isStaticAsset(path) {
			// Créer événement pageview
			eventData := map[string]interface{}{
				"path":       path,
				"method":     c.Method(),
				"user_agent": c.Get("User-Agent"),
			}

			// Extraire query params si présents
			if c.Context().QueryArgs().Len() > 0 {
				queryParams := make(map[string]string)
				c.Context().QueryArgs().VisitAll(func(key, value []byte) {
					queryParams[string(key)] = string(value)
				})
				eventData["query_params"] = queryParams
			}

			eventDataJSON, _ := json.Marshal(eventData)

			// Déterminer le type d'événement
			eventType := models.EventTypePageView

			// Si c'est une requête CV avec un thème, tracker comme theme change
			if strings.HasPrefix(path, "/api/v1/cv") {
				theme := c.Query("theme", "")
				if theme != "" {
					eventType = models.EventTypeCVThemeChange
					eventData["theme"] = theme
					eventDataJSON, _ = json.Marshal(eventData)
				}
			}

			event := &models.AnalyticsEvent{
				VisitorID: visitorID,
				EventType: eventType,
				EventData: string(eventDataJSON),
				PageURL:   c.OriginalURL(),
				Referrer:  c.Get("Referer"),
			}

			// Track event en async (non bloquant) — event est déjà copié hors de c, et le contexte
			// est détaché de la requête (cf. contexteDetache).
			go func() {
				ctx, cancel := contexteDetache()
				defer cancel()
				if err := a.service.TrackEvent(ctx, event); err != nil {
					log.Warn().Err(err).Msg("Failed to track pageview")
				}
			}()

			// Incrémenter métrique Prometheus
			metrics.IncrementPageView(path)
		}

		// Mesurer temps de réponse pour métriques
		start := time.Now()

		// Continuer la chaîne de middlewares
		err := c.Next()

		// Enregistrer durée pour routes analytics
		if strings.HasPrefix(path, "/api/v1/analytics") {
			duration := time.Since(start).Seconds()
			status := c.Response().StatusCode()

			metrics.AnalyticsRequestDuration.
				WithLabelValues(path, c.Method(), string(rune(status))).
				Observe(duration)
		}

		return err
	}
}

// isAPIRoute vérifie si c'est une route API (mais certaines routes API sont trackées)
func isAPIRoute(path string) bool {
	// Track certain API routes that represent meaningful pageviews
	trackableAPIRoutes := []string{
		"/api/v1/cv",
		"/api/v1/letters/generate",
		"/api/v1/letters/pair",
		"/api/v1/timeline",
		"/api/v1/github",
	}

	for _, route := range trackableAPIRoutes {
		if strings.HasPrefix(path, route) {
			return false // Not considered "skip" for these routes
		}
	}

	return strings.HasPrefix(path, "/api/")
}

// isWebSocketRoute vérifie si c'est une route WebSocket
func isWebSocketRoute(path string) bool {
	return strings.HasPrefix(path, "/ws/")
}

// isStaticAsset vérifie si c'est un asset statique
func isStaticAsset(path string) bool {
	staticExtensions := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot", ".map", ".json",
	}

	for _, ext := range staticExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	return false
}
