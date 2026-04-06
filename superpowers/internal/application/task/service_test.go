package task_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	domain "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type mockRepo struct {
	returnErr  error
	getByIDErr error
	updateErr  error
	deleteErr  error
	listTasks  []domain.Task
	listErr    error
	storedTask domain.Task
}

func (m *mockRepo) Create(_ context.Context, t domain.Task) (domain.Task, error) {
	if m.returnErr != nil {
		return domain.Task{}, m.returnErr
	}
	return t, nil
}

func (m *mockRepo) GetByID(_ context.Context, _ string) (domain.Task, error) {
	if m.getByIDErr != nil {
		return domain.Task{}, m.getByIDErr
	}
	return m.storedTask, nil
}

func (m *mockRepo) Update(_ context.Context, t domain.Task) (domain.Task, error) {
	if m.updateErr != nil {
		return domain.Task{}, m.updateErr
	}
	return t, nil
}

func (m *mockRepo) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockRepo) List(_ context.Context) ([]domain.Task, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listTasks != nil {
		return m.listTasks, nil
	}
	return []domain.Task{}, nil
}

// --- CreateTask tests (unchanged) ---

func TestCreateTask_Success(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	got, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title: "Buy groceries",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == "" {
		t.Error("expected UUID to be set, got empty string")
	}
	if got.Title != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %q", got.Title)
	}
	if got.Status != "pending" {
		t.Errorf("expected default status 'pending', got %q", got.Status)
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestCreateTask_DefaultsStatusToPending(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	got, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title:  "My task",
		Status: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", got.Status)
	}
}

func TestCreateTask_ExplicitStatus(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	got, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title:  "My task",
		Status: "in_progress",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", got.Status)
	}
}

func TestCreateTask_EmptyTitle(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{})
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["title"]; !ok {
		t.Error("expected ve.Fields[\"title\"] to be set")
	}
}

func TestCreateTask_TitleTooLong(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title: strings.Repeat("a", 256),
	})
	if err == nil {
		t.Fatal("expected error for title > 255 chars, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["title"]; !ok {
		t.Error("expected ve.Fields[\"title\"] to be set")
	}
}

func TestCreateTask_DescriptionTooLong(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title:       "Valid title",
		Description: strings.Repeat("x", 2001),
	})
	if err == nil {
		t.Fatal("expected error for description > 2000 chars, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["description"]; !ok {
		t.Error("expected ve.Fields[\"description\"] to be set")
	}
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	_, err := svc.CreateTask(context.Background(), apptask.CreateTaskInput{
		Title:  "Valid title",
		Status: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["status"]; !ok {
		t.Error("expected ve.Fields[\"status\"] to be set")
	}
}

// --- UpdateTask tests ---

func storedTask() domain.Task {
	return domain.Task{
		ID:          "test-id",
		Title:       "Original title",
		Description: "Original description",
		Status:      "pending",
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func strPtr(s string) *string { return &s }

func TestUpdateTask_FullUpdate(t *testing.T) {
	svc := apptask.NewService(&mockRepo{storedTask: storedTask()})

	got, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:          "test-id",
		Title:       strPtr("New title"),
		Description: strPtr("New description"),
		Status:      strPtr("done"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "New title" {
		t.Errorf("expected title 'New title', got %q", got.Title)
	}
	if got.Description != "New description" {
		t.Errorf("expected description 'New description', got %q", got.Description)
	}
	if got.Status != "done" {
		t.Errorf("expected status 'done', got %q", got.Status)
	}
	if !got.UpdatedAt.After(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected UpdatedAt to be refreshed past the stored value")
	}
}

func TestUpdateTask_PartialUpdate_StatusOnly(t *testing.T) {
	stored := storedTask()
	svc := apptask.NewService(&mockRepo{storedTask: stored})

	got, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:     "test-id",
		Status: strPtr("done"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != stored.Title {
		t.Errorf("expected title unchanged %q, got %q", stored.Title, got.Title)
	}
	if got.Description != stored.Description {
		t.Errorf("expected description unchanged %q, got %q", stored.Description, got.Description)
	}
	if got.Status != "done" {
		t.Errorf("expected status 'done', got %q", got.Status)
	}
}

func TestUpdateTask_EmptyTitle(t *testing.T) {
	svc := apptask.NewService(&mockRepo{storedTask: storedTask()})

	_, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:    "test-id",
		Title: strPtr(""),
	})
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["title"]; !ok {
		t.Error("expected ve.Fields[\"title\"] to be set")
	}
}

func TestUpdateTask_TitleTooLong(t *testing.T) {
	svc := apptask.NewService(&mockRepo{storedTask: storedTask()})

	_, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:    "test-id",
		Title: strPtr(strings.Repeat("a", 256)),
	})
	if err == nil {
		t.Fatal("expected error for title > 255 chars, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["title"]; !ok {
		t.Error("expected ve.Fields[\"title\"] to be set")
	}
}

func TestUpdateTask_DescriptionTooLong(t *testing.T) {
	svc := apptask.NewService(&mockRepo{storedTask: storedTask()})

	_, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:          "test-id",
		Description: strPtr(strings.Repeat("x", 2001)),
	})
	if err == nil {
		t.Fatal("expected error for description > 2000 chars, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["description"]; !ok {
		t.Error("expected ve.Fields[\"description\"] to be set")
	}
}

func TestUpdateTask_InvalidStatus(t *testing.T) {
	svc := apptask.NewService(&mockRepo{storedTask: storedTask()})

	_, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:     "test-id",
		Status: strPtr("invalid"),
	})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	var ve *apptask.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *apptask.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["status"]; !ok {
		t.Error("expected ve.Fields[\"status\"] to be set")
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	svc := apptask.NewService(&mockRepo{getByIDErr: domain.ErrNotFound})

	_, err := svc.UpdateTask(context.Background(), apptask.UpdateTaskInput{
		ID:     "missing-id",
		Status: strPtr("done"),
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got %v", err)
	}
}

// --- DeleteTask tests ---

func TestDeleteTask_Success(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	err := svc.DeleteTask(context.Background(), "some-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	svc := apptask.NewService(&mockRepo{deleteErr: domain.ErrNotFound})

	err := svc.DeleteTask(context.Background(), "missing-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got %v", err)
	}
}

// --- ListTasks tests ---

func TestListTasks_Empty(t *testing.T) {
	svc := apptask.NewService(&mockRepo{})

	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tasks == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(tasks) != 0 {
		t.Errorf("expected empty slice, got %d tasks", len(tasks))
	}
}

func TestListTasks_ReturnsTasks(t *testing.T) {
	stored := []domain.Task{
		{ID: "a", Title: "First"},
		{ID: "b", Title: "Second"},
	}
	svc := apptask.NewService(&mockRepo{listTasks: stored})

	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "a" || tasks[1].ID != "b" {
		t.Errorf("unexpected task order: %v", tasks)
	}
}

// --- GetTaskByID tests ---

func TestGetTaskByID_Success(t *testing.T) {
	stored := storedTask()
	svc := apptask.NewService(&mockRepo{storedTask: stored})

	got, err := svc.GetTaskByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != stored.ID {
		t.Errorf("expected ID %q, got %q", stored.ID, got.ID)
	}
	if got.Title != stored.Title {
		t.Errorf("expected title %q, got %q", stored.Title, got.Title)
	}
}

func TestGetTaskByID_NotFound(t *testing.T) {
	svc := apptask.NewService(&mockRepo{getByIDErr: domain.ErrNotFound})

	_, err := svc.GetTaskByID(context.Background(), "missing-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got %v", err)
	}
}
