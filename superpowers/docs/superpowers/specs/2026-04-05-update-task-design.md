# Design: Update Task Feature

## Overview

Implements `PATCH /tasks/{id}` — a partial-update endpoint for the task management API. Accepts a JSON body with any combination of `title`, `description`, and `status`. Fields absent from the body are left unchanged. Returns the full updated task on success, 404 if the task does not exist, 400 on validation errors. Protected by the existing API key middleware.

---

## File Map

Changes relative to the create-task implementation:

```
internal/
├── domain/
│   └── task/
│       ├── entity.go          # unchanged
│       └── repository.go      # add GetByID, Update methods; add ErrNotFound sentinel
├── application/
│   └── task/
│       ├── service.go         # add UpdateTaskInput type and UpdateTask method
│       └── service_test.go    # unchanged (unit tests for UpdateTask live here too)
└── infrastructure/
    ├── persistence/postgres/
    │   └── task_repository.go # add GetByID, Update methods
    └── http/
        ├── handler/
        │   ├── task.go                  # add Update handler method
        │   └── task_integration_test.go # add 4 new integration test functions
        └── router.go                    # mount PATCH /tasks/{id}
```

No new migrations — the existing `tasks` table schema supports updates without changes.

---

## Layer Rules

Unchanged from foundation:

- `internal/domain/` — stdlib only
- `internal/application/` — may import `internal/domain/` only
- `internal/infrastructure/` — may import `internal/application/` and `internal/domain/`

---

## Domain Layer

### `internal/domain/task/repository.go`

```go
var ErrNotFound = errors.New("task not found")

type Repository interface {
    Create(ctx context.Context, task Task) (Task, error)
    GetByID(ctx context.Context, id string) (Task, error)
    Update(ctx context.Context, task Task) (Task, error)
}
```

`GetByID` returns `ErrNotFound` when no row matches the given ID. This sentinel is defined in the domain package so the application layer can check it without importing pgx.

---

## Application Layer

### `internal/application/task/service.go`

New type added alongside the existing `CreateTaskInput`:

```go
type UpdateTaskInput struct {
    ID          string
    Title       *string  // nil = not provided, do not update
    Description *string
    Status      *string
}
```

New method on `Service`:

```go
func (s *Service) UpdateTask(ctx context.Context, in UpdateTaskInput) (domain.Task, error)
```

**`UpdateTask` logic:**

1. Validate non-nil fields using the same rules as `CreateTask`:
   - `Title`: if non-nil, must be non-empty and ≤255 characters
   - `Description`: if non-nil, must be ≤2000 characters
   - `Status`: if non-nil, must be one of `pending`, `in_progress`, `done`
2. Return `*ValidationError` if any field fails — same type, same field-name keys
3. Call `repo.GetByID(ctx, in.ID)`:
   - If it returns `domain.ErrNotFound`, return `domain.ErrNotFound` unchanged
   - On any other error, return it unchanged
4. Merge non-nil fields onto the fetched task; set `task.UpdatedAt = time.Now().UTC()`
5. Call `repo.Update(ctx, task)` and return the result

If all three fields are `nil` (empty body `{}`), validation passes and the task is fetched, `UpdatedAt` is refreshed, and it is re-persisted with no field changes. This is intentional — the spec treats an empty PATCH as a valid no-op that still refreshes `updated_at`.

The `validStatuses` map remains a package-level variable shared by both `CreateTask` and `UpdateTask`.

---

## Infrastructure Layer

### `internal/infrastructure/persistence/postgres/task_repository.go`

Two new methods:

**`GetByID`**

```sql
SELECT id, title, description, status, created_at, updated_at
FROM tasks
WHERE id = $1
```

If `pgx.ErrNoRows`, return `domain.ErrNotFound`. Otherwise scan and return the task.

**`Update`**

The service merges fields before calling `Update`, so the repository receives a fully populated task:

```sql
UPDATE tasks
SET title = $1, description = $2, status = $3, updated_at = $4
WHERE id = $5
RETURNING id, title, description, status, created_at, updated_at
```

Scans and returns the updated row. Returns the raw pgx error on failure.

### `internal/infrastructure/http/handler/task.go`

New request struct for PATCH (pointer fields so absent JSON keys decode as `nil`):

```go
type updateTaskRequest struct {
    Title       *string `json:"title"`
    Description *string `json:"description"`
    Status      *string `json:"status"`
}
```

New method on `TaskHandler`:

```go
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request)
```

**`Update` logic:**

1. Extract `id` from the URL: `chi.URLParam(r, "id")`
2. Decode JSON body into `updateTaskRequest`; on failure → `400 {"error": "invalid request body"}`
3. Call `svc.UpdateTask(ctx, UpdateTaskInput{ID: id, Title: req.Title, ...})`
4. On `domain.ErrNotFound` → `404 {"error": "task not found"}`
5. On `*ValidationError` → `400 {"errors": {"field": "message", ...}}`
6. On any other error → `500 {"error": "internal server error"}`
7. On success → `200` with the same `taskResponse` shape used by `Create`

### `internal/infrastructure/http/router.go`

```go
r.With(middleware.APIKey(apiKey)).Patch("/tasks/{id}", taskHandler.Update)
```

---

## Error Handling Summary

| Scenario | HTTP | Body |
|---|---|---|
| Missing or wrong API key | 401 | `{"error": "unauthorized"}` |
| Malformed JSON | 400 | `{"error": "invalid request body"}` |
| Validation failure | 400 | `{"errors": {"title": "...", ...}}` |
| Task not found | 404 | `{"error": "task not found"}` |
| DB / internal error | 500 | `{"error": "internal server error"}` |
| Success | 200 | Full task JSON |

**Success response shape (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Updated title",
  "description": "Updated description",
  "status": "done",
  "created_at": "2026-04-05T10:00:00Z",
  "updated_at": "2026-04-05T11:30:00Z"
}
```

---

## Testing

### Unit: `internal/application/task/service_test.go`

Mock repository gains `GetByID` and `Update` stubs. New test cases:

- Valid full update → all fields changed, `updated_at` refreshed
- Partial update (status only) → title and description unchanged
- Empty title → `*ValidationError` with `errors["title"]`
- Title over 255 chars → `*ValidationError` with `errors["title"]`
- Description over 2000 chars → `*ValidationError` with `errors["description"]`
- Invalid status → `*ValidationError` with `errors["status"]`
- Task not found → returns `domain.ErrNotFound`

### Integration: `internal/infrastructure/http/handler/task_integration_test.go`

Four new test functions (all skip if `TEST_DATABASE_URL` is not set):

1. **`TestUpdateTask_Integration_Success`** — create a task, PATCH all three fields, assert 200 and new values returned
2. **`TestUpdateTask_Integration_PartialUpdate`** — create a task, PATCH only `{"status":"done"}`, assert status changed and title/description unchanged
3. **`TestUpdateTask_Integration_NotFound`** — PATCH a random UUID, assert 404 with `{"error":"task not found"}`
4. **`TestUpdateTask_Integration_ValidationError`** — create a task, PATCH with `{"title":""}`, assert 400 with `errors["title"]`

Each test with DB writes uses `t.Cleanup` to `TRUNCATE tasks`.
