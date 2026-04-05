package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type taskRequest struct {
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

// TaskHandler handles HTTP requests for task operations.
type TaskHandler struct {
	svc *apptask.Service
}

// NewTaskHandler constructs a TaskHandler with the given service.
func NewTaskHandler(svc *apptask.Service) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create handles POST /tasks. Decodes JSON, delegates to service, returns 201 on success.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	task, err := h.svc.CreateTask(r.Context(), apptask.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		var ve *apptask.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": ve.Fields})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}

// Update handles PATCH /tasks/{id}. Applies partial updates and returns 200 with the full task.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	task, err := h.svc.UpdateTask(r.Context(), apptask.UpdateTaskInput{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		var ve *apptask.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": ve.Fields})
			return
		}
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}

// Delete handles DELETE /tasks/{id}. Returns 204 on success, 404 if not found.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.svc.DeleteTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /tasks. Returns 200 with a JSON array of all active tasks.
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
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /tasks/{id}. Returns 200 with the task, 404 if not found or deleted.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	task, err := h.svc.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}
