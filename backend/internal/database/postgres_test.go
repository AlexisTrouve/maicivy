package database

import (
	"testing"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Ce test nécessite une instance PostgreSQL de test
// Utiliser testcontainers en Phase 6 pour tests isolés
func TestConnectPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		DBHost:      "localhost",
		DBPort:      "5432",
		DBUser:      "test",
		DBPassword:  "test",
		DBName:      "maicivy_test",
		DBSSLMode:   "disable",
		Environment: "test",
	}

	db, err := ConnectPostgres(cfg)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	defer sqlDB.Close()

	// Test connection avec simple query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestRunAutoMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		DBHost:      "localhost",
		DBPort:      "5432",
		DBUser:      "test",
		DBPassword:  "test",
		DBName:      "maicivy_test",
		DBSSLMode:   "disable",
		Environment: "test",
	}

	db, err := ConnectPostgres(cfg)
	require.NoError(t, err)

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	err = RunAutoMigrations(db)
	assert.NoError(t, err)

	// Vérifier que les tables existent
	tables := []string{
		"experiences",
		"skills",
		"projects",
		"visitors",
		"generated_letters",
		"analytics_events",
	}

	for _, table := range tables {
		var exists bool
		err := db.Raw(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = ?)",
			table,
		).Scan(&exists).Error

		assert.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}
}

func TestCRUD_Experience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		DBHost:      "localhost",
		DBPort:      "5432",
		DBUser:      "test",
		DBPassword:  "test",
		DBName:      "maicivy_test",
		DBSSLMode:   "disable",
		Environment: "test",
	}

	db, err := ConnectPostgres(cfg)
	require.NoError(t, err)

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	err = RunAutoMigrations(db)
	require.NoError(t, err)

	// Create
	exp := models.Experience{
		Title:       "Test Engineer",
		Company:     "Test Corp",
		Description: "Testing database operations",
		StartDate:   time.Now(),
		Category:    "backend",
	}

	result := db.Create(&exp)
	assert.NoError(t, result.Error)
	assert.NotEqual(t, uuid.Nil, exp.ID)

	// Read
	var retrieved models.Experience
	err = db.First(&retrieved, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, exp.Title, retrieved.Title)

	// Update
	retrieved.Title = "Updated Title"
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated models.Experience
	err = db.First(&updated, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)

	// Delete (soft delete)
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	// Verify soft delete
	var deleted models.Experience
	err = db.First(&deleted, "id = ?", exp.ID).Error
	assert.Error(t, err) // Should not find (soft deleted)

	// Find with Unscoped (includes soft deleted)
	err = db.Unscoped().First(&deleted, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)

	// Cleanup (hard delete)
	db.Unscoped().Delete(&deleted)
}
