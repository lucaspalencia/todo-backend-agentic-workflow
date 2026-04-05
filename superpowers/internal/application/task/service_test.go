package task_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	domain "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type mockRepo struct {
	returnErr error
}

func (m *mockRepo) Create(_ context.Context, t domain.Task) (domain.Task, error) {
	if m.returnErr != nil {
		return domain.Task{}, m.returnErr
	}
	return t, nil
}

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
