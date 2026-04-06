# Task Comments Feature — Design

**Date:** 2026-04-05

## Overview

Add a comments sub-resource to tasks. Each comment belongs to a task and is immutable after creation. Two endpoints are exposed: add a comment and list all comments for a task. Deleting a task also removes its comments.

---

## Data Model

```
comments
  id         TEXT        PRIMARY KEY          -- UUID, generated server-side
  task_id    TEXT        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE
  content    TEXT        NOT NULL             -- max 2000 characters
  created_at TIMESTAMPTZ NOT NULL
```

No `updated_at` or soft-delete — comments are immutable and permanently deleted when removed.

---

## Architecture

Follows the existing DDD layering: domain → application → infrastructure.

### Domain Layer (`internal/domain/comment/`)

**`entity.go`**
```go
type Comment struct {
    ID        string
    TaskID    string
    Content   string
    CreatedAt time.Time
}
```

**`repository.go`**
```go
var ErrNotFound = errors.New("comment not found")

type Repository interface {
    Create(ctx context.Context, c Comment) (Comment, error)
    ListByTaskID(ctx context.Context, taskID string) ([]Comment, error)
    DeleteByTaskID(ctx context.Context, taskID string) error
}
```

### Task Application Layer — Cascade Delete Hook

A narrow interface is added to `internal/application/task/`:

```go
type CommentDeleter interface {
    DeleteByTaskID(ctx context.Context, taskID string) error
}
```

`Service` gains an optional `commentDeleter CommentDeleter` field. A `WithCommentDeleter(d CommentDeleter)` method sets it. `DeleteTask` calls `commentDeleter.DeleteByTaskID` after the soft-delete (nil-safe — no-op if unset, preserving backward compatibility with existing tests).

### Application Layer (`internal/application/comment/`)

**`service.go`** — `Service` holds a `domaincomment.Repository` and a `domaintask.Repository`.

```go
type ValidationError struct {
    Fields map[string]string
}
```

**`AddComment(ctx, taskID, content string) (Comment, error)`**
1. Validate `content`: non-empty, ≤ 2000 chars. Return `*ValidationError` on failure.
2. `taskRepo.GetByID(taskID)` — return `domaintask.ErrNotFound` if task missing or soft-deleted.
3. Generate UUID, set `CreatedAt = time.Now().UTC()`, call `repo.Create`.

**`ListComments(ctx, taskID string) ([]Comment, error)`**
1. `taskRepo.GetByID(taskID)` — return `domaintask.ErrNotFound` if task missing or soft-deleted.
2. `repo.ListByTaskID(taskID)` — always return non-nil slice.

### Infrastructure Layer

**Migration** — `000004_create_comments.up.sql`:
```sql
CREATE TABLE comments (
    id         TEXT        PRIMARY KEY,
    task_id    TEXT        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```
Down migration: `DROP TABLE IF EXISTS comments;`

**`internal/infrastructure/persistence/postgres/comment_repository.go`**
Implements `domain/comment.Repository` using pgxpool. Follows the same patterns as `task_repository.go`: `QueryRow` + `Scan` for single rows, `Query` + rows loop for lists, `Exec` for deletes.

**`internal/infrastructure/http/handler/comment.go`**

`CommentHandler` wraps `*appcomment.Service`.

- `POST /tasks/{id}/comments` — decodes `{"content": "..."}`, calls `AddComment`, returns 201 with comment JSON.
- `GET /tasks/{id}/comments` — calls `ListComments`, returns 200 with JSON array.

Error mapping (both handlers):
- `*appcomment.ValidationError` → 400 `{"errors": {"content": "..."}}`
- `domaintask.ErrNotFound` → 404 `{"error": "task not found"}`
- anything else → 500 `{"error": "internal server error"}`

Comment response shape:
```json
{
  "id": "uuid",
  "task_id": "uuid",
  "content": "...",
  "created_at": "RFC3339"
}
```

**`internal/infrastructure/http/router.go`** changes:
- Extract `taskRepo` to a named variable (currently created inline) so it can be passed to the comment service.
- Wire `commentRepo`, `commentSvc`, `commentHandler`.
- Register two new routes under `middleware.APIKey`:
  - `POST /tasks/{id}/comments`
  - `GET /tasks/{id}/comments`
- Call `taskSvc.WithCommentDeleter(commentRepo)` to wire cascade delete.

---

## HTTP API

### POST /tasks/{id}/comments

**Request**
```json
{ "content": "This is a comment." }
```

**Responses**
| Status | Body |
|--------|------|
| 201 | Comment object |
| 400 | `{"errors": {"content": "content is required"}}` |
| 404 | `{"error": "task not found"}` |
| 500 | `{"error": "internal server error"}` |

### GET /tasks/{id}/comments

**Responses**
| Status | Body |
|--------|------|
| 200 | Array of comment objects, ordered by `created_at ASC` |
| 404 | `{"error": "task not found"}` |
| 500 | `{"error": "internal server error"}` |

---

## Validation Rules

| Field | Rule |
|-------|------|
| `content` | Required (non-empty after trim), max 2000 characters |

---

## Integration Tests

File: `internal/infrastructure/http/handler/comment_integration_test.go`

Reuses `integPool` and `integServer` from `TestMain` in the same package. A `createCommentsSQL` constant sets up the comments table in test setup. Teardown drops `comments` before `tasks` (FK order).

| Test | Scenario |
|------|----------|
| `TestAddComment_Integration_Success` | POST comment → 201, all fields present in response |
| `TestAddComment_Integration_TaskNotFound` | POST to unknown task_id → 404 |
| `TestAddComment_Integration_ValidationError` | Empty content → 400 with `errors.content` set |
| `TestListComments_Integration_Empty` | GET with no comments → 200 `[]` |
| `TestListComments_Integration_OrderedAsc` | Two comments → returned oldest-first |
| `TestListComments_Integration_TaskNotFound` | GET for unknown task_id → 404 |
| `TestDeleteTask_Integration_CascadeDeletesComments` | Create task + comment, delete task via API, verify GET /tasks/{id}/comments returns 404 AND `SELECT COUNT(*) FROM comments WHERE task_id = $1` returns 0 |

---

## Files Created / Modified

| Action | Path |
|--------|------|
| Create | `migrations/000004_create_comments.up.sql` |
| Create | `migrations/000004_create_comments.down.sql` |
| Create | `internal/domain/comment/entity.go` |
| Create | `internal/domain/comment/repository.go` |
| Create | `internal/application/comment/service.go` |
| Modify | `internal/application/task/service.go` — add `CommentDeleter` interface + `WithCommentDeleter` + nil-safe call in `DeleteTask` |
| Create | `internal/infrastructure/persistence/postgres/comment_repository.go` |
| Create | `internal/infrastructure/http/handler/comment.go` |
| Modify | `internal/infrastructure/http/router.go` — wire comment dependencies + new routes |
| Create | `internal/infrastructure/http/handler/comment_integration_test.go` |
| Modify | `internal/infrastructure/http/handler/task_integration_test.go` — add `createCommentsSQL` to `TestMain` setup; drop `comments` before `tasks` in teardown (FK order) |
