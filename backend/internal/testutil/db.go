//go:build testing

package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// SetupTestDB crée une DB SQLite en mémoire + miniredis mock
// Retourne: db, redisClient, cleanup function
func SetupTestDB(t *testing.T) (*gorm.DB, *redis.Client, func()) {
	t.Helper()

	// SQLite en mémoire
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}

	// Migrations
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
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	// Miniredis (Redis mock en mémoire)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to setup miniredis: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Cleanup function
	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		redisClient.Close()
		mr.Close()
	}

	return db, redisClient, cleanup
}

// SetupTestDBOnly crée uniquement la DB SQLite (sans Redis)
func SetupTestDBOnly(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}

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
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return db, cleanup
}

// SetupMiniredis crée et retourne un client miniredis pour les tests
func SetupMiniredis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}
