//go:build scripts
// +build scripts

package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"maicivy/internal/config"
	"maicivy/internal/database"
)

func main() {
	// Charger .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Charger config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Étape 1: Se connecter à postgres (base par défaut) pour créer la DB
	log.Println("🔧 Connecting to PostgreSQL...")
	defaultDSN := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	defaultDB, err := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v\n"+
			"💡 Assurez-vous que PostgreSQL est démarré et que l'utilisateur '%s' existe.\n"+
			"   Vous pouvez le créer avec: CREATE USER %s WITH PASSWORD '%s';",
			err, cfg.DBUser, cfg.DBUser, cfg.DBPassword)
	}

	// Étape 2: Créer la base de données si elle n'existe pas
	log.Printf("📦 Creating database '%s' if not exists...", cfg.DBName)
	result := defaultDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
	if result.Error != nil {
		// Si la DB existe déjà, c'est OK
		if result.Error.Error() != fmt.Sprintf("ERROR: database \"%s\" already exists (SQLSTATE 42P04)", cfg.DBName) {
			log.Printf("⚠️  Database creation warning: %v (may already exist)", result.Error)
		} else {
			log.Printf("✅ Database '%s' already exists", cfg.DBName)
		}
	} else {
		log.Printf("✅ Database '%s' created successfully", cfg.DBName)
	}

	// Fermer la connexion à postgres
	sqlDB, _ := defaultDB.DB()
	sqlDB.Close()

	// Étape 3: Se connecter à la nouvelle base de données
	log.Printf("🔗 Connecting to database '%s'...", cfg.DBName)
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database '%s': %v", cfg.DBName, err)
	}

	// Étape 4: Exécuter les migrations
	log.Println("🚀 Running auto-migrations...")
	if err := database.RunAutoMigrations(db); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	log.Println("✅ Database initialization completed successfully!")
	log.Println("")
	log.Println("📝 Next steps:")
	log.Println("   1. Run: go run scripts/seed.go (to seed sample data)")
	log.Println("   2. Run: go run cmd/main.go (to start the server)")
}
