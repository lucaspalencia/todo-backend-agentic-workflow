package comment_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// stubTaskRepo is a minimal in-memory task.Repository for testing.
type stubTaskRepo struct {
	existing *domtask.Task
	err      error
}

func (s *stubTaskRepo) Create(_ context.Context, _ *domtask.Task) error  { return nil }
func (s *stubTaskRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) {
	return s.existing, s.err
}
func (s *stubTaskRepo) FindAll(_ context.Context) ([]domtask.Task, error) { return nil, nil }
func (s *stubTaskRepo) Save(_ context.Context, _ *domtask.Task) error     { return nil }
func (s *stubTaskRepo) Delete(_ context.Context, _ string) error          { return nil }

// stubCommentRepo is a minimal in-memory comment.Repository for testing.
type stubCommentRepo struct {
	created  *domcomment.Comment
	comments []domcomment.Comment
	err      error
}

func (s *stubCommentRepo) Create(_ context.Context, c *domcomment.Comment) error {
	if s.err != nil {
		return s.err
	}
	s.created = c
	return nil
}
func (s *stubCommentRepo) FindByTaskID(_ context.Context, _ string) ([]domcomment.Comment, error) {
	return s.comments, s.err
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var ve appcomment.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	for _, e := range ve {
		if e.Field == field {
			return
		}
	}
	t.Errorf("expected validation error for field %q, got: %v", field, err)
}

func existingTask() *domtask.Task {
	return &domtask.Task{ID: "task-1", Title: "Test task", Status: domtask.StatusPending}
}

func TestAddComment_ValidInput_ReturnsComment(t *testing.T) {
	taskRepo := &stubTaskRepo{existing: existingTask()}
	commentRepo := &stubCommentRepo{}
	svc := appcomment.NewService(taskRepo, commentRepo)

	c, err := svc.AddComment(context.Background(), "task-1", "Nice work!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID == "" {
		t.Error("expected non-empty ID")
	}
	if c.TaskID != "task-1" {
		t.Errorf("expected task_id=%q, got %q", "task-1", c.TaskID)
	}
	if c.Content != "Nice work!" {
		t.Errorf("expected content=%q, got %q", "Nice work!", c.Content)
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestAddComment_BlankContent_ReturnsValidationError(t *testing.T) {
	svc := appcomment.NewService(&stubTaskRepo{existing: existingTask()}, &stubCommentRepo{})
	_, err := svc.AddComment(context.Background(), "task-1", "   ")
	assertValidationField(t, err, "content")
}

func TestAddComment_ContentTooLong_ReturnsValidationError(t *testing.T) {
	svc := appcomment.NewService(&stubTaskRepo{existing: existingTask()}, &stubCommentRepo{})
	_, err := svc.AddComment(context.Background(), "task-1", strings.Repeat("a", 2001))
	assertValidationField(t, err, "content")
}

func TestAddComment_ContentAtMaxLength_Succeeds(t *testing.T) {
	svc := appcomment.NewService(&stubTaskRepo{existing: existingTask()}, &stubCommentRepo{})
	_, err := svc.AddComment(context.Background(), "task-1", strings.Repeat("a", 2000))
	if err != nil {
		t.Fatalf("unexpected error for max-length content: %v", err)
	}
}

func TestAddComment_TaskNotFound_ReturnsErrNotFound(t *testing.T) {
	svc := appcomment.NewService(&stubTaskRepo{existing: nil}, &stubCommentRepo{})
	_, err := svc.AddComment(context.Background(), "missing", "hello")
	if !errors.Is(err, domtask.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListComments_Success_ReturnsSlice(t *testing.T) {
	comments := []domcomment.Comment{
		{ID: "c1", TaskID: "task-1", Content: "first"},
		{ID: "c2", TaskID: "task-1", Content: "second"},
	}
	svc := appcomment.NewService(
		&stubTaskRepo{existing: existingTask()},
		&stubCommentRepo{comments: comments},
	)

	result, err := svc.ListComments(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 comments, got %d", len(result))
	}
}

func TestListComments_TaskNotFound_ReturnsErrNotFound(t *testing.T) {
	svc := appcomment.NewService(&stubTaskRepo{existing: nil}, &stubCommentRepo{})
	_, err := svc.ListComments(context.Background(), "missing")
	if !errors.Is(err, domtask.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
