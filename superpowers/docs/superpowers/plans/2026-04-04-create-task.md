# Create Task Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `POST /tasks` with full DDD layering, API key auth, input validation, PostgreSQL persistence, and integration tests.

**Architecture:** Thin handler parses JSON and maps errors to HTTP codes; application service owns validation, UUID generation, and status defaulting; domain layer is pure data structs + repository interface; PostgreSQL repository implements the interface in infrastructure. Auth is a chi middleware checking `Authorization: Bearer <key>` against `cfg.APIKey`.

**Tech Stack:** Go 1.25, chi v5, pgx v5 (pgxpool), google/uuid, crypto/subtle (stdlib), httptest (stdlib).

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `go.mod` / `go.sum` | Modify | Add `github.com/google/uuid` |
| `.env.example` | Modify | Add `API_KEY` example |
| `internal/infrastructure/config/config.go` | Modify | Add `APIKey` field, validate it |
| `internal/infrastructure/config/config_test.go` | Modify | Add API key tests; fix existing tests to set `API_KEY` |
| `migrations/000002_create_tasks.up.sql` | Create | `CREATE TABLE tasks` DDL |
| `migrations/000002_create_tasks.down.sql` | Create | `DROP TABLE tasks` |
| `internal/domain/task/entity.go` | Create | `Task` struct |
| `internal/domain/task/repository.go` | Create | `Repository` interface |
| `internal/application/task/service.go` | Create | `Service`, `CreateTaskInput`, `ValidationError` |
| `internal/application/task/service_test.go` | Create | Unit tests for validation and creation logic |
| `internal/infrastructure/persistence/postgres/task_repository.go` | Create | pgx `INSERT … RETURNING` implementation |
| `internal/infrastructure/http/middleware/auth.go` | Create | `APIKey` middleware (constant-time compare) |
| `internal/infrastructure/http/middleware/auth_test.go` | Create | Unit tests for middleware |
| `internal/infrastructure/http/handler/task.go` | Create | `TaskHandler.Create` — decode, delegate, respond |
| `internal/infrastructure/http/handler/task_integration_test.go` | Create | `TestMain` + 4 integration test cases |
| `internal/infrastructure/http/router.go` | Modify | Add `apiKey string` param, mount `POST /tasks` |
| `cmd/api/main.go` | Modify | Pass `cfg.APIKey` to `Register` |
| `cmd/api/main_test.go` | Modify | Pass `"test-api-key"` to `Register` |

---

## Task 1: Dependencies & Config (TDD)

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `.env.example`
- Modify: `internal/infrastructure/config/config_test.go`
- Modify: `internal/infrastructure/config/config.go`

- [ ] **Step 1: Write failing config tests**

The new test `TestLoad_ErrorOnEmptyAPIKey` will fail until `config.go` is updated. The existing success tests (`TestLoad_DefaultPort`, `TestLoad_DefaultEnv`) must also set `API_KEY` since it will become a required field.

Save the complete file to `internal/infrastructure/config/config_test.go`:

```go
package config_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/config"
)

func TestLoad_ErrorOnEmptyDBUrl(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Setenv("API_KEY", "test-key")
	defer os.Unsetenv("API_KEY")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is empty, got nil")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("API_KEY", "test-key")
	os.Unsetenv("PORT")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("API_KEY")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port=8080, got %s", cfg.Port)
	}
}

func TestLoad_DefaultEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("API_KEY", "test-key")
	os.Unsetenv("ENV")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("API_KEY")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env=development, got %s", cfg.Env)
	}
}

func TestLoad_ErrorOnEmptyAPIKey(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("API_KEY")
	defer os.Unsetenv("DATABASE_URL")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when API_KEY is empty, got nil")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/infrastructure/config/... -v
```

Expected: `TestLoad_ErrorOnEmptyAPIKey` FAILS (`config.Load` doesn't check `API_KEY` yet). `TestLoad_DefaultPort` and `TestLoad_DefaultEnv` may also fail if `API_KEY` happens to be unset in the environment.

- [ ] **Step 3: Add uuid dependency**

```bash
go get github.com/google/uuid
go mod tidy
```

Expected: `go.mod` now includes `github.com/google/uuid` and `go.sum` is updated.

- [ ] **Step 4: Update config.go**

Save the complete file to `internal/infrastructure/config/config.go`:

```go
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl  string
	Port   string
	Env    string
	APIKey string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env") // ignore error: .env may not exist when env vars are injected externally

	cfg := Config{
		DBUrl:  os.Getenv("DATABASE_URL"),
		Port:   os.Getenv("PORT"),
		Env:    os.Getenv("ENV"),
		APIKey: os.Getenv("API_KEY"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.DBUrl == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("API_KEY is required")
	}

	return cfg, nil
}
```

- [ ] **Step 5: Run tests — verify they pass**

```bash
go test ./internal/infrastructure/config/... -v
```

Expected: all 4 tests PASS.

- [ ] **Step 6: Update .env.example**

Save the complete file to `.env.example`:

```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable
PORT=8080
ENV=development
API_KEY=your-secret-api-key

# Used by integration tests (docker-compose.test.yml runs Postgres on 5433)
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable
```

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum .env.example internal/infrastructure/config/
git commit -m "feat: add API_KEY config field and uuid dependency"
```

---

## Task 2: Migration

**Files:**
- Create: `migrations/000002_create_tasks.up.sql`
- Create: `migrations/000002_create_tasks.down.sql`

- [ ] **Step 1: Create up migration**

Save to `migrations/000002_create_tasks.up.sql`:

```sql
CREATE TABLE tasks (
    id          TEXT        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);
```

- [ ] **Step 2: Create down migration**

Save to `migrations/000002_create_tasks.down.sql`:

```sql
DROP TABLE IF EXISTS tasks;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000002_create_tasks.up.sql migrations/000002_create_tasks.down.sql
git commit -m "feat: add tasks table migration"
```

---

## Task 3: Domain Layer

**Files:**
- Create: `internal/domain/task/entity.go`
- Create: `internal/domain/task/repository.go`

- [ ] **Step 1: Create entity.go**

Save to `internal/domain/task/entity.go`:

```go
package task

import "time"

type Task struct {
	ID          string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Create repository.go**

Save to `internal/domain/task/repository.go`:

```go
package task

import "context"

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/domain/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/task/
git commit -m "feat: add task domain entity and repository interface"
```

---

## Task 4: Application Layer — Service (TDD)

**Files:**
- Create: `internal/application/task/service_test.go`
- Create: `internal/application/task/service.go`

- [ ] **Step 1: Write failing service tests**

Save to `internal/application/task/service_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/application/task/... -v
```

Expected: FAIL — `apptask.NewService` undefined.

- [ ] **Step 3: Implement service.go**

Save to `internal/application/task/service.go`:

```go
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/application/task/... -v
```

Expected: all 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/task/
git commit -m "feat: add create task use case with validation"
```

---

## Task 5: PostgreSQL Repository

**Files:**
- Create: `internal/infrastructure/persistence/postgres/task_repository.go`

- [ ] **Step 1: Create task_repository.go**

Save to `internal/infrastructure/persistence/postgres/task_repository.go`:

```go
package postgres

import (
	"context"
	"fmt"

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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/persistence/postgres/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/persistence/postgres/task_repository.go
git commit -m "feat: add postgres task repository"
```

---

## Task 6: Auth Middleware (TDD)

**Files:**
- Create: `internal/infrastructure/http/middleware/auth_test.go`
- Create: `internal/infrastructure/http/middleware/auth.go`

- [ ] **Step 1: Write failing middleware tests**

Save to `internal/infrastructure/http/middleware/auth_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/middleware"
)

func TestAPIKey_AllowsValidKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.APIKey("secret")(next)

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIKey_RejectsMissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.APIKey("secret")(next)

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIKey_RejectsWrongKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.APIKey("secret")(next)

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/infrastructure/http/middleware/... -v
```

Expected: FAIL — `middleware.APIKey` undefined.

- [ ] **Step 3: Implement auth.go**

Save to `internal/infrastructure/http/middleware/auth.go`:

```go
package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// APIKey returns a middleware that requires a valid Bearer token in the Authorization header.
// Uses constant-time comparison to prevent timing attacks.
func APIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			key := strings.TrimPrefix(auth, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}) //nolint:errcheck
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/infrastructure/http/middleware/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/http/middleware/
git commit -m "feat: add API key auth middleware"
```

---

## Task 7: Task HTTP Handler

**Files:**
- Create: `internal/infrastructure/http/handler/task.go`

- [ ] **Step 1: Create task.go**

Save to `internal/infrastructure/http/handler/task.go`:

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
)

type taskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/http/handler/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/handler/task.go
git commit -m "feat: add task HTTP handler"
```

---

## Task 8: Router & Wiring

**Files:**
- Modify: `internal/infrastructure/http/router.go`
- Modify: `cmd/api/main.go`
- Modify: `cmd/api/main_test.go`

- [ ] **Step 1: Update router.go**

Save the complete file to `internal/infrastructure/http/router.go`:

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

	return r
}
```

- [ ] **Step 2: Update main.go**

Save the complete file to `cmd/api/main.go`:

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/config"
	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "port", cfg.Port, "env", cfg.Env)

	pool, err := postgres.Connect(cfg.DBUrl)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	router := infrahttp.Register(pool, cfg.APIKey)
	srv := infrahttp.New(router, cfg.Port)
	slog.Info("server starting", "port", cfg.Port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	infrahttp.GracefulShutdown(srv)
}
```

- [ ] **Step 3: Update main_test.go**

Save the complete file to `cmd/api/main_test.go`:

```go
package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func TestSmokeTest_HealthEndpoint(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping smoke test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	router := infrahttp.Register(pool, "test-api-key")
	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 4: Build everything**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run all unit tests**

```bash
go test ./internal/... -v
```

Expected: all unit tests PASS (config, db, health handler, service, middleware).

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/http/router.go cmd/api/main.go cmd/api/main_test.go
git commit -m "feat: wire task handler into router with API key auth"
```

---

## Task 9: Integration Test

**Files:**
- Create: `internal/infrastructure/http/handler/task_integration_test.go`

- [ ] **Step 1: Create task_integration_test.go**

This file defines `TestMain` for the entire `handler_test` package (including the health handler tests). When `TEST_DATABASE_URL` is not set, `TestMain` still calls `m.Run()` so unit tests continue to pass — only integration tests skip.

Save to `internal/infrastructure/http/handler/task_integration_test.go`:

```go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

const integTestAPIKey = "test-api-key"

const createTasksSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
)`

var (
	integPool   *pgxpool.Pool
	integServer *httptest.Server
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// No DB — unit tests still run; integration tests skip individually.
		os.Exit(m.Run())
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		fmt.Printf("connect test db: %v\n", err)
		os.Exit(1)
	}

	if _, err = pool.Exec(context.Background(), createTasksSQL); err != nil {
		fmt.Printf("create tasks table: %v\n", err)
		pool.Close()
		os.Exit(1)
	}

	router := infrahttp.Register(pool, integTestAPIKey)
	srv := httptest.NewServer(router)

	integPool = pool
	integServer = srv

	code := m.Run()

	pool.Exec(context.Background(), "DROP TABLE IF EXISTS tasks") //nolint:errcheck
	srv.Close()
	pool.Close()

	os.Exit(code)
}

func TestCreateTask_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	body := `{"title":"Buy groceries","description":"Milk and eggs"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
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
	if result["title"] != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %v", result["title"])
	}
	if result["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", result["status"])
	}
	if result["created_at"] == "" || result["created_at"] == nil {
		t.Error("expected non-empty created_at in response")
	}
}

func TestCreateTask_Integration_ValidationError(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	body := `{"title":""}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
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

func TestCreateTask_Integration_DuplicateSubmission(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	body := `{"title":"Same title","description":"Same description"}`

	send := func() string {
		req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /tasks: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		id, _ := result["id"].(string)
		return id
	}

	id1 := send()
	id2 := send()

	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty IDs from both requests")
	}
	if id1 == id2 {
		t.Errorf("expected different IDs for duplicate submission, both got %q", id1)
	}
}

func TestCreateTask_Integration_Unauthorized(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	body := `{"title":"Buy groceries"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/http/handler/...
```

Expected: no errors.

- [ ] **Step 3: Run unit tests (no DB)**

```bash
go test ./internal/infrastructure/http/handler/... -v
```

Expected: `TestHealthHandler_OK` and `TestHealthHandler_DBUnreachable` PASS. All `TestCreateTask_Integration_*` tests SKIP.

- [ ] **Step 4: Start test DB**

```bash
make docker-test-up
```

Expected: test Postgres container starts on port 5433.

- [ ] **Step 5: Run migration against test DB (verifies migration file is valid)**

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable make migrate-up
```

Expected: output includes `1/u 000001_init` and `2/u 000002_create_tasks` with no errors.

- [ ] **Step 6: Run integration tests**

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  go test ./internal/infrastructure/http/handler/... -v -run "TestCreateTask_Integration"
```

Expected: all 4 `TestCreateTask_Integration_*` tests PASS.

- [ ] **Step 7: Run the full test suite**

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  go test ./...
```

Expected: all tests PASS — unit tests (config, db, health, service, middleware) and integration tests (smoke test + task handler).

- [ ] **Step 8: Commit**

```bash
git add internal/infrastructure/http/handler/task_integration_test.go
git commit -m "test: add integration tests for POST /tasks"
```
