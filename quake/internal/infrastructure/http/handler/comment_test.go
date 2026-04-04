package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
	"github.com/lucaspalencia/todo-backend/internal/infrastructure/http/handler"
)

type stubCommentService struct {
	comment  *domcomment.Comment
	comments []domcomment.Comment
	err      error
}

func (s *stubCommentService) AddComment(_ context.Context, _, _ string) (*domcomment.Comment, error) {
	return s.comment, s.err
}
func (s *stubCommentService) ListComments(_ context.Context, _ string) ([]domcomment.Comment, error) {
	return s.comments, s.err
}

// newCommentRequest builds a request with the chi URL param "id" set.
func newCommentRequest(method, target, body, taskID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req
}

func TestCommentHandler_Add_ValidRequest_Returns201(t *testing.T) {
	stub := &stubCommentService{comment: &domcomment.Comment{
		ID: "c1", TaskID: "t1", Content: "hello", CreatedAt: time.Now(),
	}}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodPost, "/tasks/t1/comments", `{"content":"hello"}`, "t1")
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["id"] != "c1" {
		t.Errorf("expected id=c1, got %v", resp["id"])
	}
}

func TestCommentHandler_Add_MalformedJSON_Returns400(t *testing.T) {
	h := handler.NewCommentHandler(&stubCommentService{})
	req := newCommentRequest(http.MethodPost, "/tasks/t1/comments", `not-json`, "t1")
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCommentHandler_Add_ValidationError_Returns422(t *testing.T) {
	stub := &stubCommentService{err: appcomment.ValidationErrors{{Field: "content", Message: "content is required"}}}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodPost, "/tasks/t1/comments", `{"content":""}`, "t1")
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["errors"] == nil {
		t.Error("expected errors key in response")
	}
}

func TestCommentHandler_Add_TaskNotFound_Returns404(t *testing.T) {
	stub := &stubCommentService{err: domtask.ErrNotFound}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodPost, "/tasks/missing/comments", `{"content":"hi"}`, "missing")
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestCommentHandler_Add_InternalError_Returns500(t *testing.T) {
	stub := &stubCommentService{err: errors.New("db error")}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodPost, "/tasks/t1/comments", `{"content":"hi"}`, "t1")
	rec := httptest.NewRecorder()
	h.Add(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestCommentHandler_List_Success_Returns200(t *testing.T) {
	stub := &stubCommentService{comments: []domcomment.Comment{
		{ID: "c1", TaskID: "t1", Content: "first", CreatedAt: time.Now()},
		{ID: "c2", TaskID: "t1", Content: "second", CreatedAt: time.Now()},
	}}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodGet, "/tasks/t1/comments", "", "t1")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp []map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 comments, got %d", len(resp))
	}
}

func TestCommentHandler_List_TaskNotFound_Returns404(t *testing.T) {
	stub := &stubCommentService{err: domtask.ErrNotFound}
	h := handler.NewCommentHandler(stub)

	req := newCommentRequest(http.MethodGet, "/tasks/missing/comments", "", "missing")
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
