# Spec: Create Task

## Behavior

A caller can create a task by sending a `POST /tasks` request with a valid API key. The server validates the input, persists the task with a server-generated UUID, and returns the created task. Validation failures return descriptive JSON errors. Unauthorized requests are rejected before validation.

---

## Interfaces

### Task 1: Domain entity + repository
**`task.Task` struct**
- Fields: `ID string`, `Title string`, `Description string`, `Status string`, `CreatedAt time.Time`, `UpdatedAt time.Time`
- Status constants: `StatusPending = "pending"`, `StatusInProgress = "in_progress"`, `StatusDone = "done"`

**`task.Repository` interface** — adds `Create(ctx, *Task) error` alongside existing methods

**`task.ErrDuplicateTitle`** — sentinel error returned when a task with the same title already exists

### Task 2: SQL migration
**`migrations/000002_create_tasks.up.sql`**
```sql
CREATE TABLE tasks (
  id          UUID PRIMARY KEY,
  title       VARCHAR(255) NOT NULL UNIQUE,
  description VARCHAR(2000) NOT NULL DEFAULT '',
  status      VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
**`migrations/000002_create_tasks.down.sql`** — `DROP TABLE tasks;`

### Task 3: PostgreSQL repository
**`postgres.TaskRepository` struct** — implements `task.Repository` via pgxpool

**`Create(ctx, task)`**
- Inserts one row; maps PG error `23505` → `task.ErrDuplicateTitle`
- **Output**: nil on success; error otherwise

### Task 4: Config + auth middleware
**`config.Config`** — new field `APIKey string` loaded from `API_KEY` env var (required)

**`middleware.APIKeyAuth(apiKey string) func(http.Handler) http.Handler`**
- Reads `X-API-Key` header; if missing or wrong → `401 {"error":"unauthorized"}`

### Task 5: Create task use case
**`service.CreateTask(ctx, CreateTaskCmd) (*task.Task, error)`**

`CreateTaskCmd`: `Title string`, `Description string`, `Status string`

Validation rules:
- `title` required, ≤ 255 chars → `{"field":"title","error":"..."}`
- `description` ≤ 2000 chars → `{"field":"description","error":"..."}`
- `status` defaults to `"pending"` if empty; must be one of `pending|in_progress|done`
- On `ErrDuplicateTitle` from repo → return it as-is for handler to map to 409

### Task 6: HTTP task handler
**`POST /tasks`**
- **Input**: `Content-Type: application/json`, body `{"title":"...","description":"...","status":"..."}`
- **Output (201)**: full task JSON `{"id":"...","title":"...","description":"...","status":"...","created_at":"...","updated_at":"..."}`
- **Output (422)**: `{"errors":[{"field":"title","error":"title is required"}]}`
- **Output (409)**: `{"error":"a task with this title already exists"}`
- **Output (401)**: `{"error":"unauthorized"}`
- **Output (400)**: `{"error":"invalid JSON body"}` on malformed JSON

### Task 7: Wire + integration test
**`router.Register`** — new signature accepts `taskService` and `apiKey`; mounts auth middleware scoped to `/tasks`

**Integration test** (`task_integration_test.go`) — uses `TEST_DATABASE_URL`; skipped if env var unset; runs migration up before suite, truncates table between tests

---

## Edge Cases

- `title` is whitespace-only — treated as empty, fails required validation
- `status` casing mismatch (e.g. `"Pending"`) — rejected with validation error
- `description` absent from JSON body — treated as empty string, stored as `''`
- Duplicate title submitted twice — second returns 409, first row is unchanged
- `API_KEY` not set at startup — server fails to start (required config)
- Missing `Content-Type` header — still parsed; chi doesn't enforce it

---

## Constraints

- Domain layer (`internal/domain/`) must import only stdlib
- UUIDs generated server-side with `"github.com/google/uuid"` (add to go.mod)
- Integration tests gated by `TEST_DATABASE_URL` env var; not run with `go test ./... -short`

---

## Test expectations

### Task 1: Domain entity + repository
- `task.Task` compiles with `Status` field and constants accessible

### Task 4: Config + auth middleware
- Middleware returns 401 when `X-API-Key` header is absent
- Middleware returns 401 when key is wrong
- Middleware calls next handler when key matches

### Task 5: Create task use case
- Returns created task with generated UUID and `pending` status when input is valid
- Returns validation error when title is empty
- Returns validation error when title exceeds 255 chars
- Returns validation error when description exceeds 2000 chars
- Returns validation error when status is an unknown value
- Returns `ErrDuplicateTitle` when repo returns it

### Task 6: HTTP task handler
- `POST /tasks` with valid body + correct key → 201 with task JSON
- `POST /tasks` with missing title → 422 with field error
- `POST /tasks` with invalid status → 422 with field error
- `POST /tasks` without API key → 401
- `POST /tasks` with malformed JSON → 400

### Task 7: Integration test
- Creates task successfully → 201, task persisted in DB
- Duplicate title → 409
- Missing title → 422
- Wrong API key → 401

---

## NOT in scope

- GET/PUT/DELETE task endpoints
- Pagination or filtering
- JWT or any auth beyond API key header
- Rate limiting
