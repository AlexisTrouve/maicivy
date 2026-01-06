package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// AnalyticsService gère la collecte et l'agrégation des analytics
type AnalyticsService struct {
	db          *gorm.DB
	redis       *redis.Client
	pubSubTopic string
}

// NewAnalyticsService crée une nouvelle instance du service analytics
func NewAnalyticsService(db *gorm.DB, rdb *redis.Client) *AnalyticsService {
	return &AnalyticsService{
		db:          db,
		redis:       rdb,
		pubSubTopic: "analytics:realtime",
	}
}

// TrackEvent enregistre un événement utilisateur
func (s *AnalyticsService) TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	// 1. Sauvegarder en PostgreSQL (événement brut)
	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		log.Error().Err(err).Msg("Failed to save analytics event")
		return fmt.Errorf("failed to save event: %w", err)
	}

	// 2. Mettre à jour agrégations Redis (non bloquant)
	if err := s.updateRedisAggregations(ctx, event); err != nil {
		// Non bloquant : on log mais ne fail pas
		log.Warn().Err(err).Msg("Failed to update Redis aggregations")
	}

	// 3. Publier événement temps réel via Pub/Sub
	if err := s.publishRealtimeEvent(ctx, event); err != nil {
		log.Warn().Err(err).Msg("Failed to publish realtime event")
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
	case models.EventTypeCVThemeChange:
		// Extraire le thème depuis event_data (JSONB string)
		var eventData map[string]interface{}
		if err := json.Unmarshal([]byte(event.EventData), &eventData); err == nil {
			if theme, ok := eventData["theme"].(string); ok {
				// Top thèmes consultés (Sorted Set)
				pipe.ZIncrBy(ctx, "analytics:themes:top", 1, theme)
			}
		}

	case models.EventTypeLetterGenerate:
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
		"type":       string(event.EventType),
		"visitor_id": event.VisitorID,
		"timestamp":  event.CreatedAt,
		"page_url":   event.PageURL,
	}

	// Parse event_data si présent
	if event.EventData != "" {
		var eventData map[string]interface{}
		if err := json.Unmarshal([]byte(event.EventData), &eventData); err == nil {
			payload["data"] = eventData
		}
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

	// Total events aujourd'hui
	dayKey := fmt.Sprintf("analytics:stats:day:%s:total_events", time.Now().Format("2006-01-02"))
	totalEventsCmd := pipe.Get(ctx, dayKey)

	// Lettres générées aujourd'hui
	lettersKey := fmt.Sprintf("analytics:stats:day:%s:letters_generated", time.Now().Format("2006-01-02"))
	lettersCmd := pipe.Get(ctx, lettersKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	totalEvents, _ := totalEventsCmd.Int64()
	lettersGenerated, _ := lettersCmd.Int64()
	uniqueToday := uniqueTodayCmd.Val()

	// Fallback: si Redis n'a pas de visiteurs uniques aujourd'hui, lire depuis PostgreSQL
	if uniqueToday == 0 {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var count int64
		s.db.WithContext(ctx).Model(&models.Visitor{}).
			Where("created_at >= ?", startOfDay).
			Count(&count)
		uniqueToday = count
	}

	// Fallback: compter les lettres d'aujourd'hui depuis PostgreSQL
	if lettersGenerated == 0 {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var count int64
		s.db.WithContext(ctx).Model(&models.GeneratedLetter{}).
			Where("created_at >= ?", startOfDay).
			Count(&count)
		lettersGenerated = count
	}

	stats := map[string]interface{}{
		"current_visitors": currentVisitorsCmd.Val(),
		"unique_today":     uniqueToday,
		"total_events":     totalEvents,
		"letters_today":    lettersGenerated,
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
	uniqueVisitors := uniqueVisitorsCmd.Val()

	// Fallback: si Redis n'a pas de visiteurs, lire depuis PostgreSQL
	if uniqueVisitors == 0 {
		var startDate time.Time
		switch period {
		case "day":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "week":
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			startDate = now.AddDate(0, 0, -(weekday - 1))
			startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		case "month":
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}

		// Compter les visiteurs uniques depuis PostgreSQL
		var count int64
		s.db.WithContext(ctx).Model(&models.Visitor{}).
			Where("created_at >= ?", startDate).
			Count(&count)
		uniqueVisitors = count
	}

	// Fallback: compter les lettres depuis PostgreSQL si Redis vide
	if lettersGenerated == 0 {
		var startDate time.Time
		switch period {
		case "day":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "week":
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			startDate = now.AddDate(0, 0, -(weekday - 1))
			startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
		case "month":
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}

		var count int64
		s.db.WithContext(ctx).Model(&models.GeneratedLetter{}).
			Where("created_at >= ?", startDate).
			Count(&count)
		lettersGenerated = count
	}

	// Calculer taux de conversion
	conversionRate := 0.0
	if uniqueVisitors > 0 {
		conversionRate = float64(lettersGenerated) / float64(uniqueVisitors)
	}

	stats := map[string]interface{}{
		"period":            period,
		"period_key":        periodKey,
		"total_events":      totalEvents,
		"letters_generated": lettersGenerated,
		"unique_visitors":   uniqueVisitors,
		"conversion_rate":   conversionRate,
	}

	return stats, nil
}

// GetTopThemes récupère les thèmes CV les plus consultés
func (s *AnalyticsService) GetTopThemes(ctx context.Context, limit int64) ([]map[string]interface{}, error) {
	// Sorted Set avec scores = nombre de vues
	themes, err := s.redis.ZRevRangeWithScores(ctx, "analytics:themes:top", 0, limit-1).Result()
	if err != nil {
		if err == redis.Nil {
			return []map[string]interface{}{}, nil
		}
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

// GetLettersStats récupère les statistiques des lettres générées
func (s *AnalyticsService) GetLettersStats(ctx context.Context, period string) (map[string]interface{}, error) {
	stats, err := s.GetStats(ctx, period)
	if err != nil {
		return nil, err
	}

	// Récupérer également les stats par type depuis PostgreSQL
	var motivationCount int64
	var antiMotivationCount int64

	var startDate time.Time
	now := time.Now()

	switch period {
	case "day":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		// Début de semaine (lundi)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = now.AddDate(0, 0, -(weekday - 1))
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	// Query motivation letters
	s.db.WithContext(ctx).Model(&models.GeneratedLetter{}).
		Where("letter_type = ? AND created_at >= ?", "motivation", startDate).
		Count(&motivationCount)

	// Query anti-motivation letters
	s.db.WithContext(ctx).Model(&models.GeneratedLetter{}).
		Where("letter_type = ? AND created_at >= ?", "anti_motivation", startDate).
		Count(&antiMotivationCount)

	// Get history data (letters per day for the period)
	history, err := s.getLettersHistory(ctx, period, startDate)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get letters history")
		history = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"period":          period,
		"total":           stats["letters_generated"],
		"motivation":      motivationCount,
		"anti_motivation": antiMotivationCount,
		"unique_visitors": stats["unique_visitors"],
		"conversion_rate": stats["conversion_rate"],
		"history":         history,
	}, nil
}

// getLettersHistory récupère l'historique des lettres générées par jour
func (s *AnalyticsService) getLettersHistory(ctx context.Context, period string, startDate time.Time) ([]map[string]interface{}, error) {
	// Query PostgreSQL for daily counts
	type DailyCount struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:count"`
	}

	var dailyCounts []DailyCount

	// Use date_trunc to group by day
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM generated_letters
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	if err := s.db.WithContext(ctx).Raw(query, startDate).Scan(&dailyCounts).Error; err != nil {
		return nil, err
	}

	// Convert to expected format
	history := make([]map[string]interface{}, len(dailyCounts))
	for i, dc := range dailyCounts {
		history[i] = map[string]interface{}{
			"date":  dc.Date,
			"count": dc.Count,
		}
	}

	// If no data, create empty entries for each day in the period
	if len(history) == 0 {
		now := time.Now()
		var days int
		switch period {
		case "day":
			days = 1
		case "week":
			days = 7
		case "month":
			days = 30
		}

		history = make([]map[string]interface{}, days)
		for i := 0; i < days; i++ {
			date := now.AddDate(0, 0, -(days - 1 - i))
			history[i] = map[string]interface{}{
				"date":  date.Format("2006-01-02"),
				"count": 0,
			}
		}
	}

	return history, nil
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
func (s *AnalyticsService) CleanupOldEvents(ctx context.Context) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -90)

	result := s.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&models.AnalyticsEvent{})

	if result.Error != nil {
		return 0, result.Error
	}

	log.Info().
		Int64("deleted", result.RowsAffected).
		Msg("Cleaned up old analytics events")

	return result.RowsAffected, nil
}

// GetTimeline récupère les événements récents pour une timeline
func (s *AnalyticsService) GetTimeline(ctx context.Context, limit int, offset int) ([]models.AnalyticsEvent, error) {
	var events []models.AnalyticsEvent

	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetHeatmapData récupère les données pour une heatmap d'interactions
func (s *AnalyticsService) GetHeatmapData(ctx context.Context, pageURL string, hours int) ([]map[string]interface{}, error) {
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	var events []models.AnalyticsEvent
	query := s.db.WithContext(ctx).
		Where("created_at >= ?", cutoffTime).
		Where("event_type IN ?", []models.EventType{
			models.EventTypeButtonClick,
			models.EventTypeLinkClick,
		})

	if pageURL != "" {
		query = query.Where("page_url = ?", pageURL)
	}

	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}

	// Agréger les interactions par position
	heatmap := make(map[string]int)
	for _, event := range events {
		if event.EventData != "" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(event.EventData), &data); err == nil {
				if x, okX := data["x"].(float64); okX {
					if y, okY := data["y"].(float64); okY {
						key := fmt.Sprintf("%.0f,%.0f", x, y)
						heatmap[key]++
					}
				}
			}
		}
	}

	// Convertir en slice
	result := make([]map[string]interface{}, 0, len(heatmap))
	for key, count := range heatmap {
		var x, y float64
		fmt.Sscanf(key, "%f,%f", &x, &y)
		result = append(result, map[string]interface{}{
			"x":     x,
			"y":     y,
			"count": count,
		})
	}

	return result, nil
}
