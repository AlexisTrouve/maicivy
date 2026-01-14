package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupTestDB(t *testing.T) (*gorm.DB, *redis.Client, func()) {
	// Setup SQLite in-memory pour tests
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.Visitor{},
		&models.AnalyticsEvent{},
		&models.GeneratedLetter{},
	)
	require.NoError(t, err)

	// Setup miniredis (Redis mock)
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		mr.Close()
		redisClient.Close()
	}

	return db, redisClient, cleanup
}

func TestAnalyticsService_TrackEvent(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)

	// Créer visiteur test
	visitor := &models.Visitor{
		SessionID:  uuid.New().String(),
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	require.NoError(t, db.Create(visitor).Error)

	// Créer événement
	eventData := map[string]interface{}{
		"path": "/cv",
		"theme": "backend",
	}
	eventDataJSON, _ := json.Marshal(eventData)

	event := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: models.EventTypePageView,
		EventData: string(eventDataJSON),
		PageURL:   "https://example.com/cv",
		Referrer:  "https://google.com",
	}

	// Track event
	err := service.TrackEvent(context.Background(), event)
	require.NoError(t, err)

	// Vérifier PostgreSQL
	var savedEvent models.AnalyticsEvent
	err = db.Where("visitor_id = ?", visitor.ID).First(&savedEvent).Error
	require.NoError(t, err)
	assert.Equal(t, models.EventTypePageView, savedEvent.EventType)
	assert.Equal(t, visitor.ID, savedEvent.VisitorID)

	// Vérifier Redis (compteur total events)
	dayKey := "analytics:stats:day:" + time.Now().Format("2006-01-02") + ":total_events"
	count, err := redisClient.Get(context.Background(), dayKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Vérifier HyperLogLog visiteurs uniques
	uniqueKey := "analytics:visitors:unique:day:" + time.Now().Format("2006-01-02")
	uniqueCount, err := redisClient.PFCount(context.Background(), uniqueKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), uniqueCount)
}

func TestAnalyticsService_GetTopThemes(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)
	ctx := context.Background()

	// Insérer données test dans Sorted Set
	redisClient.ZAdd(ctx, "analytics:themes:top",
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
	assert.Equal(t, int64(5), themes[1]["views"])
}

func TestAnalyticsService_GetRealtimeStats(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)
	ctx := context.Background()

	// Simuler quelques visiteurs actifs
	visitor1 := uuid.New()
	visitor2 := uuid.New()

	redisClient.SAdd(ctx, "analytics:realtime:visitors", visitor1.String(), visitor2.String())

	// Simuler visiteurs uniques aujourd'hui
	today := time.Now().Format("2006-01-02")
	redisClient.PFAdd(ctx, "analytics:visitors:unique:day:"+today, visitor1.String(), visitor2.String())

	// Simuler événements
	dayKey := "analytics:stats:day:" + today
	redisClient.Set(ctx, dayKey+":total_events", 42, 0)
	redisClient.Set(ctx, dayKey+":letters_generated", 5, 0)

	// Récupérer stats
	stats, err := service.GetRealtimeStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(2), stats["current_visitors"])
	assert.Equal(t, int64(2), stats["unique_today"])
	assert.Equal(t, int64(42), stats["total_events"])
	assert.Equal(t, int64(5), stats["letters_today"])
}

func TestAnalyticsService_CleanupOldEvents(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)

	// Créer visiteur
	visitor := &models.Visitor{
		SessionID:  uuid.New().String(),
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	require.NoError(t, db.Create(visitor).Error)

	// Créer événement ancien (95 jours)
	oldEvent := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: models.EventTypePageView,
		EventData: "{}",
	}
	require.NoError(t, db.Create(oldEvent).Error)
	// Update CreatedAt to make it old (95 jours)
	db.Model(oldEvent).Update("created_at", time.Now().AddDate(0, 0, -95))

	// Créer événement récent
	recentEvent := &models.AnalyticsEvent{
		VisitorID: visitor.ID,
		EventType: models.EventTypeButtonClick,
		EventData: "{}",
	}
	require.NoError(t, db.Create(recentEvent).Error)

	// Cleanup
	deleted, err := service.CleanupOldEvents(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Vérifier : ancien supprimé, récent conservé
	var count int64
	db.Model(&models.AnalyticsEvent{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var remaining models.AnalyticsEvent
	db.First(&remaining)
	assert.Equal(t, models.EventTypeButtonClick, remaining.EventType)
}

func TestAnalyticsService_MarkVisitorActive(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)
	ctx := context.Background()

	visitorID := uuid.New()

	// Marquer visiteur comme actif
	err := service.MarkVisitorActive(ctx, visitorID)
	require.NoError(t, err)

	// Vérifier dans Redis Set
	isMember, err := redisClient.SIsMember(ctx, "analytics:realtime:visitors", visitorID.String()).Result()
	require.NoError(t, err)
	assert.True(t, isMember)
}

func TestAnalyticsService_GetStats(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)
	ctx := context.Background()

	// Simuler données pour aujourd'hui
	today := time.Now().Format("2006-01-02")
	dayKey := "analytics:stats:day:" + today

	redisClient.Set(ctx, dayKey+":total_events", 100, 0)
	redisClient.Set(ctx, dayKey+":letters_generated", 10, 0)
	redisClient.PFAdd(ctx, "analytics:visitors:unique:day:"+today, uuid.New().String())

	// Récupérer stats
	stats, err := service.GetStats(ctx, "day")
	require.NoError(t, err)

	assert.Equal(t, "day", stats["period"])
	assert.Equal(t, int64(100), stats["total_events"])
	assert.Equal(t, int64(10), stats["letters_generated"])
	assert.Equal(t, int64(1), stats["unique_visitors"])
	assert.Equal(t, 10.0, stats["conversion_rate"]) // 10/1 = 10
}

func TestAnalyticsService_GetStats_InvalidPeriod(t *testing.T) {
	db, redisClient, cleanup := setupTestDB(t)
	defer cleanup()

	service := services.NewAnalyticsService(db, redisClient)

	_, err := service.GetStats(context.Background(), "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid period")
}
