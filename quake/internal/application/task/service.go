package task

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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

// UpdateTaskCmd holds the input for the UpdateTask use case.
// Nil fields mean "not provided — keep the existing value".
type UpdateTaskCmd struct {
	Title       *string
	Description *string
	Status      *string
}

// UpdateTask fetches the task by id, applies non-nil fields, refreshes updated_at, and persists.
func (s *Service) UpdateTask(ctx context.Context, id string, cmd UpdateTaskCmd) (*task.Task, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, task.ErrNotFound
	}

	var errs ValidationErrors

	if cmd.Title != nil {
		title := strings.TrimSpace(*cmd.Title)
		if title == "" {
			errs = append(errs, ValidationError{Field: "title", Message: "title is required"})
		} else if utf8.RuneCountInString(title) > 255 {
			errs = append(errs, ValidationError{Field: "title", Message: "title must not exceed 255 characters"})
		} else {
			t.Title = title
		}
	}

	if cmd.Description != nil {
		if utf8.RuneCountInString(*cmd.Description) > 2000 {
			errs = append(errs, ValidationError{Field: "description", Message: "description must not exceed 2000 characters"})
		} else {
			t.Description = *cmd.Description
		}
	}

	if cmd.Status != nil {
		if !allowedStatuses[*cmd.Status] {
			errs = append(errs, ValidationError{Field: "status", Message: "status must be one of: pending, in_progress, done"})
		} else {
			t.Status = *cmd.Status
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	t.UpdatedAt = time.Now().UTC()
	if err := s.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask soft-deletes the task with the given id.
// Returns ErrNotFound if the task does not exist or is already deleted.
func (s *Service) DeleteTask(ctx context.Context, id string) error {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return task.ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

// ListTasks returns all non-deleted tasks.
func (s *Service) ListTasks(ctx context.Context) ([]task.Task, error) {
	return s.repo.FindAll(ctx)
}

// GetTaskByID returns the task with the given id.
// Returns ErrNotFound if the task does not exist or is soft-deleted.
func (s *Service) GetTaskByID(ctx context.Context, id string) (*task.Task, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, task.ErrNotFound
	}
	return t, nil
}

// CreateTask validates the command, builds a Task entity, and persists it.
func (s *Service) CreateTask(ctx context.Context, cmd CreateTaskCmd) (*task.Task, error) {
	var errs ValidationErrors

	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		errs = append(errs, ValidationError{Field: "title", Message: "title is required"})
	} else if utf8.RuneCountInString(title) > 255 {
		errs = append(errs, ValidationError{Field: "title", Message: "title must not exceed 255 characters"})
	}

	if utf8.RuneCountInString(cmd.Description) > 2000 {
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
