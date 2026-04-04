package task_test

import (
	"context"
	"errors"
	"testing"

	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// stubRepo is a minimal in-memory task.Repository for testing.
type stubRepo struct {
	created *domtask.Task
	err     error
}

func (s *stubRepo) Create(_ context.Context, t *domtask.Task) error {
	if s.err != nil {
		return s.err
	}
	s.created = t
	return nil
}
func (s *stubRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) { return nil, nil }
func (s *stubRepo) FindAll(_ context.Context) ([]domtask.Task, error)           { return nil, nil }
func (s *stubRepo) Save(_ context.Context, _ *domtask.Task) error               { return nil }
func (s *stubRepo) Delete(_ context.Context, _ string) error                    { return nil }

func TestCreateTask_ValidInput_ReturnsTask(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	task, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{
		Title:       "Buy milk",
		Description: "From the corner shop",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Status != domtask.StatusPending {
		t.Errorf("expected status=pending, got %q", task.Status)
	}
	if task.Title != "Buy milk" {
		t.Errorf("expected title=%q, got %q", "Buy milk", task.Title)
	}
}

func TestCreateTask_EmptyTitle_ReturnsValidationError(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{Title: ""})
	assertValidationField(t, err, "title")
}

func TestCreateTask_WhitespaceTitle_ReturnsValidationError(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{Title: "   "})
	assertValidationField(t, err, "title")
}

func TestCreateTask_TitleTooLong_ReturnsValidationError(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{Title: string(make([]byte, 256))})
	assertValidationField(t, err, "title")
}

func TestCreateTask_DescriptionTooLong_ReturnsValidationError(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{
		Title:       "Valid",
		Description: string(make([]byte, 2001)),
	})
	assertValidationField(t, err, "description")
}

func TestCreateTask_InvalidStatus_ReturnsValidationError(t *testing.T) {
	svc := apptask.NewService(&stubRepo{})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{
		Title:  "Valid",
		Status: "Pending",
	})
	assertValidationField(t, err, "status")
}

func TestCreateTask_DuplicateTitle_ReturnsErrDuplicateTitle(t *testing.T) {
	svc := apptask.NewService(&stubRepo{err: domtask.ErrDuplicateTitle})
	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskCmd{Title: "Dup"})
	if !errors.Is(err, domtask.ErrDuplicateTitle) {
		t.Errorf("expected ErrDuplicateTitle, got %v", err)
	}
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ve apptask.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	for _, e := range ve {
		if e.Field == field {
			return
		}
	}
	t.Errorf("expected validation error for field %q, got %v", field, ve)
}
