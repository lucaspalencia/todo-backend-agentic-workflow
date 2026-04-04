package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucaspalencia/todo-backend/internal/infrastructure/http/middleware"
)

const testKey = "secret-key"

func TestAPIKeyAuth_MissingHeader_Returns401(t *testing.T) {
	handler := middleware.APIKeyAuth(testKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	assertErrorBody(t, rec, "unauthorized")
}

func TestAPIKeyAuth_WrongKey_Returns401(t *testing.T) {
	handler := middleware.APIKeyAuth(testKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	assertErrorBody(t, rec, "unauthorized")
}

func TestAPIKeyAuth_CorrectKey_CallsNext(t *testing.T) {
	nextCalled := false
	handler := middleware.APIKeyAuth(testKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("X-API-Key", testKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Error("expected next handler to be called")
	}
}

func assertErrorBody(t *testing.T, rec *httptest.ResponseRecorder, wantError string) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["error"] != wantError {
		t.Errorf("expected error=%q, got %q", wantError, body["error"])
	}
}
