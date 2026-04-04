package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func TestSmokeTest_HealthEndpoint(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping smoke test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	router := infrahttp.Register(pool)
	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
