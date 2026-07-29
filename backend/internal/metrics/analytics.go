package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UniqueVisitorsTotal compte le nombre total de visiteurs uniques
	UniqueVisitorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "maicivy_visitors_total",
		Help: "Total number of unique visitors",
	})

	// LettersGeneratedTotal compte les lettres générées par type
	LettersGeneratedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "maicivy_letters_generated_total",
		Help: "Total number of letters generated",
	}, []string{"type"}) // type = motivation | anti_motivation

	// CurrentVisitors indique le nombre de visiteurs actuellement actifs
	CurrentVisitors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "maicivy_current_visitors",
		Help: "Number of current active visitors (last 5 minutes)",
	})

	// AnalyticsRequestDuration mesure la durée des requêtes analytics
	AnalyticsRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "maicivy_analytics_request_duration_seconds",
		Help:    "Duration of analytics API requests in seconds",
		Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
	}, []string{"endpoint", "method", "status"})

	// EventsTotal compte les événements par type
	EventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "maicivy_events_total",
		Help: "Total number of analytics events tracked",
	}, []string{"event_type"})

	// ThemeViews mesure les vues par thème CV
	ThemeViews = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "maicivy_cv_theme_views",
		Help: "Number of views per CV theme",
	}, []string{"theme"})

	// WebSocketConnections compte les connexions WebSocket actives
	WebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "maicivy_websocket_connections",
		Help: "Number of active WebSocket connections",
	})

	// PageViewsTotal compte les page views
	PageViewsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "maicivy_page_views_total",
		Help: "Total number of page views",
	}, []string{"path"})

	// ConversionRate mesure le taux de conversion (lettres/visiteurs)
	ConversionRate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "maicivy_conversion_rate",
		Help: "Conversion rate (letters generated / unique visitors)",
	})

	// RedisOperationDuration mesure la durée des opérations Redis
	RedisOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "maicivy_redis_operation_duration_seconds",
		Help:    "Duration of Redis operations in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"operation"})

	// DatabaseQueryDuration mesure la durée des requêtes PostgreSQL
	DatabaseQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "maicivy_database_query_duration_seconds",
		Help:    "Duration of database queries in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"query_type"})
)

// IncrementEvent incrémente le compteur d'un type d'événement
func IncrementEvent(eventType string) {
	EventsTotal.WithLabelValues(eventType).Inc()
}

// IncrementLetter incrémente le compteur de lettres générées
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

// UpdateWebSocketConnections met à jour le nombre de connexions WebSocket
func UpdateWebSocketConnections(count float64) {
	WebSocketConnections.Set(count)
}

// IncrementPageView incrémente le compteur de page views
func IncrementPageView(path string) {
	PageViewsTotal.WithLabelValues(path).Inc()
}

// UpdateConversionRate met à jour le taux de conversion
func UpdateConversionRate(rate float64) {
	ConversionRate.Set(rate)
}

// ObserveRedisOperation enregistre la durée d'une opération Redis
func ObserveRedisOperation(operation string, duration float64) {
	RedisOperationDuration.WithLabelValues(operation).Observe(duration)
}

// ObserveDatabaseQuery enregistre la durée d'une requête DB
func ObserveDatabaseQuery(queryType string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(queryType).Observe(duration)
}
