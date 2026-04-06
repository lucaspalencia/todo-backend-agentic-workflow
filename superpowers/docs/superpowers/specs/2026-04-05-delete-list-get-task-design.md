# Delete, List, and Get Task — Design Spec

**Date:** 2026-04-05
**Status:** Approved

---

## Overview

Add three new endpoints to the task API:

| Method   | Path           | Description                              |
|----------|----------------|------------------------------------------|
| DELETE   | /tasks/{id}    | Soft-delete a task                       |
| GET      | /tasks         | List all non-deleted tasks, newest first |
| GET      | /tasks/{id}    | Fetch a single non-deleted task by ID    |

All endpoints are protected by the existing API key middleware.

---

## Database Migration

A new migration (`000003_soft_delete_tasks`) adds a nullable `deleted_at TIMESTAMPTZ` column to the `tasks` table.

```sql
-- up
ALTER TABLE tasks ADD COLUMN deleted_at TIMESTAMPTZ;

-- down
ALTER TABLE tasks DROP COLUMN deleted_at;
```

Existing rows are unaffected — `NULL` means active. No data migration needed.

---

## Domain Layer

`domain.Task` is unchanged. Soft-delete is an infrastructure concern; the domain models task existence, not tombstone state.

The `Repository` interface gains two new methods and one clarification:

```go
// Delete soft-deletes a task. Returns ErrNotFound if the task does not exist
// or has already been deleted.
Delete(ctx context.Context, id string) error

// List returns all active (non-deleted) tasks, newest first.
List(ctx context.Context) ([]Task, error)

// GetByID returns a single active task. Returns ErrNotFound if the task does
// not exist or has been soft-deleted. (already declared; behaviour clarified)
GetByID(ctx context.Context, id string) (Task, error)
```

`ErrNotFound` is the single sentinel for "does not exist or was deleted" — the spec collapses both cases to 404.

---

## Application Layer

Three use cases added to the existing `Service`:

### `DeleteTask`

```go
func (s *Service) DeleteTask(ctx context.Context, id string) error
```

Delegates to `repo.Delete`. Propagates `ErrNotFound` unchanged. No validation logic — the ID comes from the URL path.

### `ListTasks`

```go
func (s *Service) ListTasks(ctx context.Context) ([]domain.Task, error)
```

Delegates to `repo.List`. Returns the slice as-is (empty slice, never nil, to guarantee a JSON array).

### `GetTaskByID`

```go
func (s *Service) GetTaskByID(ctx context.Context, id string) (domain.Task, error)
```

Delegates to `repo.GetByID`. Propagates `ErrNotFound` unchanged.

---

## Infrastructure — Postgres Repository

### `Delete`

```sql
UPDATE tasks
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
```

Checks `RowsAffected() == 0` → returns `domain.ErrNotFound`.

### `List`

```sql
SELECT id, title, description, status, created_at, updated_at
FROM tasks
WHERE deleted_at IS NULL
ORDER BY created_at DESC
```

Returns `[]domain.Task` (empty slice if no rows).

### `GetByID` (updated)

```sql
SELECT id, title, description, status, created_at, updated_at
FROM tasks
WHERE id = $1 AND deleted_at IS NULL
```

Adds `AND deleted_at IS NULL` to the existing query. Returns `domain.ErrNotFound` on no rows.

---

## HTTP Layer

### Handler methods

**`Delete`** (`DELETE /tasks/{id}`)
- On success: `204 No Content`, no body.
- On `ErrNotFound`: `404 {"error":"task not found"}`.
- On other errors: `500 {"error":"internal server error"}`.

**`List`** (`GET /tasks`)
- Always `200 OK` with a JSON array of `taskResponse` objects.
- Empty array `[]` when no active tasks exist (never `null`).

**`GetByID`** (`GET /tasks/{id}`)
- On success: `200 OK` with a single `taskResponse`.
- On `ErrNotFound`: `404 {"error":"task not found"}`.
- On other errors: `500 {"error":"internal server error"}`.

The existing `taskResponse` struct is reused unchanged.

### Router additions

All routes are added behind the existing `APIKey` middleware:

```go
r.With(middleware.APIKey(apiKey)).Get("/tasks", taskHandler.List)
r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}", taskHandler.GetByID)
r.With(middleware.APIKey(apiKey)).Delete("/tasks/{id}", taskHandler.Delete)
```

---

## Integration Tests

All tests go in the existing `task_integration_test.go`, following the established pattern (skip if `TEST_DATABASE_URL` unset, `t.Cleanup` to truncate).

### DELETE /tasks/{id}

| Test | Setup | Expected |
|------|-------|----------|
| Success | Create a task, DELETE it | 204; subsequent GET /tasks/{id} returns 404 |
| Not found | DELETE a nonexistent UUID | 404 `{"error":"task not found"}` |
| Already deleted | Create, DELETE, DELETE again | 404 `{"error":"task not found"}` |

### GET /tasks

| Test | Setup | Expected |
|------|-------|----------|
| Empty list | Truncate table, GET /tasks | 200 `[]` |
| Multiple tasks | Create two tasks, GET /tasks | 200 array with both tasks, newest first |

### GET /tasks/{id}

| Test | Setup | Expected |
|------|-------|----------|
| Success | Create a task, GET /tasks/{id} | 200 with correct fields |
| Deleted task | Create, DELETE, GET /tasks/{id} | 404 `{"error":"task not found"}` |

---

## Out of Scope

- Pagination for `GET /tasks` — not required.
- Returning `deleted_at` in any response — internal implementation detail.
- Hard delete / undelete — not required.
- Filtering or sorting beyond newest-first `created_at` — not required.
