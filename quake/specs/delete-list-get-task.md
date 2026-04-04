# Spec: Delete Task, List Tasks, Get Task by ID

## Behavior

A caller can soft-delete a task via `DELETE /tasks/{id}` — the record is kept in the DB with `deleted_at` set. Deleted tasks are excluded from `GET /tasks` and return 404 on `GET /tasks/{id}`. `GET /tasks` returns all non-deleted tasks as a JSON array (empty array if none). `GET /tasks/{id}` returns a single task or 404 if missing or deleted.

## Interfaces

### Task 1: Migration
**`migrations/000003_soft_delete_tasks.up.sql`**
- `ALTER TABLE tasks ADD COLUMN deleted_at TIMESTAMPTZ NULL`

**`migrations/000003_soft_delete_tasks.down.sql`**
- `ALTER TABLE tasks DROP COLUMN deleted_at`

### Task 2: Domain entity
**`internal/domain/task/entity.go`**
- Add field: `DeletedAt *time.Time` (nil = active, non-nil = soft-deleted)

### Task 3: Repository + persistence
**`FindByID`** — add `AND deleted_at IS NULL` to WHERE clause; returns nil → `ErrNotFound` in app layer (no change to nil return convention)

**`FindAll`** — add `WHERE deleted_at IS NULL` to SELECT

**`Delete`** — change from `DELETE FROM tasks WHERE id = $1` to `UPDATE tasks SET deleted_at = now() WHERE id = $1`

**Scan** — update all `row.Scan` / `rows.Scan` calls to include `deleted_at`

### Task 4: Application — DeleteTask use case
**`Service.DeleteTask(ctx, id string) error`**
- Calls `FindByID`; if nil → returns `ErrNotFound`
- Calls `repo.Delete(ctx, id)`; returns any repo error

### Task 5: Application — ListTasks use case
**`Service.ListTasks(ctx) ([]task.Task, error)`**
- Calls `repo.FindAll(ctx)`; returns `([]task.Task, error)`
- Caller gets empty slice (not nil) if no tasks — handled at HTTP layer

### Task 6: Application — GetTaskByID use case
**`Service.GetTaskByID(ctx, id string) (*task.Task, error)`**
- Calls `repo.FindByID`; if nil → returns `nil, ErrNotFound`
- Returns `(*task.Task, nil)` on success

### Task 7: HTTP handlers + router
**`DELETE /tasks/{id}`** (requires `X-API-Key`)
- **204**: no body
- **404**: `{ "error": "task not found" }`

**`GET /tasks`** (requires `X-API-Key`)
- **200**: `[{ "id", "title", "description", "status", "created_at", "updated_at" }, ...]` or `[]`

**`GET /tasks/{id}`** (requires `X-API-Key`)
- **200**: `{ "id", "title", "description", "status", "created_at", "updated_at" }`
- **404**: `{ "error": "task not found" }`

**`TaskService` interface** — extend with:
```
TaskDeleter  → DeleteTask(ctx, id string) error
TaskLister   → ListTasks(ctx) ([]task.Task, error)
TaskGetter   → GetTaskByID(ctx, id string) (*task.Task, error)
```

### Task 8: Integration tests
Added to `task_integration_test.go`. Uses existing `newIntegrationServer` and `TRUNCATE TABLE tasks` cleanup pattern.

## Edge Cases

- `DELETE` on already-deleted task: `FindByID` returns nil (deleted_at IS NULL filter) → 404
- `GET /tasks` with all tasks deleted → `[]` (empty JSON array, not null)
- `GET /tasks/{id}` for soft-deleted task → 404
- Non-UUID `id` path param: `FindByID` returns nil → 404

## Constraints

- `GET /tasks` must return `[]` not `null` — initialize slice as `[]taskResponse{}` before JSON encode

## Test expectations

### Task 4: Application — DeleteTask
- `DeleteTask` with valid ID returns nil error and task is unreachable via `FindByID` afterward
- `DeleteTask` with unknown ID returns `ErrNotFound`

### Task 5: Application — ListTasks
- `ListTasks` returns all non-deleted tasks
- `ListTasks` excludes soft-deleted tasks

### Task 6: Application — GetTaskByID
- `GetTaskByID` returns task for existing ID
- `GetTaskByID` returns `ErrNotFound` for unknown ID
- `GetTaskByID` returns `ErrNotFound` for soft-deleted ID

### Task 7+8: Integration tests
- `DELETE /tasks/{id}` → 204; subsequent `GET /tasks/{id}` → 404
- `DELETE /tasks/{id}` for non-existent ID → 404 `{ "error": "task not found" }`
- `GET /tasks` with no tasks → 200 `[]`
- `GET /tasks` returns only non-deleted tasks
- `GET /tasks/{id}` for existing task → 200 with task fields
- `GET /tasks/{id}` for non-existent/deleted ID → 404 `{ "error": "task not found" }`
- All new endpoints without API key → 401

## NOT in scope

- Restore/undelete endpoint
- Pagination or filtering on `GET /tasks`
- Sorting options beyond `ORDER BY created_at DESC`
