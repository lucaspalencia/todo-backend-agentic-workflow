package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	infrahttp "github.com/lucaspalencia/todo-backend/internal/infrastructure/http"
	"github.com/lucaspalencia/todo-backend/internal/infrastructure/persistence/postgres"
)

const integrationAPIKey = "integration-test-key"

// newIntegrationServer connects to TEST_DATABASE_URL and returns a test server.
// Skips if TEST_DATABASE_URL is not set.
func newIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}
	t.Cleanup(pool.Close)

	// Truncate tasks before and after each test for isolation.
	truncate := func() { _, _ = pool.Exec(t.Context(), "TRUNCATE TABLE tasks") }
	truncate()
	t.Cleanup(truncate)

	taskRepo := postgres.NewTaskRepository(pool)
	taskSvc := apptask.NewService(taskRepo)

	router := infrahttp.Register(pool, taskSvc, integrationAPIKey)
	return httptest.NewServer(router)
}

func postTask(t *testing.T, srv *httptest.Server, body string, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tasks", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestIntegration_CreateTask_Success(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postTask(t, srv, `{"title":"Integration task","description":"hello"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["id"] == "" {
		t.Error("expected non-empty id")
	}
	if body["title"] != "Integration task" {
		t.Errorf("expected title=%q, got %v", "Integration task", body["title"])
	}
	if body["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", body["status"])
	}
}

func TestIntegration_CreateTask_MissingTitle_Returns422(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postTask(t, srv, `{"description":"no title here"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["errors"] == nil {
		t.Error("expected errors field in response")
	}
}

func TestIntegration_CreateTask_InvalidStatus_Returns422(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postTask(t, srv, `{"title":"Valid title","status":"INVALID"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestIntegration_CreateTask_DuplicateTitle_Returns409(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	body := `{"title":"Unique task"}`
	resp1 := postTask(t, srv, body, integrationAPIKey)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", resp1.StatusCode)
	}

	resp2 := postTask(t, srv, body, integrationAPIKey)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate create: expected 409, got %d", resp2.StatusCode)
	}
}

func TestIntegration_CreateTask_MissingAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postTask(t, srv, `{"title":"Unauthorized"}`, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func patchTask(t *testing.T, srv *httptest.Server, id string, body string, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/tasks/"+id, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func createTaskForUpdate(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	resp := postTask(t, srv, `{"title":"Task to update","description":"original","status":"pending"}`, integrationAPIKey)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("setup: failed to decode: %v", err)
	}
	return body["id"].(string)
}

func TestIntegration_UpdateTask_FullUpdate_Returns200(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	resp := patchTask(t, srv, id, `{"title":"Updated title","description":"updated desc","status":"done"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["title"] != "Updated title" {
		t.Errorf("expected title=%q, got %v", "Updated title", body["title"])
	}
	if body["description"] != "updated desc" {
		t.Errorf("expected description=%q, got %v", "updated desc", body["description"])
	}
	if body["status"] != "done" {
		t.Errorf("expected status=done, got %v", body["status"])
	}
	if body["updated_at"] == "" {
		t.Error("expected non-empty updated_at")
	}
}

func TestIntegration_UpdateTask_SingleField_Returns200(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	resp := patchTask(t, srv, id, `{"status":"in_progress"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["title"] != "Task to update" {
		t.Errorf("expected title unchanged, got %v", body["title"])
	}
	if body["description"] != "original" {
		t.Errorf("expected description unchanged, got %v", body["description"])
	}
	if body["status"] != "in_progress" {
		t.Errorf("expected status=in_progress, got %v", body["status"])
	}
}

func TestIntegration_UpdateTask_NotFound_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := patchTask(t, srv, "00000000-0000-0000-0000-000000000000", `{"status":"done"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == nil {
		t.Error("expected error field in response")
	}
}

func TestIntegration_UpdateTask_InvalidStatus_Returns422(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	resp := patchTask(t, srv, id, `{"status":"INVALID"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["errors"] == nil {
		t.Error("expected errors field in response")
	}
}

func TestIntegration_UpdateTask_MissingAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := patchTask(t, srv, "any-id", `{"status":"done"}`, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
