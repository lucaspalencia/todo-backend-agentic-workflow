# Delete, List, and Get Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `DELETE /tasks/{id}` (soft delete), `GET /tasks` (list active tasks newest-first), and `GET /tasks/{id}` (fetch single active task) endpoints to the existing Go DDD task API.

**Architecture:** Soft-delete is handled transparently in the infrastructure layer — `domain.Task` gains no `deleted_at` field. The repository filters `WHERE deleted_at IS NULL` so callers treat deleted tasks identically to absent ones (`ErrNotFound`). Three new use cases (`DeleteTask`, `ListTasks`, `GetTaskByID`) are added to the existing `Service`.

**Tech Stack:** Go, chi router, pgx v5 (PostgreSQL), `httptest` for integration tests, `go-migrate` for migrations.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `migrations/000003_soft_delete_tasks.up.sql` | Create | Add `deleted_at` column |
| `migrations/000003_soft_delete_tasks.down.sql` | Create | Drop `deleted_at` column |
| `internal/domain/task/repository.go` | Modify | Add `Delete` and `List` to interface |
| `internal/application/task/service.go` | Modify | Add `DeleteTask`, `ListTasks`, `GetTaskByID` |
| `internal/application/task/service_test.go` | Modify | Add mock methods + unit tests for new use cases |
| `internal/infrastructure/persistence/postgres/task_repository.go` | Modify | Implement `Delete`, `List`; update `GetByID` |
| `internal/infrastructure/http/handler/task.go` | Modify | Add `Delete`, `List`, `GetByID` handlers |
| `internal/infrastructure/http/router.go` | Modify | Register three new routes |
| `internal/infrastructure/http/handler/task_integration_test.go` | Modify | Add `deleted_at` to schema + new test cases |

---

## Task 1: Database migration — add `deleted_at` column

**Files:**
- Create: `migrations/000003_soft_delete_tasks.up.sql`
- Create: `migrations/000003_soft_delete_tasks.down.sql`

- [ ] **Step 1: Create the up migration**

File: `migrations/000003_soft_delete_tasks.up.sql`
```sql
ALTER TABLE tasks ADD COLUMN deleted_at TIMESTAMPTZ;
```

- [ ] **Step 2: Create the down migration**

File: `migrations/000003_soft_delete_tasks.down.sql`
```sql
ALTER TABLE tasks DROP COLUMN deleted_at;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000003_soft_delete_tasks.up.sql migrations/000003_soft_delete_tasks.down.sql
git commit -m "feat: add deleted_at column migration for task soft delete"
```

---

## Task 2: Extend the domain Repository interface

**Files:**
- Modify: `internal/domain/task/repository.go`

- [ ] **Step 1: Add `Delete` and `List` to the Repository interface**

Replace the entire file content of `internal/domain/task/repository.go`:

```go
package task

import (
	"context"
	"errors"
)

// ErrNotFound is returned by the repository when no task matches the given ID.
var ErrNotFound = errors.New("task not found")

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
	GetByID(ctx context.Context, id string) (Task, error)
	Update(ctx context.Context, task Task) (Task, error)
	// Delete soft-deletes a task. Returns ErrNotFound if the task does not
	// exist or has already been deleted.
	Delete(ctx context.Context, id string) error
	// List returns all active (non-deleted) tasks ordered by created_at DESC.
	List(ctx context.Context) ([]Task, error)
}
```

- [ ] **Step 2: Verify the project still compiles**

Run:
```bash
go build ./...
```

Expected: compile error — `TaskRepository` in `postgres` package no longer satisfies the interface (missing `Delete` and `List`). This is expected; we will implement them in Task 4.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/task/repository.go
git commit -m "feat: add Delete and List to task Repository interface"
```

---

## Task 3: Service unit tests + implement DeleteTask, ListTasks, GetTaskByID

**Files:**
- Modify: `internal/application/task/service_test.go`
- Modify: `internal/application/task/service.go`

- [ ] **Step 1: Extend the mock repository in service_test.go**

Add `deleteErr`, `listTasks`, and `listErr` fields to `mockRepo`, and implement the two new interface methods. The complete updated `mockRepo` struct and its methods (replace only the struct definition and methods block at the top of service_test.go — keep all existing test functions unchanged):

```go
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
```

- [ ] **Step 2: Write failing unit tests for DeleteTask**

Append to `internal/application/task/service_test.go`:

```go
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
```

- [ ] **Step 3: Write failing unit tests for ListTasks**

Append to `internal/application/task/service_test.go`:

```go
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
```

- [ ] **Step 4: Write failing unit tests for GetTaskByID**

Append to `internal/application/task/service_test.go`:

```go
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
```

- [ ] **Step 5: Run the new tests to verify they fail**

```bash
go test ./internal/application/task/... -run "TestDeleteTask|TestListTasks|TestGetTaskByID" -v
```

Expected: FAIL — `svc.DeleteTask`, `svc.ListTasks`, `svc.GetTaskByID` undefined.

- [ ] **Step 6: Implement DeleteTask, ListTasks, GetTaskByID in service.go**

Append to the end of `internal/application/task/service.go`:

```go
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
```

- [ ] **Step 7: Run the new tests to verify they pass**

```bash
go test ./internal/application/task/... -v
```

Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/application/task/service.go internal/application/task/service_test.go
git commit -m "feat: add DeleteTask, ListTasks, GetTaskByID use cases"
```

---

## Task 4: Postgres repository — implement Delete and List, update GetByID

**Files:**
- Modify: `internal/infrastructure/persistence/postgres/task_repository.go`

- [ ] **Step 1: Add `Delete`, `List`, and update `GetByID`**

Replace the entire file `internal/infrastructure/persistence/postgres/task_repository.go` with:

```go
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/lucaspalencia/superpowers/internal/domain/task"
)

// TaskRepository implements domain/task.Repository using PostgreSQL.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository constructs a TaskRepository backed by the given pool.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// Create inserts a task and returns the persisted row.
func (r *TaskRepository) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, title, description, status, created_at, updated_at`,
		task.ID, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}

// GetByID fetches a single active task by its ID.
// Returns domain.ErrNotFound if no row matches or the task is soft-deleted.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, domain.ErrNotFound
		}
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}

// Update writes the full task back to the database and returns the persisted row.
func (r *TaskRepository) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE tasks
		 SET title = $1, description = $2, status = $3, updated_at = $4
		 WHERE id = $5
		 RETURNING id, title, description, status, created_at, updated_at`,
		task.Title, task.Description, task.Status, task.UpdatedAt, task.ID,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}

// Delete soft-deletes a task by setting deleted_at to the current time.
// Returns domain.ErrNotFound if the task does not exist or is already deleted.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List returns all active (non-deleted) tasks ordered by created_at DESC.
func (r *TaskRepository) List(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}
```

- [ ] **Step 2: Verify the project compiles**

```bash
go build ./...
```

Expected: success — `TaskRepository` now satisfies the updated interface.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/persistence/postgres/task_repository.go
git commit -m "feat: implement Delete and List in postgres task repository, update GetByID filter"
```

---

## Task 5: HTTP handler — add Delete, List, GetByID

**Files:**
- Modify: `internal/infrastructure/http/handler/task.go`

- [ ] **Step 1: Add the three handler methods**

Append to the end of `internal/infrastructure/http/handler/task.go` (after the `Update` method, before `writeJSON`):

```go
// Delete handles DELETE /tasks/{id}. Returns 204 on success, 404 if not found.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.svc.DeleteTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /tasks. Returns 200 with a JSON array of all active tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListTasks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, taskResponse{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /tasks/{id}. Returns 200 with the task, 404 if not found or deleted.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	task, err := h.svc.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domaintask.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}
```

- [ ] **Step 2: Verify the project compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/handler/task.go
git commit -m "feat: add Delete, List, GetByID HTTP handlers for tasks"
```

---

## Task 6: Router — register the three new routes

**Files:**
- Modify: `internal/infrastructure/http/router.go`

- [ ] **Step 1: Add the three routes**

Replace the entire file `internal/infrastructure/http/router.go` with:

```go
package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	taskSvc := apptask.NewService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskSvc)

	r.With(middleware.APIKey(apiKey)).Post("/tasks", taskHandler.Create)
	r.With(middleware.APIKey(apiKey)).Get("/tasks", taskHandler.List)
	r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}", taskHandler.GetByID)
	r.With(middleware.APIKey(apiKey)).Patch("/tasks/{id}", taskHandler.Update)
	r.With(middleware.APIKey(apiKey)).Delete("/tasks/{id}", taskHandler.Delete)

	return r
}
```

- [ ] **Step 2: Verify the project compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/router.go
git commit -m "feat: register GET /tasks, GET /tasks/{id}, DELETE /tasks/{id} routes"
```

---

## Task 7: Integration tests — DELETE /tasks/{id}

**Files:**
- Modify: `internal/infrastructure/http/handler/task_integration_test.go`

- [ ] **Step 1: Update `createTasksSQL` to include `deleted_at`**

In `task_integration_test.go`, find the `createTasksSQL` constant and replace it:

```go
const createTasksSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    deleted_at  TIMESTAMPTZ
)`
```

- [ ] **Step 2: Write integration tests for DELETE /tasks/{id}**

Append to `task_integration_test.go`:

```go
func TestDeleteTask_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task to delete
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(`{"title":"Task to delete"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer createResp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)

	// Delete the task
	delReq, _ := http.NewRequest(http.MethodDelete, integServer.URL+"/tasks/"+id, nil)
	delReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /tasks/%s: %v", id, err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// Verify the task is actually gone — GET should return 404
	getReq, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks/"+id, nil)
	getReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET /tasks/%s: %v", id, err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestDeleteTask_Integration_NotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(
		http.MethodDelete,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /tasks: %v", err)
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

func TestDeleteTask_Integration_AlreadyDeleted(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(`{"title":"Delete twice"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer createResp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)

	del := func() int {
		req, _ := http.NewRequest(http.MethodDelete, integServer.URL+"/tasks/"+id, nil)
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("DELETE /tasks/%s: %v", id, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := del(); code != http.StatusNoContent {
		t.Fatalf("first delete: expected 204, got %d", code)
	}
	if code := del(); code != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d", code)
	}
}
```

- [ ] **Step 3: Start the test database**

```bash
docker compose -f docker-compose.test.yml up -d
```

Wait a moment, then run:

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable" \
  go test ./internal/infrastructure/http/handler/... \
  -run "TestDeleteTask_Integration" -v
```

Expected: all three DELETE tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/http/handler/task_integration_test.go
git commit -m "test: add integration tests for DELETE /tasks/{id}"
```

---

## Task 8: Integration tests — GET /tasks and GET /tasks/{id}

**Files:**
- Modify: `internal/infrastructure/http/handler/task_integration_test.go`

- [ ] **Step 1: Write integration tests for GET /tasks**

Append to `task_integration_test.go`:

```go
func TestListTasks_Integration_Empty(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck

	req, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
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

func TestListTasks_Integration_NewestFirst(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	create := func(title string) string {
		req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
			bytes.NewBufferString(`{"title":"`+title+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /tasks: %v", err)
		}
		defer resp.Body.Close()
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		id, _ := result["id"].(string)
		return id
	}

	id1 := create("First task")
	id2 := create("Second task")

	req, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
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
		t.Fatalf("expected 2 tasks, got %d", len(result))
	}
	// Newest first: id2 was created after id1
	if result[0]["id"] != id2 {
		t.Errorf("expected newest task first (id=%q), got id=%q", id2, result[0]["id"])
	}
	if result[1]["id"] != id1 {
		t.Errorf("expected oldest task second (id=%q), got id=%q", id1, result[1]["id"])
	}
}

func TestListTasks_Integration_ExcludesDeleted(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create two tasks
	create := func(title string) string {
		req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
			bytes.NewBufferString(`{"title":"`+title+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /tasks: %v", err)
		}
		defer resp.Body.Close()
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		id, _ := result["id"].(string)
		return id
	}

	id1 := create("Keep this")
	id2 := create("Delete this")

	// Delete id2
	delReq, _ := http.NewRequest(http.MethodDelete, integServer.URL+"/tasks/"+id2, nil)
	delReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /tasks/%s: %v", id2, err)
	}
	delResp.Body.Close()

	// List should only return id1
	req, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	defer resp.Body.Close()

	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if result[0]["id"] != id1 {
		t.Errorf("expected id=%q, got id=%q", id1, result[0]["id"])
	}
}
```

- [ ] **Step 2: Write integration tests for GET /tasks/{id}**

Append to `task_integration_test.go`:

```go
func TestGetTaskByID_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(`{"title":"Fetchable task","description":"Some description","status":"in_progress"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer createResp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)

	// Fetch by ID
	req, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["id"] != id {
		t.Errorf("expected id=%q, got %v", id, result["id"])
	}
	if result["title"] != "Fetchable task" {
		t.Errorf("expected title 'Fetchable task', got %v", result["title"])
	}
	if result["description"] != "Some description" {
		t.Errorf("expected description 'Some description', got %v", result["description"])
	}
	if result["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", result["status"])
	}
	if result["created_at"] == "" || result["created_at"] == nil {
		t.Error("expected non-empty created_at")
	}
}

func TestGetTaskByID_Integration_DeletedReturns404(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create and then delete a task
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks",
		bytes.NewBufferString(`{"title":"Soon deleted"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer createResp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)

	delReq, _ := http.NewRequest(http.MethodDelete, integServer.URL+"/tasks/"+id, nil)
	delReq.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE /tasks/%s: %v", id, err)
	}
	delResp.Body.Close()

	// GET should now return 404
	req, _ := http.NewRequest(http.MethodGet, integServer.URL+"/tasks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/%s: %v", id, err)
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

func TestGetTaskByID_Integration_NotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(
		http.MethodGet,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks/{id}: %v", err)
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
```

- [ ] **Step 3: Run all integration tests**

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable" \
  go test ./internal/infrastructure/http/handler/... -v
```

Expected: all tests PASS (including existing create/update tests).

- [ ] **Step 4: Run the full test suite**

```bash
go test ./...
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/http/handler/task_integration_test.go
git commit -m "test: add integration tests for GET /tasks and GET /tasks/{id}"
```
