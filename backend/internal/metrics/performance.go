package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Request metrics
var (
	// RequestDuration tracks HTTP request latencies
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latencies in seconds",
			// Buckets optimized for web app latencies (10ms to 5s)
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint", "status"},
	)

	// RequestsTotal counts total HTTP requests
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// RequestSize tracks HTTP request body sizes
	RequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request body size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// ResponseSize tracks HTTP response body sizes
	ResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response body size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "endpoint"},
	)
)

// Database metrics
var (
	// DBConnections tracks active database connections
	DBConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Active database connections in the pool",
		},
		[]string{"pool", "state"}, // state: idle, active, total
	)

	// DBQueryDuration tracks database query execution time
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_query_duration_seconds",
			Help: "Database query execution duration",
			// Buckets optimized for DB queries (1ms to 500ms)
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		},
		[]string{"query_type", "table"},
	)

	// DBQueriesTotal counts total database queries
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total database queries executed",
		},
		[]string{"query_type", "table", "status"},
	)

	// DBErrors counts database errors
	DBErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total database errors",
		},
		[]string{"query_type", "error_type"},
	)
)

// Cache metrics
var (
	// CacheHits counts cache hit events
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_name", "key_pattern"},
	)

	// CacheMisses counts cache miss events
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_name", "key_pattern"},
	)

	// CacheOperationDuration tracks cache operation latencies
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation duration",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05},
		},
		[]string{"cache_name", "operation"}, // operation: get, set, delete
	)

	// CacheSize tracks cache size (entries count)
	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_size_entries",
			Help: "Number of entries in cache",
		},
		[]string{"cache_name"},
	)

	// CacheEvictions counts cache evictions
	CacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_evictions_total",
			Help: "Total cache evictions",
		},
		[]string{"cache_name", "reason"}, // reason: ttl_expired, max_size, manual
	)
)

// AI Service metrics
var (
	// AIRequestDuration tracks AI API request latencies
	AIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ai_request_duration_seconds",
			Help: "AI API request duration",
			// AI requests can take 5-30 seconds
			Buckets: []float64{1, 2, 5, 10, 15, 20, 30, 60},
		},
		[]string{"provider", "model", "request_type"},
	)

	// AIRequestsTotal counts total AI API requests
	AIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_requests_total",
			Help: "Total AI API requests",
		},
		[]string{"provider", "model", "status"},
	)

	// AITokensUsed tracks token usage
	AITokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_tokens_used_total",
			Help: "Total AI tokens used",
		},
		[]string{"provider", "model", "token_type"}, // token_type: prompt, completion
	)

	// AICostEstimate tracks estimated AI costs
	AICostEstimate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_cost_estimate_usd",
			Help: "Estimated AI cost in USD",
		},
		[]string{"provider", "model"},
	)
)

// Application metrics
var (
	// LettersGenerated counts generated letters
	LettersGenerated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "letters_generated_total",
			Help: "Total letters generated",
		},
		[]string{"letter_type", "status"},
	)

	// VisitorsActive tracks currently active visitors
	VisitorsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "visitors_active",
			Help: "Currently active visitors",
		},
	)

	// VisitorsTotal counts total visitors
	VisitorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "visitors_total",
			Help: "Total unique visitors",
		},
	)

	// CVViewsTotal counts CV page views
	CVViewsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cv_views_total",
			Help: "Total CV page views",
		},
		[]string{"theme"},
	)

	// PDFExportsTotal counts PDF exports
	PDFExportsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_exports_total",
			Help: "Total PDF exports",
		},
		[]string{"export_type"}, // export_type: cv, letter
	)
)

// System metrics (Go runtime)
var (
	// GoroutinesActive tracks active goroutines
	GoroutinesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_active",
			Help: "Number of active goroutines",
		},
	)

	// MemoryAllocated tracks memory allocated
	MemoryAllocated = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_allocated_bytes",
			Help: "Memory allocated in bytes",
		},
	)

	// GCPauses tracks garbage collection pause durations
	GCPauses = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gc_pause_duration_seconds",
			Help:    "Garbage collection pause duration",
			Buckets: []float64{0.00001, 0.0001, 0.001, 0.01, 0.1},
		},
	)
)

// Rate limiting metrics
var (
	// RateLimitHits counts rate limit hits
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
		[]string{"limiter_type", "visitor_type"},
	)

	// RateLimitRemaining tracks remaining rate limit quota
	RateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rate_limit_remaining",
			Help: "Remaining rate limit quota",
		},
		[]string{"limiter_type", "visitor_id"},
	)
)

// Helper functions for recording metrics

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, status string, duration float64, requestSize, responseSize int) {
	RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
	RequestsTotal.WithLabelValues(method, endpoint, status).Inc()

	if requestSize > 0 {
		RequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		ResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
	}
}

// RecordDBQuery records database query metrics
func RecordDBQuery(queryType, table, status string, duration float64) {
	DBQueryDuration.WithLabelValues(queryType, table).Observe(duration)
	DBQueriesTotal.WithLabelValues(queryType, table, status).Inc()
}

// RecordCacheHit records cache hit
func RecordCacheHit(cacheName, keyPattern string, duration float64) {
	CacheHits.WithLabelValues(cacheName, keyPattern).Inc()
	CacheOperationDuration.WithLabelValues(cacheName, "get").Observe(duration)
}

// RecordCacheMiss records cache miss
func RecordCacheMiss(cacheName, keyPattern string, duration float64) {
	CacheMisses.WithLabelValues(cacheName, keyPattern).Inc()
	CacheOperationDuration.WithLabelValues(cacheName, "get").Observe(duration)
}

// RecordAIRequest records AI API request
func RecordAIRequest(provider, model, status string, duration float64, promptTokens, completionTokens int) {
	AIRequestDuration.WithLabelValues(provider, model, "letter_generation").Observe(duration)
	AIRequestsTotal.WithLabelValues(provider, model, status).Inc()

	if promptTokens > 0 {
		AITokensUsed.WithLabelValues(provider, model, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		AITokensUsed.WithLabelValues(provider, model, "completion").Add(float64(completionTokens))
	}

	// Estimate cost (example rates)
	// Claude: $0.015/1K tokens (prompt) + $0.075/1K tokens (completion)
	// GPT-4: $0.03/1K tokens (prompt) + $0.06/1K tokens (completion)
	if provider == "anthropic" {
		cost := float64(promptTokens)*0.015/1000 + float64(completionTokens)*0.075/1000
		AICostEstimate.WithLabelValues(provider, model).Add(cost)
	} else if provider == "openai" {
		cost := float64(promptTokens)*0.03/1000 + float64(completionTokens)*0.06/1000
		AICostEstimate.WithLabelValues(provider, model).Add(cost)
	}
}

// CalculateCacheHitRate calculates cache hit rate
func CalculateCacheHitRate(cacheName string) float64 {
	// This is a helper function to calculate hit rate in Grafana
	// Use this query: rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))
	return 0.0 // Placeholder, actual calculation done in Prometheus queries
}
