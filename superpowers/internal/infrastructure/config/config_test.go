package config_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/config"
)

func TestLoad_ErrorOnEmptyDBUrl(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is empty, got nil")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("PORT")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port=8080, got %s", cfg.Port)
	}
}

func TestLoad_DefaultEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("ENV")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env=development, got %s", cfg.Env)
	}
}
