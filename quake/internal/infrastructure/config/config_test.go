package config_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/todo-backend/internal/infrastructure/config"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	original, exists := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if exists {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	})
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, exists := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if exists {
			os.Setenv(key, original)
		}
	})
}

func TestLoad_MissingDBUrl_ReturnsError(t *testing.T) {
	unsetEnv(t, "DATABASE_URL")
	unsetEnv(t, "PORT")
	unsetEnv(t, "ENV")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	setEnv(t, "DATABASE_URL", "postgres://user:pass@localhost/db")
	unsetEnv(t, "PORT")
	unsetEnv(t, "ENV")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
}
