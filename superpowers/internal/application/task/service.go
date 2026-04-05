package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/lucaspalencia/superpowers/internal/domain/task"
)

// CreateTaskInput holds the caller-supplied fields for creating a task.
type CreateTaskInput struct {
	Title       string
	Description string
	Status      string
}

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

// Service implements the create task use case.
type Service struct {
	repo domain.Repository
}

// NewService constructs a Service with the given repository.
func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

var validStatuses = map[string]bool{
	"pending":     true,
	"in_progress": true,
	"done":        true,
}

// CreateTask validates input, generates a UUID, sets timestamps, and persists the task.
func (s *Service) CreateTask(ctx context.Context, in CreateTaskInput) (domain.Task, error) {
	errs := make(map[string]string)

	if strings.TrimSpace(in.Title) == "" {
		errs["title"] = "title is required"
	} else if len(in.Title) > 255 {
		errs["title"] = "title must be 255 characters or fewer"
	}

	if len(in.Description) > 2000 {
		errs["description"] = "description must be 2000 characters or fewer"
	}

	if in.Status != "" && !validStatuses[in.Status] {
		errs["status"] = "invalid status: must be pending, in_progress, or done"
	}

	if len(errs) > 0 {
		return domain.Task{}, &ValidationError{Fields: errs}
	}

	status := in.Status
	if status == "" {
		status = "pending"
	}

	now := time.Now().UTC()
	task := domain.Task{
		ID:          uuid.New().String(),
		Title:       in.Title,
		Description: in.Description,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, task)
}
