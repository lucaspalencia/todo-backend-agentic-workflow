package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	infrahttp "github.com/lucaspalencia/todo-backend/internal/infrastructure/http"
)

// okPinger satisfies handler.Pinger without a real DB.
type okPinger struct{}

func (p *okPinger) Ping(_ context.Context) error { return nil }

func TestHealthEndpoint_Smoke(t *testing.T) {
	router := infrahttp.Register(&okPinger{})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
