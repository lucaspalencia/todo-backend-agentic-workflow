# Update Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `PATCH /tasks/{id}` for partial task updates, returning the full updated task on success.

**Architecture:** Pointer fields in `UpdateTaskInput` represent optional fields — `nil` means "not sent, leave unchanged". The service fetches the task, validates and merges non-nil fields, refreshes `updated_at`, then writes back. A `domain.ErrNotFound` sentinel flows from repository → service → handler to produce a 404.

**Tech Stack:** Go, chi router, pgx/v5, PostgreSQL — same as create-task.

**All commands run from:** `/Users/lucas.palencia/dev/todo-backend-agentic-workflow/superpowers`

---

### Task 1: Extend the domain Repository interface and add ErrNotFound

**Files:**
- Modify: `internal/domain/task/repository.go`

- [ ] **Step 1: Replace the contents of `internal/domain/task/repository.go`**

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
}
```

- [ ] **Step 2: Verify the domain package still compiles**

```bash
go build ./internal/domain/...
```

Expected: no output (success). The application and infrastructure layers will fail to compile until the mock and postgres repository are updated — that is expected and intentional.

---

### Task 2: Update the mock repository and write failing UpdateTask unit tests

**Files:**
- Modify: `internal/application/task/service_test.go`

- [ ] **Step 1: Replace `mockRepo` to satisfy the extended interface and add UpdateTask test cases**

Replace the entire `internal/application/task/service_test.go` with the following (all existing `CreateTask` tests are preserved, new tests are added at the bottom):

```go
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
```

- [ ] **Step 2: Run the tests to confirm they fail to compile (UpdateTask not yet defined)**

```bash
go test ./internal/application/task/...
```

Expected: compile error mentioning `apptask.UpdateTaskInput` undefined. This confirms the tests are wired correctly.

---

### Task 3: Implement UpdateTask in the application service

**Files:**
- Modify: `internal/application/task/service.go`

- [ ] **Step 1: Add `UpdateTaskInput` and `UpdateTask` to the service**

Append the following to `internal/application/task/service.go` (after the existing `CreateTask` function):

```go
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
```

- [ ] **Step 2: Run unit tests and confirm they pass**

```bash
go test ./internal/application/task/... -v
```

Expected: all tests PASS including the new `TestUpdateTask_*` cases.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/task/repository.go \
        internal/application/task/service.go \
        internal/application/task/service_test.go
git commit -m "feat: add UpdateTask use case and extend Repository interface"
```

---

### Task 4: Implement GetByID and Update in the PostgreSQL repository

**Files:**
- Modify: `internal/infrastructure/persistence/postgres/task_repository.go`

- [ ] **Step 1: Replace the contents of `task_repository.go` with the extended implementation**

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

// GetByID fetches a single task by its ID. Returns domain.ErrNotFound if no row matches.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE id = $1`,
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
```

- [ ] **Step 2: Confirm the project compiles**

```bash
go build ./...
```

Expected: no output (success).

---

### Task 5: Add the Update HTTP handler and wire the PATCH route

**Files:**
- Modify: `internal/infrastructure/http/handler/task.go`
- Modify: `internal/infrastructure/http/router.go`

- [ ] **Step 1: Replace the contents of `internal/infrastructure/http/handler/task.go`**

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	domaintask "github.com/lucaspalencia/superpowers/internal/domain/task"
)

type taskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type taskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TaskHandler handles HTTP requests for task operations.
type TaskHandler struct {
	svc *apptask.Service
}

// NewTaskHandler constructs a TaskHandler with the given service.
func NewTaskHandler(svc *apptask.Service) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create handles POST /tasks. Decodes JSON, delegates to service, returns 201 on success.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	task, err := h.svc.CreateTask(r.Context(), apptask.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		var ve *apptask.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": ve.Fields})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}

// Update handles PATCH /tasks/{id}. Applies partial updates and returns 200 with the full task.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	task, err := h.svc.UpdateTask(r.Context(), apptask.UpdateTaskInput{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		var ve *apptask.ValidationError
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

	writeJSON(w, http.StatusOK, taskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}
```

- [ ] **Step 2: Add the PATCH route to `internal/infrastructure/http/router.go`**

Replace the contents of `internal/infrastructure/http/router.go`:

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
	r.With(middleware.APIKey(apiKey)).Patch("/tasks/{id}", taskHandler.Update)

	return r
}
```

- [ ] **Step 3: Confirm the full project compiles**

```bash
go build ./...
```

Expected: no output (success).

---

### Task 6: Write integration tests and verify end-to-end

**Files:**
- Modify: `internal/infrastructure/http/handler/task_integration_test.go`

- [ ] **Step 1: Start the test database**

```bash
make docker-test-up
```

Expected: postgres container starts on port 5433.

- [ ] **Step 2: Append the four UpdateTask integration tests to `task_integration_test.go`**

Add the following functions at the end of `internal/infrastructure/http/handler/task_integration_test.go`:

```go
func TestUpdateTask_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task to update
	createBody := `{"title":"Original title","description":"Original description","status":"pending"}`
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(createBody))
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

	// Update all fields
	updateBody := `{"title":"Updated title","description":"Updated description","status":"done"}`
	req, _ := http.NewRequest(http.MethodPatch, integServer.URL+"/tasks/"+id, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /tasks/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if result["title"] != "Updated title" {
		t.Errorf("expected title 'Updated title', got %v", result["title"])
	}
	if result["description"] != "Updated description" {
		t.Errorf("expected description 'Updated description', got %v", result["description"])
	}
	if result["status"] != "done" {
		t.Errorf("expected status 'done', got %v", result["status"])
	}
	if result["updated_at"] == created["updated_at"] {
		t.Error("expected updated_at to be refreshed")
	}
}

func TestUpdateTask_Integration_PartialUpdate(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task to update
	createBody := `{"title":"Keep this title","description":"Keep this description","status":"pending"}`
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(createBody))
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

	// Update only status
	updateBody := `{"status":"done"}`
	req, _ := http.NewRequest(http.MethodPatch, integServer.URL+"/tasks/"+id, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /tasks/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if result["title"] != "Keep this title" {
		t.Errorf("expected title unchanged 'Keep this title', got %v", result["title"])
	}
	if result["description"] != "Keep this description" {
		t.Errorf("expected description unchanged 'Keep this description', got %v", result["description"])
	}
	if result["status"] != "done" {
		t.Errorf("expected status 'done', got %v", result["status"])
	}
}

func TestUpdateTask_Integration_NotFound(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	req, _ := http.NewRequest(
		http.MethodPatch,
		integServer.URL+"/tasks/00000000-0000-0000-0000-000000000000",
		bytes.NewBufferString(`{"status":"done"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /tasks: %v", err)
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

func TestUpdateTask_Integration_ValidationError(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	// Create a task first
	createBody := `{"title":"Valid task"}`
	createReq, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(createBody))
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

	// Attempt to set title to empty string
	req, _ := http.NewRequest(http.MethodPatch, integServer.URL+"/tasks/"+id, bytes.NewBufferString(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /tasks/%s: %v", id, err)
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
	if _, ok := errs["title"]; !ok {
		t.Error("expected errors[\"title\"] to be set")
	}
}
```

- [ ] **Step 3: Run integration tests against the test database**

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable" \
  go test ./internal/infrastructure/http/handler/... -v -run TestUpdateTask
```

Expected: all four `TestUpdateTask_Integration_*` tests PASS.

- [ ] **Step 4: Run the full test suite to confirm nothing regressed**

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable" \
  go test ./...
```

Expected: all tests PASS (integration tests run, unit tests pass, no compile errors).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/persistence/postgres/task_repository.go \
        internal/infrastructure/http/handler/task.go \
        internal/infrastructure/http/handler/task_integration_test.go \
        internal/infrastructure/http/router.go
git commit -m "feat: implement PATCH /tasks/{id} with partial update support"
```
