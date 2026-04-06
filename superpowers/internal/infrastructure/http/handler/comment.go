package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	appcomment "github.com/lucaspalencia/superpowers/internal/application/comment"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type commentRequest struct {
	Content string `json:"content"`
}

type commentResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// CommentHandler handles HTTP requests for comment operations.
type CommentHandler struct {
	svc *appcomment.Service
}

// NewCommentHandler constructs a CommentHandler with the given service.
func NewCommentHandler(svc *appcomment.Service) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// Create handles POST /tasks/{id}/comments. Returns 201 on success.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req commentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	comment, err := h.svc.AddComment(r.Context(), taskID, req.Content)
	if err != nil {
		var ve *appcomment.ValidationError
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

	writeJSON(w, http.StatusCreated, commentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
	})
}

// List handles GET /tasks/{id}/comments. Returns 200 with array ordered by created_at ASC.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	comments, err := h.svc.ListComments(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, commentResponse{
			ID:        c.ID,
			TaskID:    c.TaskID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
