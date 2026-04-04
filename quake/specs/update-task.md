# Spec: Update Task

## Behavior

A caller can partially update any combination of `title`, `description`, or `status` on an existing task via `PATCH /tasks/{id}`. Fields omitted from the request body are left unchanged. On success, the full updated task is returned with a refreshed `updated_at`. If the task does not exist, a 404 with a JSON error is returned.

## Interfaces

### Task 1: Domain — ErrNotFound
**`internal/domain/task/repository.go`**
- Add: `ErrNotFound = errors.New("task not found")`

### Task 2: Application — UpdateTask use case
**`UpdateTaskCmd`**
- Fields: `Title *string`, `Description *string`, `Status *string` (nil = not provided)

**`Service.UpdateTask(ctx, id string, cmd UpdateTaskCmd) (*task.Task, error)`**
- Fetches task by ID; returns `ErrNotFound` if `FindByID` returns nil
- Applies non-nil fields: trims and validates `title` (required, ≤255 runes), validates `description` (≤2000 runes), validates `status` against allowed set
- Sets `UpdatedAt = time.Now().UTC()`; calls `Save`
- Returns updated `*task.Task` on success, `ValidationErrors` on invalid input

### Task 3: Handler + router + integration tests
**`PATCH /tasks/{id}`** (requires `X-API-Key`)
- **Request body**: `{ "title"?: string, "description"?: string, "status"?: string }`
- **200**: full task JSON (same shape as POST /tasks response)
- **404**: `{ "error": "task not found" }`
- **422**: `{ "errors": [{ "field": "...", "error": "..." }] }`
- **401**: missing/invalid API key (existing middleware)

**`TaskService` interface** (in handler package)
```
type TaskService interface {
    TaskCreator
    TaskUpdater
}
```

## Edge Cases

- PATCH body with no fields provided: no-op — fetch, skip all nil fields, still refresh `updated_at` and save
- `title` set to empty string or whitespace: treated as invalid (required), returns 422
- `status` set to an unlisted value: returns 422
- `id` path param is not a valid UUID: `FindByID` will return nil → 404 (no need for format validation)
- `updated_at` must always advance even on a no-field-change request

## Constraints

- Partial update uses pointer semantics (`*string`) — absence of a JSON key must be distinguishable from an explicit empty string. Use a pointer-based decode struct in the handler.

## Test expectations

### Task 2: Application — UpdateTask use case
- `UpdateTask` with all fields returns updated task with refreshed `updated_at`
- `UpdateTask` with only `status` changed leaves `title` and `description` unchanged
- `UpdateTask` with unknown ID returns `ErrNotFound`
- `UpdateTask` with empty title returns `ValidationErrors` for field `title`
- `UpdateTask` with invalid status returns `ValidationErrors` for field `status`

### Task 3: Handler + router + integration tests
- `PATCH /tasks/{id}` with valid full body → 200 + updated task fields
- `PATCH /tasks/{id}` with only one field → 200 + that field changed, others unchanged
- `PATCH /tasks/{id}` for non-existent ID → 404 `{ "error": "task not found" }`
- `PATCH /tasks/{id}` with invalid status → 422 `{ "errors": [...] }`
- `PATCH /tasks/{id}` without API key → 401

## NOT in scope

- Title uniqueness check on update (no `ErrDuplicateTitle` handling in update path)
- Optimistic locking / ETags
- Partial response (always returns full task)
- List / get / delete task endpoints
