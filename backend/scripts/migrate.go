package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	// Charger .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run migrate.go [up|down|version|force <version>]")
	}

	command := os.Args[1]

	// Database URL
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	// Migrations path
	migrationsPath := "file://./migrations"

	// Create migrate instance
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("✅ Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("✅ Migrations rolled back successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d (dirty: %v)", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: go run migrate.go force <version>")
		}
		var version int
		fmt.Sscanf(os.Args[2], "%d", &version)
		if err := m.Force(version); err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		log.Printf("✅ Forced version to %d", version)

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
