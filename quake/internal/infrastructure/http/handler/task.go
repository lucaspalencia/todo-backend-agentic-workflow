package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// TaskCreator is the interface for the create task use case.
type TaskCreator interface {
	CreateTask(ctx context.Context, cmd apptask.CreateTaskCmd) (*domtask.Task, error)
}

// TaskUpdater is the interface for the update task use case.
type TaskUpdater interface {
	UpdateTask(ctx context.Context, id string, cmd apptask.UpdateTaskCmd) (*domtask.Task, error)
}

// TaskDeleter is the interface for the delete task use case.
type TaskDeleter interface {
	DeleteTask(ctx context.Context, id string) error
}

// TaskLister is the interface for the list tasks use case.
type TaskLister interface {
	ListTasks(ctx context.Context) ([]domtask.Task, error)
}

// TaskGetter is the interface for the get task by id use case.
type TaskGetter interface {
	GetTaskByID(ctx context.Context, id string) (*domtask.Task, error)
}

// TaskService combines all task use-case interfaces the handler depends on.
type TaskService interface {
	TaskCreator
	TaskUpdater
	TaskDeleter
	TaskLister
	TaskGetter
}

// TaskHandler handles task-related HTTP endpoints.
type TaskHandler struct {
	svc TaskService
}

// NewTaskHandler constructs a TaskHandler with the given service.
func NewTaskHandler(svc TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
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

// Update handles PATCH /tasks/{id}.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	t, err := h.svc.UpdateTask(r.Context(), id, apptask.UpdateTaskCmd{
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
		if errors.Is(err, domtask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": domtask.ErrNotFound.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Delete handles DELETE /tasks/{id}.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.svc.DeleteTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, domtask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": domtask.ErrNotFound.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListTasks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, taskResponse{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /tasks/{id}.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.svc.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domtask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": domtask.ErrNotFound.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, taskResponse{
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
