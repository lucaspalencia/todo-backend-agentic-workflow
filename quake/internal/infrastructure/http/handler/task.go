package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// TaskCreator is the minimal interface the task handler needs from the application layer.
type TaskCreator interface {
	CreateTask(ctx context.Context, cmd apptask.CreateTaskCmd) (*domtask.Task, error)
}

// TaskHandler handles task-related HTTP endpoints.
type TaskHandler struct {
	svc TaskCreator
}

// NewTaskHandler constructs a TaskHandler with the given service.
func NewTaskHandler(svc TaskCreator) *TaskHandler {
	return &TaskHandler{svc: svc}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type taskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Create handles POST /tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	t, err := h.svc.CreateTask(r.Context(), apptask.CreateTaskCmd{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		var ve apptask.ValidationErrors
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": ve})
			return
		}
		if errors.Is(err, domtask.ErrDuplicateTitle) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": domtask.ErrDuplicateTitle.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
