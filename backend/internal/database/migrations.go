package database

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// tableExists checks if a table exists in the database
func tableExists(db *gorm.DB, tableName string) bool {
	var count int64
	db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?", tableName).Scan(&count)
	return count > 0
}

// RunAutoMigrations runs GORM automatic migrations with all models
// This adds new columns to existing tables without dropping data
func RunAutoMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	log.Info().Msg("Starting database auto-migration...")

	// Models that have JSONB with custom types - migrate only if table doesn't exist
	// This avoids GORM scanning issues with existing JSONB data
	jsonbModels := []struct {
		model     interface{}
		tableName string
	}{
		{&models.Experience{}, "experiences"},
		{&models.Project{}, "projects"},
	}

	for _, m := range jsonbModels {
		if !tableExists(db, m.tableName) {
			if err := db.AutoMigrate(m.model); err != nil {
				return fmt.Errorf("auto-migration failed for %s: %w", m.tableName, err)
			}
			log.Info().Str("table", m.tableName).Msg("Created table")
		} else {
			log.Info().Str("table", m.tableName).Msg("Table exists, skipping migration")
		}
	}

	// Other models - also skip if table exists to avoid GORM scanning issues
	// Note: GitHubToken is embedded in GitHubProfile (JSONB), not a separate table
	otherModels := []struct {
		model     interface{}
		tableName string
	}{
		{&models.Skill{}, "skills"},
		{&models.Visitor{}, "visitors"},
		{&models.GeneratedLetter{}, "generated_letters"},
		{&models.AnalyticsEvent{}, "analytics_events"},
		{&models.GitHubProfile{}, "github_profiles"},
		{&models.GitHubRepository{}, "github_repositories"},
	}

	for _, m := range otherModels {
		if !tableExists(db, m.tableName) {
			if err := db.AutoMigrate(m.model); err != nil {
				return fmt.Errorf("auto-migration failed for %s: %w", m.tableName, err)
			}
			log.Info().Str("table", m.tableName).Msg("Created table")
		} else {
			log.Info().Str("table", m.tableName).Msg("Table exists, skipping migration")
		}
	}

	log.Info().Msg("Database auto-migration completed successfully")
	return nil
}
