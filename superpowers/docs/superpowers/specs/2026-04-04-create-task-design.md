# Design: Create Task Feature

## Overview

Implements `POST /tasks` — the first business endpoint of the task management API. Accepts a JSON body with `title`, `description`, and optional `status`, persists the task to PostgreSQL, and returns the created task with a server-generated UUID and timestamps. Protected by an API key middleware.

---

## File Map

New files added to the existing DDD foundation:

```
internal/
├── domain/
│   └── task/
│       ├── entity.go                    # Task struct (plain data, no methods)
│       └── repository.go                # Repository interface
├── application/
│   └── task/
│       ├── service.go                   # CreateTask use case
│       └── service_test.go              # Unit tests with mock repository
└── infrastructure/
    ├── persistence/postgres/
    │   └── task_repository.go           # PostgreSQL implementation
    └── http/
        ├── middleware/
        │   └── auth.go                  # API key middleware
        ├── handler/
        │   ├── task.go                  # POST /tasks handler
        │   └── task_integration_test.go # Integration tests
        └── router.go                    # Updated: mounts POST /tasks with auth

migrations/
├── 000002_create_tasks.up.sql
└── 000002_create_tasks.down.sql
```

`internal/infrastructure/config/config.go` gains one field: `APIKey string` (read from `API_KEY` env var, required).

---

## Layer Rules

Unchanged from foundation. These rules govern **internal package** imports:

- `internal/domain/` — no external imports; stdlib only
- `internal/application/` — may import `internal/domain/` only (external packages like `uuid` are permitted)
- `internal/infrastructure/` — may import `internal/application/` and `internal/domain/`

---

## Domain Layer

### `internal/domain/task/entity.go`

```go
type Task struct {
    ID          string
    Title       string
    Description string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Plain data struct, no methods. The domain does not generate UUIDs or enforce invariants — that responsibility belongs to the use case.

### `internal/domain/task/repository.go`

```go
type Repository interface {
    Create(ctx context.Context, task Task) (Task, error)
}
```

---

## Application Layer

### `internal/application/task/service.go`

```go
type CreateTaskInput struct {
    Title       string
    Description string
    Status      string
}

type ValidationError struct {
    Fields map[string]string // field name → error message
}

func (e *ValidationError) Error() string

type Service struct {
    repo domain.Repository
}

func NewService(repo domain.Repository) *Service
func (s *Service) CreateTask(ctx context.Context, in CreateTaskInput) (domain.Task, error)
```

**`CreateTask` logic:**

1. Validate `Title`: required, max 255 characters
2. Validate `Description`: max 2000 characters (empty is allowed)
3. Validate `Status`: if non-empty, must be one of `pending`, `in_progress`, `done`
4. Default `Status` to `"pending"` if empty
5. Generate UUID for `ID` using `github.com/google/uuid`
6. Set `CreatedAt` and `UpdatedAt` to `time.Now().UTC()`
7. Call `repo.Create(ctx, task)` and return the result

Returns `*ValidationError` if any field fails validation. Returns the repository error directly for DB failures (no wrapping at this layer).

---

## Infrastructure Layer

### Migration: `000002_create_tasks.up.sql`

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

No unique constraint on `title` — duplicate titles are allowed.

### `internal/infrastructure/persistence/postgres/task_repository.go`

Implements `domain/task.Repository`.

```go
type TaskRepository struct {
    pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository
func (r *TaskRepository) Create(ctx context.Context, task domain.Task) (domain.Task, error)
```

`Create` executes:
```sql
INSERT INTO tasks (id, title, description, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, title, description, status, created_at, updated_at
```

Returns the scanned row as a `domain.Task`. Returns the raw pgx error on failure — no wrapping.

### Auth Middleware: `internal/infrastructure/http/middleware/auth.go`

Reads the `Authorization` header. Expects `Bearer <key>`. Compares the key to `cfg.APIKey` using `subtle.ConstantTimeCompare` to prevent timing attacks. Returns `401 {"error": "unauthorized"}` if missing or incorrect.

```go
func APIKey(apiKey string) func(http.Handler) http.Handler
```

### HTTP Handler: `internal/infrastructure/http/handler/task.go`

```go
type TaskHandler struct {
    svc *application.Service
}

func NewTaskHandler(svc *application.Service) *TaskHandler
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request)
```

**`Create` logic:**

1. Decode JSON body into request struct; on failure → `400 {"error": "invalid request body"}`
2. Call `svc.CreateTask(ctx, input)`
3. On `*ValidationError` → `400 {"errors": {"field": "message", ...}}`
4. On any other error → `500 {"error": "internal server error"}`
5. On success → `201` with task JSON

**Response shape (201):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Buy groceries",
  "description": "Milk, eggs, bread",
  "status": "pending",
  "created_at": "2026-04-04T10:00:00Z",
  "updated_at": "2026-04-04T10:00:00Z"
}
```

### Router: `internal/infrastructure/http/router.go`

Updated to accept `apiKey string` and mount the task route:

```go
func Register(pool *pgxpool.Pool, apiKey string) nethttp.Handler
```

```
POST /tasks  →  middleware.APIKey(apiKey)  →  taskHandler.Create
```

`main.go` passes `cfg.APIKey` to `Register`.

---

## Config

`config.go` adds:

```go
APIKey string  // read from API_KEY env var, required
```

Returns error if `API_KEY` is empty.

`.env.example` gains:
```
API_KEY=your-secret-api-key
```

---

## Error Handling Summary

| Scenario | HTTP | Body |
|---|---|---|
| Missing or wrong API key | 401 | `{"error": "unauthorized"}` |
| Malformed JSON | 400 | `{"error": "invalid request body"}` |
| Validation failure | 400 | `{"errors": {"title": "...", ...}}` |
| DB / internal error | 500 | `{"error": "internal server error"}` |
| Success | 201 | Task JSON |

---

## Testing

### Unit: `internal/application/task/service_test.go`

Mock repository defined inline (no external library). Tests:

- Valid input → task created with correct fields, UUID set, status defaults to `"pending"`
- Empty title → `*ValidationError` with `errors.title`
- Title over 255 chars → `*ValidationError` with `errors.title`
- Description over 2000 chars → `*ValidationError` with `errors.description`
- Invalid status value → `*ValidationError` with `errors.status`
- Explicit valid status → task created with that status

### Integration: `internal/infrastructure/http/handler/task_integration_test.go`

Uses `httptest.NewServer` against a real test DB (skipped if `TEST_DATABASE_URL` not set). A `TestMain` function connects to the DB, executes the `CREATE TABLE tasks` DDL directly via pgx (no CLI dependency), runs the tests, then executes `DROP TABLE tasks` to clean up.

Three test cases:

1. **Success** — `POST /tasks` with valid body + valid API key → 201, response has UUID `id`, `status` is `"pending"`, `created_at` is non-zero
2. **Validation error** — empty `title` → 400, `errors.title` present in response
3. **Duplicate submission** — same body twice → two 201 responses with different `id` values

---

## Dependencies Added

- `github.com/google/uuid` — UUID v4 generation in the use case

---

## Not in Scope

- `GET`, `PUT`, `DELETE` /tasks endpoints
- Pagination, filtering, sorting
- Soft delete
- Rate limiting
