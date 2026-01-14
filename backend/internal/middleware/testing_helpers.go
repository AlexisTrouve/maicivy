//go:build testing

package middleware

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// setupTestDB initialise une base de données SQLite en mémoire pour les tests
func setupTestDB(t *testing.T) (*gorm.DB, *redis.Client) {
	t.Helper()

	// SQLite en mémoire pour tests
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate les models (CORRECTED: utiliser les vrais noms de models)
	err = db.AutoMigrate(
		&models.Visitor{},
		&models.Experience{},
		&models.Skill{},
		&models.Project{},
		&models.GeneratedLetter{},
		&models.AnalyticsEvent{},
		&models.GitHubToken{},
		&models.GitHubProfile{},
		&models.GitHubRepository{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Redis client pour tests (utilise miniredis - Redis mock en mémoire)
	redisClient := setupMiniredis(t)

	return db, redisClient
}

// setupMiniredis initialise un miniredis (Redis mock en mémoire) pour les tests
func setupMiniredis(t *testing.T) *redis.Client {
	t.Helper()

	// Démarrer miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Cleanup automatique quand le test se termine
	t.Cleanup(func() {
		mr.Close()
	})

	// Créer client Redis pointant vers miniredis
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client
}
