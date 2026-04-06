package comment_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appcomment "github.com/lucaspalencia/superpowers/internal/application/comment"
	domaincomment "github.com/lucaspalencia/superpowers/internal/domain/comment"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

// mockCommentRepo implements domain/comment.Repository.
type mockCommentRepo struct {
	createErr      error
	listErr        error
	storedComments []domaincomment.Comment
}

func (m *mockCommentRepo) Create(_ context.Context, c domaincomment.Comment) (domaincomment.Comment, error) {
	if m.createErr != nil {
		return domaincomment.Comment{}, m.createErr
	}
	return c, nil
}

func (m *mockCommentRepo) ListByTaskID(_ context.Context, _ string) ([]domaincomment.Comment, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.storedComments != nil {
		return m.storedComments, nil
	}
	return []domaincomment.Comment{}, nil
}

func (m *mockCommentRepo) DeleteByTaskID(_ context.Context, _ string) error { return nil }

// mockTaskRepo implements domain/task.Repository.
type mockTaskRepo struct {
	getByIDErr error
	storedTask domaintask.Task
}

func (m *mockTaskRepo) Create(_ context.Context, t domaintask.Task) (domaintask.Task, error) {
	return t, nil
}
func (m *mockTaskRepo) GetByID(_ context.Context, _ string) (domaintask.Task, error) {
	if m.getByIDErr != nil {
		return domaintask.Task{}, m.getByIDErr
	}
	return m.storedTask, nil
}
func (m *mockTaskRepo) Update(_ context.Context, t domaintask.Task) (domaintask.Task, error) {
	return t, nil
}
func (m *mockTaskRepo) Delete(_ context.Context, _ string) error            { return nil }
func (m *mockTaskRepo) List(_ context.Context) ([]domaintask.Task, error) {
	return []domaintask.Task{}, nil
}

// --- AddComment tests ---

func TestAddComment_Success(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{storedTask: domaintask.Task{ID: "task-1"}},
	)

	got, err := svc.AddComment(context.Background(), "task-1", "Great progress!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
	if got.TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got %q", got.TaskID)
	}
	if got.Content != "Great progress!" {
		t.Errorf("expected content 'Great progress!', got %q", got.Content)
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestAddComment_EmptyContent(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{storedTask: domaintask.Task{ID: "task-1"}},
	)

	_, err := svc.AddComment(context.Background(), "task-1", "   ")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
	var ve *appcomment.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *appcomment.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["content"]; !ok {
		t.Error("expected ve.Fields[\"content\"] to be set")
	}
}

func TestAddComment_ContentTooLong(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{storedTask: domaintask.Task{ID: "task-1"}},
	)

	_, err := svc.AddComment(context.Background(), "task-1", strings.Repeat("x", 2001))
	if err == nil {
		t.Fatal("expected error for content > 2000 chars, got nil")
	}
	var ve *appcomment.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *appcomment.ValidationError, got %T", err)
	}
	if _, ok := ve.Fields["content"]; !ok {
		t.Error("expected ve.Fields[\"content\"] to be set")
	}
}

func TestAddComment_TaskNotFound(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{getByIDErr: domaintask.ErrNotFound},
	)

	_, err := svc.AddComment(context.Background(), "missing-task", "Hello")
	if !errors.Is(err, domaintask.ErrNotFound) {
		t.Fatalf("expected domaintask.ErrNotFound, got %v", err)
	}
}

// --- ListComments tests ---

func TestListComments_Empty(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{storedTask: domaintask.Task{ID: "task-1"}},
	)

	comments, err := svc.ListComments(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comments == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(comments) != 0 {
		t.Errorf("expected empty slice, got %d comments", len(comments))
	}
}

func TestListComments_ReturnsComments(t *testing.T) {
	stored := []domaincomment.Comment{
		{ID: "c1", TaskID: "task-1", Content: "First", CreatedAt: time.Now().UTC()},
		{ID: "c2", TaskID: "task-1", Content: "Second", CreatedAt: time.Now().UTC()},
	}
	svc := appcomment.NewService(
		&mockCommentRepo{storedComments: stored},
		&mockTaskRepo{storedTask: domaintask.Task{ID: "task-1"}},
	)

	comments, err := svc.ListComments(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != "c1" {
		t.Errorf("expected first ID 'c1', got %q", comments[0].ID)
	}
}

func TestListComments_TaskNotFound(t *testing.T) {
	svc := appcomment.NewService(
		&mockCommentRepo{},
		&mockTaskRepo{getByIDErr: domaintask.ErrNotFound},
	)

	_, err := svc.ListComments(context.Background(), "missing-task")
	if !errors.Is(err, domaintask.ErrNotFound) {
		t.Fatalf("expected domaintask.ErrNotFound, got %v", err)
	}
}
