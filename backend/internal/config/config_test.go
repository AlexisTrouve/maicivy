package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set env vars for test
	os.Setenv("DB_PASSWORD", "test123")
	defer os.Unsetenv("DB_PASSWORD")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DBPassword != "test123" {
		t.Errorf("Expected DB_PASSWORD=test123, got %s", cfg.DBPassword)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("Expected default ServerPort=8080, got %s", cfg.ServerPort)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	if result != "value" {
		t.Errorf("Expected 'value', got '%s'", result)
	}

	result = getEnv("NON_EXISTENT", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}
