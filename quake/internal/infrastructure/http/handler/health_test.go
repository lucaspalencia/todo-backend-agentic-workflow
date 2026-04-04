package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucaspalencia/todo-backend/internal/infrastructure/http/handler"
)

type stubPinger struct{ err error }

func (s *stubPinger) Ping(_ context.Context) error { return s.err }

func TestHealthHandler_DBUnreachable_Returns503(t *testing.T) {
	h := handler.NewHealthHandler(&stubPinger{err: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Check(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("expected status=error, got %s", body["status"])
	}
	if body["db"] != "unreachable" {
		t.Errorf("expected db=unreachable, got %s", body["db"])
	}
}

func TestHealthHandler_DBReachable_Returns200(t *testing.T) {
	h := handler.NewHealthHandler(&stubPinger{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Check(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", body["status"])
	}
	if body["db"] != "ok" {
		t.Errorf("expected db=ok, got %s", body["db"])
	}
}
