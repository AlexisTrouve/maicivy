package database

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"maicivy/internal/config"
)

func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	// Configuration GORM logger
	gormLogger := logger.Default.LogMode(logger.Silent)
	if cfg.Environment == "development" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configuration du pool de connexions
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Pool settings (selon IMPLEMENTATION_PLAN.md)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test de connexion
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Msg("PostgreSQL connected successfully")

	return db, nil
}

// AutoMigrate exécute les migrations automatiques GORM
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Import dynamique pour éviter les cycles d'import
	// Les models sont importés ici
	var models []interface{}

	// Note: Cette fonction sera étendue avec les vrais models
	// dans le fichier migrations.go séparé pour éviter les imports circulaires

	// Auto-migration
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Info().Msg("Database auto-migration completed")
	return nil
}
