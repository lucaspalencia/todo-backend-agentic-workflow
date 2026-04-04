package postgres_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/todo-backend/internal/infrastructure/persistence/postgres"
)

func TestConnect_InvalidDSN_ReturnsError(t *testing.T) {
	_, err := postgres.Connect("not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestConnect_Integration(t *testing.T) {
	dbUrl := os.Getenv("TEST_DATABASE_URL")
	if dbUrl == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	pool, err := postgres.Connect(dbUrl)
	if err != nil {
		t.Fatalf("expected successful connection, got: %v", err)
	}
	defer pool.Close()
}
