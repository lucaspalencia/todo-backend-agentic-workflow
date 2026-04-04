package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/handler"
)

type mockPinger struct {
	err error
}

func (m mockPinger) Ping(_ context.Context) error { return m.err }

func TestHealthHandler_OK(t *testing.T) {
	h := handler.NewHealthHandler(mockPinger{err: nil})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("expected body to contain status:ok, got %s", body)
	}
	if !strings.Contains(body, `"db":"ok"`) {
		t.Errorf("expected body to contain db:ok, got %s", body)
	}
}

func TestHealthHandler_DBUnreachable(t *testing.T) {
	h := handler.NewHealthHandler(mockPinger{err: errors.New("connection refused")})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"error"`) {
		t.Errorf("expected body to contain status:error, got %s", body)
	}
	if !strings.Contains(body, `"db":"unreachable"`) {
		t.Errorf("expected body to contain db:unreachable, got %s", body)
	}
}
