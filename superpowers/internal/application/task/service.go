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

// UpdateTaskInput holds the caller-supplied fields for updating a task.
// A nil pointer means the field was not provided and should not be changed.
type UpdateTaskInput struct {
	ID          string
	Title       *string
	Description *string
	Status      *string
}

// UpdateTask validates the provided fields, fetches the existing task, merges
// changes, refreshes updated_at, and persists the result.
func (s *Service) UpdateTask(ctx context.Context, in UpdateTaskInput) (domain.Task, error) {
	errs := make(map[string]string)

	if in.Title != nil {
		if strings.TrimSpace(*in.Title) == "" {
			errs["title"] = "title is required"
		} else if len(*in.Title) > 255 {
			errs["title"] = "title must be 255 characters or fewer"
		}
	}

	if in.Description != nil && len(*in.Description) > 2000 {
		errs["description"] = "description must be 2000 characters or fewer"
	}

	if in.Status != nil && !validStatuses[*in.Status] {
		errs["status"] = "invalid status: must be pending, in_progress, or done"
	}

	if len(errs) > 0 {
		return domain.Task{}, &ValidationError{Fields: errs}
	}

	task, err := s.repo.GetByID(ctx, in.ID)
	if err != nil {
		return domain.Task{}, err
	}

	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}
	if in.Status != nil {
		task.Status = *in.Status
	}
	task.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, task)
}

// DeleteTask soft-deletes a task. Returns domain.ErrNotFound if the task
// does not exist or has already been deleted.
func (s *Service) DeleteTask(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ListTasks returns all active (non-deleted) tasks ordered by newest first.
// Always returns a non-nil slice.
func (s *Service) ListTasks(ctx context.Context) ([]domain.Task, error) {
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return []domain.Task{}, nil
	}
	return tasks, nil
}

// GetTaskByID returns a single active task by ID. Returns domain.ErrNotFound
// if the task does not exist or has been soft-deleted.
func (s *Service) GetTaskByID(ctx context.Context, id string) (domain.Task, error) {
	return s.repo.GetByID(ctx, id)
}
