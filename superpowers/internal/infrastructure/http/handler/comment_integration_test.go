package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAddComment_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task for comments")

	body := `{"content":"This is a comment"}`
	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["id"] == "" || result["id"] == nil {
		t.Error("expected non-empty id in response")
	}
	if result["task_id"] != taskID {
		t.Errorf("expected task_id %q, got %v", taskID, result["task_id"])
	}
	if result["content"] != "This is a comment" {
		t.Errorf("expected content 'This is a comment', got %v", result["content"])
	}
	if result["created_at"] == "" || result["created_at"] == nil {
		t.Error("expected non-empty created_at in response")
	}
}

func TestAddComment_Integration_TaskNotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000/comments",
		bytes.NewBufferString(`{"content":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/{id}/comments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != "task not found" {
		t.Errorf("expected error 'task not found', got %v", result["error"])
	}
}

func TestAddComment_Integration_ValidationError(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task for validation test")

	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(`{"content":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errs, ok := result["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'errors' object in response body, got %v", result)
	}
	if _, ok := errs["content"]; !ok {
		t.Error("expected errors[\"content\"] to be set")
	}
}

func TestListComments_Integration_Empty(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task with no comments")

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result []any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestListComments_Integration_OrderedAsc(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task with ordered comments")

	addComment := func(content string) {
		req, _ := http.NewRequest(http.MethodPost,
			integServer.URL+"/tasks/"+taskID+"/comments",
			bytes.NewBufferString(`{"content":"`+content+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST comment: %v", err)
		}
		resp.Body.Close()
	}

	addComment("First comment")
	addComment("Second comment")

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(result))
	}
	if result[0]["content"] != "First comment" {
		t.Errorf("expected first comment 'First comment', got %v", result[0]["content"])
	}
	if result[1]["content"] != "Second comment" {
		t.Errorf("expected second comment 'Second comment', got %v", result[1]["content"])
	}
}

func TestListComments_Integration_TaskNotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/{id}/comments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != "task not found" {
		t.Errorf("expected error 'task not found', got %v", result["error"])
	}
}

func TestDeleteTask_Integration_CascadeDeletesComments(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task to cascade-delete")

	// Add a comment to the task.
	addReq, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(`{"content":"Should be deleted"}`))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatalf("POST comment: %v", err)
	}
	addResp.Body.Close()
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding comment, got %d", addResp.StatusCode)
	}

	// Delete the task (soft-delete + application-layer comment cleanup).
	delReq, _ := http.NewRequest(http.MethodDelete,
		integServer.URL+"/tasks/"+taskID, nil)
	delReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /tasks/%s: %v", taskID, err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting task, got %d", delResp.StatusCode)
	}

	// API-level: GET /tasks/{id}/comments returns 404 (task is gone).
	getReq, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	getReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after task delete, got %d", getResp.StatusCode)
	}

	// DB-level: confirm comment rows are physically gone.
	var count int
	row := integPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM comments WHERE task_id = $1", taskID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 comment rows after task delete, got %d", count)
	}
}

// createTestTask creates a task via POST /tasks and returns its ID.
func createTestTask(t *testing.T, title string) string {
	t.Helper()
	body := `{"title":"` + title + `"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode create task response: %v", err)
	}
	id, _ := result["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty task ID from POST /tasks")
	}
	return id
}
