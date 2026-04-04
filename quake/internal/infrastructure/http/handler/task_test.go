package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
	"github.com/lucaspalencia/todo-backend/internal/infrastructure/http/handler"
)

type stubTaskCreator struct {
	task *domtask.Task
	err  error
}

func (s *stubTaskCreator) CreateTask(_ context.Context, _ apptask.CreateTaskCmd) (*domtask.Task, error) {
	return s.task, s.err
}

func TestTaskHandler_Create_ValidRequest_Returns201(t *testing.T) {
	stub := &stubTaskCreator{task: &domtask.Task{
		ID: "abc", Title: "Buy milk", Status: domtask.StatusPending,
	}}
	h := handler.NewTaskHandler(stub)

	body := bytes.NewBufferString(`{"title":"Buy milk"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["id"] != "abc" {
		t.Errorf("expected id=abc, got %v", resp["id"])
	}
}

func TestTaskHandler_Create_ValidationError_Returns422(t *testing.T) {
	stub := &stubTaskCreator{err: apptask.ValidationErrors{{Field: "title", Message: "title is required"}}}
	h := handler.NewTaskHandler(stub)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_DuplicateTitle_Returns409(t *testing.T) {
	stub := &stubTaskCreator{err: domtask.ErrDuplicateTitle}
	h := handler.NewTaskHandler(stub)

	body := bytes.NewBufferString(`{"title":"Dup"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_MalformedJSON_Returns400(t *testing.T) {
	stub := &stubTaskCreator{}
	h := handler.NewTaskHandler(stub)

	body := bytes.NewBufferString(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_InternalError_Returns500(t *testing.T) {
	stub := &stubTaskCreator{err: errors.New("db exploded")}
	h := handler.NewTaskHandler(stub)

	body := bytes.NewBufferString(`{"title":"Valid"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
