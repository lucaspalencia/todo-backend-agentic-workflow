package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
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

	commentRepo := postgres.NewCommentRepository(pool)
	commentSvc := appcomment.NewService(taskRepo, commentRepo)

	router := infrahttp.Register(pool, taskSvc, commentSvc, integrationAPIKey)
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

// --- helpers for delete / list / get ---

func deleteTask(t *testing.T, srv *httptest.Server, id string, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/tasks/"+id, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func getTask(t *testing.T, srv *httptest.Server, id string, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/tasks/"+id, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func listTasks(t *testing.T, srv *httptest.Server, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/tasks", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// --- DELETE /tasks/{id} ---

func TestIntegration_DeleteTask_Success_Returns204(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	resp := deleteTask(t, srv, id, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestIntegration_DeleteTask_VerifyTaskAbsent(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	del := deleteTask(t, srv, id, integrationAPIKey)
	del.Body.Close()
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", del.StatusCode)
	}

	get := getTask(t, srv, id, integrationAPIKey)
	defer get.Body.Close()
	if get.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", get.StatusCode)
	}
}

func TestIntegration_DeleteTask_NotFound_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := deleteTask(t, srv, "00000000-0000-0000-0000-000000000000", integrationAPIKey)
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

func TestIntegration_DeleteTask_MissingAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := deleteTask(t, srv, "any-id", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GET /tasks ---

func TestIntegration_ListTasks_Empty_ReturnsEmptyArray(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := listTasks(t, srv, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body []any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty array, got %d items", len(body))
	}
}

func TestIntegration_ListTasks_ExcludesDeletedTasks(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id1 := createTaskForUpdate(t, srv)
	postTask(t, srv, `{"title":"Task to keep"}`, integrationAPIKey).Body.Close()

	del := deleteTask(t, srv, id1, integrationAPIKey)
	del.Body.Close()

	resp := listTasks(t, srv, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected 1 task, got %d", len(body))
	}
	if body[0]["title"] != "Task to keep" {
		t.Errorf("expected title=%q, got %v", "Task to keep", body[0]["title"])
	}
}

func TestIntegration_ListTasks_MissingAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := listTasks(t, srv, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GET /tasks/{id} ---

func TestIntegration_GetTaskByID_Success_Returns200(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)

	resp := getTask(t, srv, id, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["id"] != id {
		t.Errorf("expected id=%q, got %v", id, body["id"])
	}
	if body["title"] != "Task to update" {
		t.Errorf("expected title=%q, got %v", "Task to update", body["title"])
	}
}

func TestIntegration_GetTaskByID_NotFound_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := getTask(t, srv, "00000000-0000-0000-0000-000000000000", integrationAPIKey)
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

func TestIntegration_GetTaskByID_DeletedTask_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	id := createTaskForUpdate(t, srv)
	deleteTask(t, srv, id, integrationAPIKey).Body.Close()

	resp := getTask(t, srv, id, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted task, got %d", resp.StatusCode)
	}
}

func TestIntegration_GetTaskByID_MissingAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := getTask(t, srv, "any-id", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
