package database

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// RunAutoMigrations exécute les migrations automatiques GORM avec tous les models
func RunAutoMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Liste de tous les models à migrer
	modelsList := []interface{}{
		&models.Experience{},
		&models.Skill{},
		&models.Project{},
		&models.Visitor{},
		&models.GeneratedLetter{},
		&models.AnalyticsEvent{},
		&models.GitHubProfile{},
		&models.GitHubToken{},
		&models.GitHubRepository{},
	}

	log.Info().Msg("Starting database auto-migration...")

	// Auto-migration
	if err := db.AutoMigrate(modelsList...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Info().Msg("✅ Database auto-migration completed successfully")
	return nil
}
