package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// CommentService combines the comment use-case interfaces the handler depends on.
type CommentService interface {
	AddComment(ctx context.Context, taskID, content string) (*domcomment.Comment, error)
	ListComments(ctx context.Context, taskID string) ([]domcomment.Comment, error)
}

// CommentHandler handles comment-related HTTP endpoints.
type CommentHandler struct {
	svc CommentService
}

// NewCommentHandler constructs a CommentHandler with the given service.
func NewCommentHandler(svc CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

type addCommentRequest struct {
	Content string `json:"content"`
}

type commentResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func toCommentResponse(c *domcomment.Comment) commentResponse {
	return commentResponse{
		ID:        c.ID,
		TaskID:    c.TaskID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Add handles POST /tasks/{id}/comments.
func (h *CommentHandler) Add(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req addCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	c, err := h.svc.AddComment(r.Context(), taskID, req.Content)
	if err != nil {
		var ve appcomment.ValidationErrors
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

	writeJSON(w, http.StatusCreated, toCommentResponse(c))
}

// List handles GET /tasks/{id}/comments.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	comments, err := h.svc.ListComments(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, domtask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": domtask.ErrNotFound.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]commentResponse, 0, len(comments))
	for i := range comments {
		resp = append(resp, toCommentResponse(&comments[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}
