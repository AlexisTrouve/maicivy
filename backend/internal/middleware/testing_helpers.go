// +build testing

package middleware

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// setupTestDB initialise une base de données SQLite en mémoire pour les tests
func setupTestDB(t *testing.T) (*gorm.DB, *redis.Client) {
	// SQLite en mémoire pour tests
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate les models
	err = db.AutoMigrate(
		&models.Visitor{},
		&models.CVTheme{},
		&models.CVExperience{},
		&models.CVProject{},
		&models.CVSkill{},
		&models.GeneratedLetter{},
		&models.AnalyticsEvent{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Redis client pour tests (utilise miniredis ou Redis test)
	redisClient := setupTestRedis(t)

	return db, redisClient
}

// setupTestRedis initialise un client Redis pour les tests
func setupTestRedis(t *testing.T) *redis.Client {
	// Option 1: Utiliser miniredis (mock Redis en mémoire)
	// Pour les tests d'intégration réels, utiliser un vrai Redis
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Adapter selon env test
		Password: "",
		DB:       15, // Utiliser DB 15 pour tests
	})

	// Flush la DB de test avant chaque test
	client.FlushDB(nil)

	return client
}
