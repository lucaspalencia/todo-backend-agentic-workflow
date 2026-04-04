package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postComment(t *testing.T, srv *httptest.Server, taskID, body, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/tasks/"+taskID+"/comments", bytes.NewBufferString(body))
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

func listComments(t *testing.T, srv *httptest.Server, taskID, apiKey string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/tasks/"+taskID+"/comments", nil)
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

func TestIntegration_AddComment_Success(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	// Create parent task first.
	taskResp := postTask(t, srv, `{"title":"Task for comment"}`, integrationAPIKey)
	defer taskResp.Body.Close()
	if taskResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected task 201, got %d", taskResp.StatusCode)
	}
	var taskBody map[string]any
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody["id"].(string)

	resp := postComment(t, srv, taskID, `{"content":"First comment"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["id"] == "" {
		t.Error("expected non-empty id")
	}
	if body["task_id"] != taskID {
		t.Errorf("expected task_id=%q, got %v", taskID, body["task_id"])
	}
	if body["content"] != "First comment" {
		t.Errorf("expected content=%q, got %v", "First comment", body["content"])
	}
	if body["created_at"] == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestIntegration_ListComments_OrderedByCreatedAt(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	taskResp := postTask(t, srv, `{"title":"Task for listing comments"}`, integrationAPIKey)
	defer taskResp.Body.Close()
	var taskBody map[string]any
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody["id"].(string)

	for _, content := range []string{"alpha", "beta", "gamma"} {
		r := postComment(t, srv, taskID, `{"content":"`+content+`"}`, integrationAPIKey)
		r.Body.Close()
	}

	resp := listComments(t, srv, taskID, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var comments []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&comments)
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}
	// Verify ascending order by checking created_at strings are non-decreasing.
	for i := 1; i < len(comments); i++ {
		prev := comments[i-1]["created_at"].(string)
		curr := comments[i]["created_at"].(string)
		if prev > curr {
			t.Errorf("comments not in ascending order: %q > %q", prev, curr)
		}
	}
}

func TestIntegration_ListComments_EmptyList(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	taskResp := postTask(t, srv, `{"title":"Empty comment task"}`, integrationAPIKey)
	defer taskResp.Body.Close()
	var taskBody map[string]any
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody["id"].(string)

	resp := listComments(t, srv, taskID, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var comments []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&comments)
	if len(comments) != 0 {
		t.Errorf("expected empty list, got %d comments", len(comments))
	}
}

func TestIntegration_AddComment_TaskNotFound_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postComment(t, srv, "00000000-0000-0000-0000-000000000000", `{"content":"hi"}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestIntegration_ListComments_TaskNotFound_Returns404(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := listComments(t, srv, "00000000-0000-0000-0000-000000000000", integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestIntegration_AddComment_BlankContent_Returns422(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	taskResp := postTask(t, srv, `{"title":"Validation task"}`, integrationAPIKey)
	defer taskResp.Body.Close()
	var taskBody map[string]any
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody["id"].(string)

	resp := postComment(t, srv, taskID, `{"content":""}`, integrationAPIKey)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["errors"] == nil {
		t.Error("expected errors key in response")
	}
}

func TestIntegration_AddComment_NoAPIKey_Returns401(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postComment(t, srv, "any-id", `{"content":"hi"}`, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_CascadeDelete_RemovesComments(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	// Create task and comment.
	taskResp := postTask(t, srv, `{"title":"Cascade task"}`, integrationAPIKey)
	defer taskResp.Body.Close()
	var taskBody map[string]any
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody["id"].(string)

	commentResp := postComment(t, srv, taskID, `{"content":"Will be deleted"}`, integrationAPIKey)
	commentResp.Body.Close()

	// Delete the task.
	delResp := deleteTask(t, srv, taskID, integrationAPIKey)
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected task delete 204, got %d", delResp.StatusCode)
	}

	// GET /tasks/{id}/comments should now return 404 (task is gone).
	listResp := listComments(t, srv, taskID, integrationAPIKey)
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after cascade delete, got %d", listResp.StatusCode)
	}
}
