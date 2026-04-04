package postgres_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := postgres.Connect("not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestConnect_Success(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("expected successful connection, got: %v", err)
	}
	defer pool.Close()
}
