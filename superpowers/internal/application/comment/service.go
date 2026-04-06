package comment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domaincomment "github.com/lucaspalencia/superpowers/internal/domain/comment"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

// ValidationError is returned when one or more input fields are invalid.
// Fields maps each invalid field name to a human-readable error message.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return "validation error: " + strings.Join(parts, "; ")
}

// Service implements the comment use cases.
type Service struct {
	repo     domaincomment.Repository
	taskRepo domaintask.Repository
}

// NewService constructs a Service with the given repositories.
func NewService(repo domaincomment.Repository, taskRepo domaintask.Repository) *Service {
	return &Service{repo: repo, taskRepo: taskRepo}
}

// AddComment validates input, verifies the parent task exists, and persists the comment.
func (s *Service) AddComment(ctx context.Context, taskID, content string) (domaincomment.Comment, error) {
	errs := make(map[string]string)

	if strings.TrimSpace(content) == "" {
		errs["content"] = "content is required"
	} else if len(content) > 2000 {
		errs["content"] = "content must be 2000 characters or fewer"
	}

	if len(errs) > 0 {
		return domaincomment.Comment{}, &ValidationError{Fields: errs}
	}

	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return domaincomment.Comment{}, err
	}

	comment := domaincomment.Comment{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}

	return s.repo.Create(ctx, comment)
}

// ListComments verifies the parent task exists, then returns all comments ordered
// by created_at ASC. Always returns a non-nil slice.
func (s *Service) ListComments(ctx context.Context, taskID string) ([]domaincomment.Comment, error) {
	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return nil, err
	}

	comments, err := s.repo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		return []domaincomment.Comment{}, nil
	}
	return comments, nil
}
