# Task Comments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a comments sub-resource to tasks with POST and GET endpoints, cascade-delete comments when a task is deleted, and full integration test coverage.

**Architecture:** Domain-first DDD — Comment entity and repository interface in `internal/domain/comment/`, use cases in `internal/application/comment/`, postgres implementation in `internal/infrastructure/persistence/postgres/`. A narrow `CommentDeleter` interface in the task application package lets the task service delete comments on soft-delete without importing the comment package. All routes sit behind the existing API key middleware.

**Tech Stack:** Go 1.25, chi v5, pgx v5, google/uuid, standard-library testing with hand-written mocks (no testify/gomock — match existing pattern).

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `migrations/000004_create_comments.up.sql` | DDL for comments table with FK + CASCADE |
| Create | `migrations/000004_create_comments.down.sql` | Drops comments table |
| Create | `internal/domain/comment/entity.go` | Comment struct |
| Create | `internal/domain/comment/repository.go` | Repository interface + ErrNotFound |
| Modify | `internal/application/task/service.go` | CommentDeleter interface, WithCommentDeleter, nil-safe cascade in DeleteTask |
| Modify | `internal/application/task/service_test.go` | Two new tests for CommentDeleter wiring |
| Create | `internal/application/comment/service_test.go` | Unit tests (written before implementation) |
| Create | `internal/application/comment/service.go` | AddComment + ListComments use cases |
| Create | `internal/infrastructure/persistence/postgres/comment_repository.go` | Postgres implementation |
| Create | `internal/infrastructure/http/handler/comment.go` | HTTP handler (Create + List) |
| Modify | `internal/infrastructure/http/router.go` | Wire comment repo/svc/handler + new routes |
| Modify | `internal/infrastructure/http/handler/task_integration_test.go` | TestMain: create/drop comments table |
| Create | `internal/infrastructure/http/handler/comment_integration_test.go` | 7 integration tests |

---

## Task 1: SQL Migrations

**Files:**
- Create: `migrations/000004_create_comments.up.sql`
- Create: `migrations/000004_create_comments.down.sql`

- [ ] **Step 1: Create the up migration**

`migrations/000004_create_comments.up.sql`:
```sql
CREATE TABLE comments (
    id         TEXT        PRIMARY KEY,
    task_id    TEXT        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

- [ ] **Step 2: Create the down migration**

`migrations/000004_create_comments.down.sql`:
```sql
DROP TABLE IF EXISTS comments;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000004_create_comments.up.sql migrations/000004_create_comments.down.sql
git commit -m "feat: add comments table migration with FK to tasks"
```

---

## Task 2: Domain Layer

**Files:**
- Create: `internal/domain/comment/entity.go`
- Create: `internal/domain/comment/repository.go`

- [ ] **Step 1: Create the Comment entity**

`internal/domain/comment/entity.go`:
```go
package comment

import "time"

type Comment struct {
	ID        string
	TaskID    string
	Content   string
	CreatedAt time.Time
}
```

- [ ] **Step 2: Create the repository interface**

`internal/domain/comment/repository.go`:
```go
package comment

import (
	"context"
	"errors"
)

// ErrNotFound is returned when no comment matches the given criteria.
var ErrNotFound = errors.New("comment not found")

type Repository interface {
	Create(ctx context.Context, c Comment) (Comment, error)
	ListByTaskID(ctx context.Context, taskID string) ([]Comment, error)
	// DeleteByTaskID removes all comments for the given task. No-op if none exist.
	DeleteByTaskID(ctx context.Context, taskID string) error
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/domain/comment/...
```
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/comment/
git commit -m "feat: add Comment domain entity and repository interface"
```

---

## Task 3: Task Service — CommentDeleter Wiring

**Files:**
- Modify: `internal/application/task/service.go`
- Modify: `internal/application/task/service_test.go`

- [ ] **Step 1: Write the failing tests first**

Add to `internal/application/task/service_test.go` (append after `TestDeleteTask_NotFound`):

```go
// mockCommentDeleter records calls to DeleteByTaskID.
type mockCommentDeleter struct {
	called    bool
	taskIDArg string
	returnErr error
}

func (m *mockCommentDeleter) DeleteByTaskID(_ context.Context, taskID string) error {
	m.called = true
	m.taskIDArg = taskID
	return m.returnErr
}

func TestDeleteTask_CallsCommentDeleter(t *testing.T) {
	deleter := &mockCommentDeleter{}
	svc := apptask.NewService(&mockRepo{})
	svc.WithCommentDeleter(deleter)

	err := svc.DeleteTask(context.Background(), "some-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleter.called {
		t.Error("expected commentDeleter.DeleteByTaskID to be called")
	}
	if deleter.taskIDArg != "some-id" {
		t.Errorf("expected taskID %q, got %q", "some-id", deleter.taskIDArg)
	}
}

func TestDeleteTask_NilCommentDeleter(t *testing.T) {
	// DeleteTask must not panic when no comment deleter is set.
	svc := apptask.NewService(&mockRepo{})

	err := svc.DeleteTask(context.Background(), "some-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run the tests — expect compilation failure**

```bash
go test ./internal/application/task/... -run TestDeleteTask_CallsCommentDeleter -v
```
Expected: compile error — `svc.WithCommentDeleter undefined`.

- [ ] **Step 3: Add CommentDeleter interface and wiring to service.go**

In `internal/application/task/service.go`, add the `CommentDeleter` interface after the `validStatuses` block:

```go
// CommentDeleter is satisfied by any type that can remove all comments for a task.
// The comment repository implements this automatically.
type CommentDeleter interface {
	DeleteByTaskID(ctx context.Context, taskID string) error
}
```

Change the `Service` struct from:
```go
type Service struct {
	repo domain.Repository
}
```
to:
```go
type Service struct {
	repo            domain.Repository
	commentDeleter  CommentDeleter
}
```

Add `WithCommentDeleter` after `NewService`:
```go
// WithCommentDeleter wires in a comment deleter so that DeleteTask also removes
// all comments belonging to the task. Safe to call with nil — no-op if unset.
func (s *Service) WithCommentDeleter(d CommentDeleter) {
	s.commentDeleter = d
}
```

Replace `DeleteTask`:
```go
// DeleteTask soft-deletes a task and removes its comments. Returns domain.ErrNotFound
// if the task does not exist or has already been deleted.
func (s *Service) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.commentDeleter != nil {
		if err := s.commentDeleter.DeleteByTaskID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run all task service tests — expect all pass**

```bash
go test ./internal/application/task/... -v
```
Expected: all tests PASS, including the two new ones.

- [ ] **Step 5: Commit**

```bash
git add internal/application/task/service.go internal/application/task/service_test.go
git commit -m "feat: add CommentDeleter interface and cascade delete to task service"
```

---

## Task 4: Comment Application Service (TDD)

**Files:**
- Create: `internal/application/comment/service_test.go`
- Create: `internal/application/comment/service.go`

- [ ] **Step 1: Write the tests first**

Create `internal/application/comment/service_test.go`:

```go
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
func (m *mockTaskRepo) Delete(_ context.Context, _ string) error             { return nil }
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
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./internal/application/comment/... -v
```
Expected: compile error — package `comment` does not exist yet.

- [ ] **Step 3: Implement the service**

Create `internal/application/comment/service.go`:

```go
package comment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domaincomment "github.com/lucaspalencia/superpowers/internal/domain/comment"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

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

// Service implements the comment use cases.
type Service struct {
	repo     domaincomment.Repository
	taskRepo domaintask.Repository
}

// NewService constructs a Service with the given repositories.
func NewService(repo domaincomment.Repository, taskRepo domaintask.Repository) *Service {
	return &Service{repo: repo, taskRepo: taskRepo}
}

// AddComment validates input, verifies the parent task exists, and persists the comment.
func (s *Service) AddComment(ctx context.Context, taskID, content string) (domaincomment.Comment, error) {
	errs := make(map[string]string)

	if strings.TrimSpace(content) == "" {
		errs["content"] = "content is required"
	} else if len(content) > 2000 {
		errs["content"] = "content must be 2000 characters or fewer"
	}

	if len(errs) > 0 {
		return domaincomment.Comment{}, &ValidationError{Fields: errs}
	}

	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return domaincomment.Comment{}, err
	}

	comment := domaincomment.Comment{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}

	return s.repo.Create(ctx, comment)
}

// ListComments verifies the parent task exists, then returns all comments ordered
// by created_at ASC. Always returns a non-nil slice.
func (s *Service) ListComments(ctx context.Context, taskID string) ([]domaincomment.Comment, error) {
	if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
		return nil, err
	}

	comments, err := s.repo.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		return []domaincomment.Comment{}, nil
	}
	return comments, nil
}
```

- [ ] **Step 4: Run all comment service tests — expect all pass**

```bash
go test ./internal/application/comment/... -v
```
Expected: all 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/comment/
git commit -m "feat: add comment application service with AddComment and ListComments"
```

---

## Task 5: Postgres Comment Repository

**Files:**
- Create: `internal/infrastructure/persistence/postgres/comment_repository.go`

- [ ] **Step 1: Implement the repository**

Create `internal/infrastructure/persistence/postgres/comment_repository.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/lucaspalencia/superpowers/internal/domain/comment"
)

// CommentRepository implements domain/comment.Repository using PostgreSQL.
type CommentRepository struct {
	pool *pgxpool.Pool
}

// NewCommentRepository constructs a CommentRepository backed by the given pool.
func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

// Create inserts a comment and returns the persisted row.
func (r *CommentRepository) Create(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO comments (id, task_id, content, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, task_id, content, created_at`,
		c.ID, c.TaskID, c.Content, c.CreatedAt,
	)

	var out domain.Comment
	if err := row.Scan(&out.ID, &out.TaskID, &out.Content, &out.CreatedAt); err != nil {
		return domain.Comment{}, fmt.Errorf("scan comment: %w", err)
	}
	return out, nil
}

// ListByTaskID returns all comments for a task ordered by created_at ASC.
func (r *CommentRepository) ListByTaskID(ctx context.Context, taskID string) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, content, created_at
		 FROM comments
		 WHERE task_id = $1
		 ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return comments, nil
}

// DeleteByTaskID removes all comments for a task. No-op if none exist.
func (r *CommentRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM comments WHERE task_id = $1`,
		taskID,
	); err != nil {
		return fmt.Errorf("delete comments: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/persistence/postgres/...
```
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/persistence/postgres/comment_repository.go
git commit -m "feat: add postgres CommentRepository"
```

---

## Task 6: Comment HTTP Handler

**Files:**
- Create: `internal/infrastructure/http/handler/comment.go`

- [ ] **Step 1: Implement the handler**

Create `internal/infrastructure/http/handler/comment.go`:

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	appcomment "github.com/lucaspalencia/superpowers/internal/application/comment"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type commentRequest struct {
	Content string `json:"content"`
}

type commentResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// CommentHandler handles HTTP requests for comment operations.
type CommentHandler struct {
	svc *appcomment.Service
}

// NewCommentHandler constructs a CommentHandler with the given service.
func NewCommentHandler(svc *appcomment.Service) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// Create handles POST /tasks/{id}/comments. Returns 201 on success.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req commentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	comment, err := h.svc.AddComment(r.Context(), taskID, req.Content)
	if err != nil {
		var ve *appcomment.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": ve.Fields})
			return
		}
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, commentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
	})
}

// List handles GET /tasks/{id}/comments. Returns 200 with array ordered by created_at ASC.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	comments, err := h.svc.ListComments(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, commentResponse{
			ID:        c.ID,
			TaskID:    c.TaskID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/http/handler/...
```
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/handler/comment.go
git commit -m "feat: add CommentHandler with Create and List"
```

---

## Task 7: Router Wiring

**Files:**
- Modify: `internal/infrastructure/http/router.go`

- [ ] **Step 1: Replace router.go with the updated version**

Replace the entire contents of `internal/infrastructure/http/router.go`:

```go
package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appcomment "github.com/lucaspalencia/superpowers/internal/application/comment"
	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/handler"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/middleware"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func Register(pool *pgxpool.Pool, apiKey string) nethttp.Handler {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler(pool)
	r.Get("/health", healthHandler.ServeHTTP)

	taskRepo := postgres.NewTaskRepository(pool)
	commentRepo := postgres.NewCommentRepository(pool)

	taskSvc := apptask.NewService(taskRepo)
	taskSvc.WithCommentDeleter(commentRepo)
	taskHandler := handler.NewTaskHandler(taskSvc)

	commentSvc := appcomment.NewService(commentRepo, taskRepo)
	commentHandler := handler.NewCommentHandler(commentSvc)

	r.With(middleware.APIKey(apiKey)).Post("/tasks", taskHandler.Create)
	r.With(middleware.APIKey(apiKey)).Get("/tasks", taskHandler.List)
	r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}", taskHandler.GetByID)
	r.With(middleware.APIKey(apiKey)).Patch("/tasks/{id}", taskHandler.Update)
	r.With(middleware.APIKey(apiKey)).Delete("/tasks/{id}", taskHandler.Delete)

	r.With(middleware.APIKey(apiKey)).Post("/tasks/{id}/comments", commentHandler.Create)
	r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}/comments", commentHandler.List)

	return r
}
```

- [ ] **Step 2: Verify the whole project compiles**

```bash
go build ./...
```
Expected: no output (success).

- [ ] **Step 3: Run all unit tests**

```bash
go test ./internal/application/... -v
```
Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/http/router.go
git commit -m "feat: wire comment handler and routes into router"
```

---

## Task 8: Integration Tests — TestMain + Comment Tests

**Files:**
- Modify: `internal/infrastructure/http/handler/task_integration_test.go`
- Create: `internal/infrastructure/http/handler/comment_integration_test.go`

- [ ] **Step 1: Add createCommentsSQL constant and update TestMain**

In `internal/infrastructure/http/handler/task_integration_test.go`, add the `createCommentsSQL` constant directly after the existing `createTasksSQL` constant:

```go
const createCommentsSQL = `
CREATE TABLE IF NOT EXISTS comments (
    id         TEXT        PRIMARY KEY,
    task_id    TEXT        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
)`
```

In `TestMain`, after the existing `pool.Exec(context.Background(), createTasksSQL)` block, add:

```go
	if _, err = pool.Exec(context.Background(), createCommentsSQL); err != nil {
		fmt.Printf("create comments table: %v\n", err)
		pool.Close()
		os.Exit(1)
	}
```

Replace the teardown line:
```go
	pool.Exec(context.Background(), "DROP TABLE IF EXISTS tasks") //nolint:errcheck
```
with:
```go
	pool.Exec(context.Background(), "DROP TABLE IF EXISTS comments") //nolint:errcheck
	pool.Exec(context.Background(), "DROP TABLE IF EXISTS tasks")    //nolint:errcheck
```

- [ ] **Step 2: Write the comment integration tests**

Create `internal/infrastructure/http/handler/comment_integration_test.go`:

```go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAddComment_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task for comments")

	body := `{"content":"This is a comment"}`
	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["id"] == "" || result["id"] == nil {
		t.Error("expected non-empty id in response")
	}
	if result["task_id"] != taskID {
		t.Errorf("expected task_id %q, got %v", taskID, result["task_id"])
	}
	if result["content"] != "This is a comment" {
		t.Errorf("expected content 'This is a comment', got %v", result["content"])
	}
	if result["created_at"] == "" || result["created_at"] == nil {
		t.Error("expected non-empty created_at in response")
	}
}

func TestAddComment_Integration_TaskNotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000/comments",
		bytes.NewBufferString(`{"content":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/{id}/comments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != "task not found" {
		t.Errorf("expected error 'task not found', got %v", result["error"])
	}
}

func TestAddComment_Integration_ValidationError(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task for validation test")

	req, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(`{"content":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errs, ok := result["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'errors' object in response body, got %v", result)
	}
	if _, ok := errs["content"]; !ok {
		t.Error("expected errors[\"content\"] to be set")
	}
}

func TestListComments_Integration_Empty(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task with no comments")

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result []any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestListComments_Integration_OrderedAsc(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task with ordered comments")

	addComment := func(content string) {
		req, _ := http.NewRequest(http.MethodPost,
			integServer.URL+"/tasks/"+taskID+"/comments",
			bytes.NewBufferString(`{"content":"`+content+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST comment: %v", err)
		}
		resp.Body.Close()
	}

	addComment("First comment")
	addComment("Second comment")

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(result))
	}
	if result[0]["content"] != "First comment" {
		t.Errorf("expected first comment 'First comment', got %v", result[0]["content"])
	}
	if result[1]["content"] != "Second comment" {
		t.Errorf("expected second comment 'Second comment', got %v", result[1]["content"])
	}
}

func TestListComments_Integration_TaskNotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000/comments", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/{id}/comments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != "task not found" {
		t.Errorf("expected error 'task not found', got %v", result["error"])
	}
}

func TestDeleteTask_Integration_CascadeDeletesComments(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks CASCADE") //nolint:errcheck
	})

	taskID := createTestTask(t, "Task to cascade-delete")

	// Add a comment to the task.
	addReq, _ := http.NewRequest(http.MethodPost,
		integServer.URL+"/tasks/"+taskID+"/comments",
		bytes.NewBufferString(`{"content":"Should be deleted"}`))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatalf("POST comment: %v", err)
	}
	addResp.Body.Close()
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding comment, got %d", addResp.StatusCode)
	}

	// Delete the task (soft-delete + application-layer comment cleanup).
	delReq, _ := http.NewRequest(http.MethodDelete,
		integServer.URL+"/tasks/"+taskID, nil)
	delReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /tasks/%s: %v", taskID, err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting task, got %d", delResp.StatusCode)
	}

	// API-level: GET /tasks/{id}/comments returns 404 (task is gone).
	getReq, _ := http.NewRequest(http.MethodGet,
		integServer.URL+"/tasks/"+taskID+"/comments", nil)
	getReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET /tasks/%s/comments: %v", taskID, err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after task delete, got %d", getResp.StatusCode)
	}

	// DB-level: confirm comment rows are physically gone.
	var count int
	row := integPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM comments WHERE task_id = $1", taskID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 comment rows after task delete, got %d", count)
	}
}

// createTestTask creates a task via POST /tasks and returns its ID.
func createTestTask(t *testing.T, title string) string {
	t.Helper()
	body := `{"title":"` + title + `"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode create task response: %v", err)
	}
	id, _ := result["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty task ID from POST /tasks")
	}
	return id
}
```

- [ ] **Step 3: Run unit tests (no DB required)**

```bash
go test ./... -v -short
```
Expected: all unit tests PASS, integration tests skipped (no `TEST_DATABASE_URL`).

- [ ] **Step 4: Start test database**

```bash
docker compose -f docker-compose.test.yml up -d
```
Expected: container `db-test` starts on port 5433.

- [ ] **Step 5: Run integration tests**

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable" \
  go test ./internal/infrastructure/http/handler/... -v -count=1
```
Expected: all integration tests PASS including the 7 new comment tests.

- [ ] **Step 6: Stop test database**

```bash
docker compose -f docker-compose.test.yml down
```

- [ ] **Step 7: Commit**

```bash
git add internal/infrastructure/http/handler/task_integration_test.go \
        internal/infrastructure/http/handler/comment_integration_test.go
git commit -m "test: add comment integration tests and update TestMain for comments table"
```
