# Spec: Task Comments

## Behavior

Callers can add a comment to an existing task and list all comments for a task.
Both operations require the parent task to exist; a missing task yields 404.
Comments are returned in chronological order. Deleting a task removes all its
comments automatically (ON DELETE CASCADE — no explicit delete endpoint).

## Interfaces

### Task 1: Migration
**`000004_create_comments` table**
- `id UUID PRIMARY KEY`
- `task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE`
- `content VARCHAR(2000) NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Down migration: `DROP TABLE comments`

### Task 2: Domain
**`comment.Comment` struct**
- Fields: `ID string`, `TaskID string`, `Content string`, `CreatedAt time.Time`

**`comment.Repository` interface**
- `Create(ctx, *Comment) error`
- `FindByTaskID(ctx, taskID string) ([]Comment, error)`

**`comment.ErrNotFound`** — sentinel (unused now, but mirrors task pattern)

### Task 3: Application
**`comment.Service.AddComment(ctx, taskID, content string) (*Comment, error)`**
- Validates: `content` required, `utf8.RuneCountInString(content) <= 2000`
- Checks task exists via injected `task.Repository`; returns `task.ErrNotFound` if not
- Generates UUID + `time.Now().UTC()` for `ID` and `CreatedAt`
- Validation error shape: `ValidationErrors` (same type as `apptask.ValidationErrors`)
- **Input**: `taskID string`, `content string`
- **Output**: `*comment.Comment` or error

**`comment.Service.ListComments(ctx, taskID string) ([]Comment, error)`**
- Checks task exists; returns `task.ErrNotFound` if not
- Delegates to `repo.FindByTaskID` — ordering is done at DB level

**`comment.NewService(taskRepo task.Repository, commentRepo Repository) *Service`**

### Task 4: Persistence
**`postgres.CommentRepository`**
- `Create`: `INSERT INTO comments (id, task_id, content, created_at) VALUES ($1,$2,$3,$4)`
- `FindByTaskID`: `SELECT ... FROM comments WHERE task_id = $1 ORDER BY created_at ASC`

### Task 5: HTTP Handler
**`POST /tasks/{id}/comments`** → 201
```json
// Request
{ "content": "string" }
// Response
{ "id": "uuid", "task_id": "uuid", "content": "string", "created_at": "ISO8601" }
```
- 422: `{"errors": [{"field":"content","error":"..."}]}`
- 404: `{"error": "task not found"}`

**`GET /tasks/{id}/comments`** → 200
```json
// Response — array, may be empty []
[{ "id": "uuid", "task_id": "uuid", "content": "string", "created_at": "ISO8601" }]
```
- 404: `{"error": "task not found"}`

**`CommentHandler`** takes a `CommentService` interface:
```go
type CommentService interface {
    AddComment(ctx, taskID, content string) (*domcomment.Comment, error)
    ListComments(ctx, taskID string) ([]domcomment.Comment, error)
}
```

### Task 6: Router + Wiring
**`router.go`** — `Register` signature gains `commentSvc handler.CommentService`:
```go
r.With(appmiddleware.APIKeyAuth(apiKey)).Post("/tasks/{id}/comments", commentHandler.Add)
r.With(appmiddleware.APIKeyAuth(apiKey)).Get("/tasks/{id}/comments", commentHandler.List)
```
**`main.go`** — wire `CommentRepository` and `CommentService`, pass to `Register`.

### Task 7: Integration Tests
Setup: truncate both `tasks` AND `comments` before/after each test.
Helper functions: `postComment(t, srv, taskID, body, apiKey)`, `listComments(t, srv, taskID, apiKey)`.

## Edge Cases

- Empty `content` (blank string after trim) → 422 `{"errors":[{"field":"content","error":"content is required"}]}`
- `content` exactly 2000 chars → 201 (valid boundary)
- `content` 2001 chars → 422 `{"errors":[{"field":"content","error":"content must not exceed 2000 characters"}]}`
- Task ID in path does not exist → 404 for both POST and GET
- GET on task with no comments → 200 with `[]`
- Delete task → cascade removes comments (verified by GET returning 404, not empty list)

## Test Expectations

### Task 1: Migration
- Up migration runs clean on existing DB (tasks table present)
- Down migration drops comments table without affecting tasks

### Task 3: Application (unit, stub repos)
- `AddComment` success: returns comment with generated ID and correct fields
- `AddComment` blank content → `ValidationErrors` with field `"content"`
- `AddComment` content > 2000 chars → `ValidationErrors` with field `"content"`
- `AddComment` unknown taskID → returns `task.ErrNotFound`
- `ListComments` success: returns slice from repo
- `ListComments` unknown taskID → returns `task.ErrNotFound`

### Task 5: HTTP Handler (unit, stub service)
- `Add`: valid body → 201 with comment JSON
- `Add`: invalid JSON → 400
- `Add`: validation error → 422 with errors array
- `Add`: task not found → 404
- `List`: success → 200 with array
- `List`: task not found → 404

### Task 7: Integration Tests
- `POST /tasks/{id}/comments` → 201, response has `id`, `task_id`, `content`, `created_at`
- `GET /tasks/{id}/comments` → 200, ordered by `created_at` ASC
- `POST` with missing task ID → 404
- `GET` with missing task ID → 404
- `POST` with blank content → 422
- `POST` with no API key → 401
- Delete task → `GET /tasks/{id}/comments` returns 404 (task gone, cascade deleted comments)

## NOT in scope
- Updating or deleting individual comments
- Pagination
- Soft delete on comments
- Comments on anything other than tasks
