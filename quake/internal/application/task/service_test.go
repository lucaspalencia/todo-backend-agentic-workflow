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
	created  *domtask.Task
	existing *domtask.Task
	err      error
}

func (s *stubRepo) Create(_ context.Context, t *domtask.Task) error {
	if s.err != nil {
		return s.err
	}
	s.created = t
	return nil
}
func (s *stubRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) {
	return s.existing, nil
}
func (s *stubRepo) FindAll(_ context.Context) ([]domtask.Task, error) { return nil, nil }
func (s *stubRepo) Save(_ context.Context, t *domtask.Task) error {
	s.existing = t
	return nil
}
func (s *stubRepo) Delete(_ context.Context, _ string) error { return nil }

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

func strPtr(s string) *string { return &s }

func existingTask() *domtask.Task {
	return &domtask.Task{
		ID:          "abc-123",
		Title:       "Original title",
		Description: "Original description",
		Status:      domtask.StatusPending,
	}
}

func TestUpdateTask_AllFields_ReturnsUpdatedTask(t *testing.T) {
	repo := &stubRepo{existing: existingTask()}
	svc := apptask.NewService(repo)
	updated, err := svc.UpdateTask(context.Background(), "abc-123", apptask.UpdateTaskCmd{
		Title:       strPtr("New title"),
		Description: strPtr("New description"),
		Status:      strPtr(domtask.StatusDone),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("expected title=%q, got %q", "New title", updated.Title)
	}
	if updated.Description != "New description" {
		t.Errorf("expected description=%q, got %q", "New description", updated.Description)
	}
	if updated.Status != domtask.StatusDone {
		t.Errorf("expected status=%q, got %q", domtask.StatusDone, updated.Status)
	}
	if updated.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestUpdateTask_SingleField_LeavesOthersUnchanged(t *testing.T) {
	repo := &stubRepo{existing: existingTask()}
	svc := apptask.NewService(repo)
	updated, err := svc.UpdateTask(context.Background(), "abc-123", apptask.UpdateTaskCmd{
		Status: strPtr(domtask.StatusInProgress),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Original title" {
		t.Errorf("expected title unchanged, got %q", updated.Title)
	}
	if updated.Description != "Original description" {
		t.Errorf("expected description unchanged, got %q", updated.Description)
	}
	if updated.Status != domtask.StatusInProgress {
		t.Errorf("expected status=%q, got %q", domtask.StatusInProgress, updated.Status)
	}
}

func TestUpdateTask_NotFound_ReturnsErrNotFound(t *testing.T) {
	repo := &stubRepo{existing: nil}
	svc := apptask.NewService(repo)
	_, err := svc.UpdateTask(context.Background(), "missing-id", apptask.UpdateTaskCmd{})
	if !errors.Is(err, domtask.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateTask_EmptyTitle_ReturnsValidationError(t *testing.T) {
	repo := &stubRepo{existing: existingTask()}
	svc := apptask.NewService(repo)
	_, err := svc.UpdateTask(context.Background(), "abc-123", apptask.UpdateTaskCmd{
		Title: strPtr("   "),
	})
	assertValidationField(t, err, "title")
}

func TestUpdateTask_InvalidStatus_ReturnsValidationError(t *testing.T) {
	repo := &stubRepo{existing: existingTask()}
	svc := apptask.NewService(repo)
	_, err := svc.UpdateTask(context.Background(), "abc-123", apptask.UpdateTaskCmd{
		Status: strPtr("INVALID"),
	})
	assertValidationField(t, err, "status")
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
