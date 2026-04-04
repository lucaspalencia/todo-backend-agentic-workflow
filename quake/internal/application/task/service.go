package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// Service orchestrates task use cases. It depends only on the domain repository
// interface — no infrastructure imports.
type Service struct {
	repo task.Repository
}

// NewService constructs a Service with the given repository implementation.
func NewService(repo task.Repository) *Service {
	return &Service{repo: repo}
}

// CreateTaskCmd holds the input for the CreateTask use case.
type CreateTaskCmd struct {
	Title       string
	Description string
	Status      string
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

var allowedStatuses = map[string]bool{
	task.StatusPending:    true,
	task.StatusInProgress: true,
	task.StatusDone:       true,
}

// CreateTask validates the command, builds a Task entity, and persists it.
func (s *Service) CreateTask(ctx context.Context, cmd CreateTaskCmd) (*task.Task, error) {
	var errs ValidationErrors

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		errs = append(errs, ValidationError{Field: "title", Message: "title is required"})
	} else if len(title) > 255 {
		errs = append(errs, ValidationError{Field: "title", Message: "title must not exceed 255 characters"})
	}

	if len(cmd.Description) > 2000 {
		errs = append(errs, ValidationError{Field: "description", Message: "description must not exceed 2000 characters"})
	}

	status := cmd.Status
	if status == "" {
		status = task.StatusPending
	} else if !allowedStatuses[status] {
		errs = append(errs, ValidationError{Field: "status", Message: "status must be one of: pending, in_progress, done"})
	}

	if len(errs) > 0 {
		return nil, errs
	}

	now := time.Now().UTC()
	t := &task.Task{
		ID:          uuid.New().String(),
		Title:       title,
		Description: cmd.Description,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
