package comment

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// Service orchestrates comment use cases.
type Service struct {
	taskRepo    domtask.Repository
	commentRepo domcomment.Repository
}

// NewService constructs a Service with the given repository implementations.
func NewService(taskRepo domtask.Repository, commentRepo domcomment.Repository) *Service {
	return &Service{taskRepo: taskRepo, commentRepo: commentRepo}
}

// ValidationError represents a field-level validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"error"`
}

// ValidationErrors is a slice of field validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, len(e))
	for i, v := range e {
		msgs[i] = fmt.Sprintf("%s: %s", v.Field, v.Message)
	}
	return strings.Join(msgs, "; ")
}

// AddComment validates input, verifies the parent task exists, and persists a new comment.
func (s *Service) AddComment(ctx context.Context, taskID, content string) (*domcomment.Comment, error) {
	var errs ValidationErrors

	content = strings.TrimSpace(content)
	if content == "" {
		errs = append(errs, ValidationError{Field: "content", Message: "content is required"})
	} else if utf8.RuneCountInString(content) > 2000 {
		errs = append(errs, ValidationError{Field: "content", Message: "content must not exceed 2000 characters"})
	}

	if len(errs) > 0 {
		return nil, errs
	}

	t, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domtask.ErrNotFound
	}

	c := &domcomment.Comment{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.commentRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListComments returns all comments for the given task, ordered by created_at ASC.
// Returns task.ErrNotFound if the task does not exist.
func (s *Service) ListComments(ctx context.Context, taskID string) ([]domcomment.Comment, error) {
	t, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domtask.ErrNotFound
	}

	return s.commentRepo.FindByTaskID(ctx, taskID)
}
