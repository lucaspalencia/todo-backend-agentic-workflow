# Code Review — `quake` Go API

Overall quality is high: clean architecture, parameterized queries throughout, constant-time auth comparison, graceful shutdown, good test coverage. The issues below are sorted within each section by severity.

---

## 1. Code Organization & Idiomatic Go

### [MEDIUM] `ValidationError` / `ValidationErrors` are duplicated
`internal/application/task/service.go:33–47`
`internal/application/comment/service.go:27–41`

Both packages define identical types. The handler already imports both and uses them separately (`apptask.ValidationErrors`, `appcomment.ValidationErrors`). This will drift over time. Extract to a shared package, e.g. `internal/application/apperrors`.

```go
// internal/application/apperrors/validation.go
package apperrors

import "fmt"

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"error"`
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string { ... }
```

Both services and both handlers import from the same package; the `errors.As` calls in handlers still work.

---

### [MEDIUM] `FindByID` signals "not found" with `(nil, nil)` instead of a sentinel error
`internal/infrastructure/persistence/postgres/task_repository.go:46–48`

Returning `(nil, nil)` forces every caller to perform a nil check. This is an idiomatic Go anti-pattern; `nil, nil` conventionally means "success, no result", but that ambiguity triggers subtle bugs when a caller forgets the nil check.

```go
if errors.Is(err, pgx.ErrNoRows) {
    return nil, task.ErrNotFound   // propagate the sentinel, don't absorb it
}
```

Then callers simply `errors.Is(err, task.ErrNotFound)` and the `t == nil` guards throughout the service layer disappear.

---

### [LOW] Middleware applied per-route instead of a chi group
`internal/infrastructure/http/router.go:25–33`

`r.With(...)` is repeated seven times. Use a group:

```go
r.Group(func(r chi.Router) {
    r.Use(appmiddleware.APIKeyAuth(apiKey))
    r.Post("/tasks", taskHandler.Create)
    r.Get("/tasks", taskHandler.List)
    r.Get("/tasks/{id}", taskHandler.GetByID)
    r.Patch("/tasks/{id}", taskHandler.Update)
    r.Delete("/tasks/{id}", taskHandler.Delete)
    r.Post("/tasks/{id}/comments", commentHandler.Add)
    r.Get("/tasks/{id}/comments", commentHandler.List)
})
```

---

### [LOW] Time format string is repeated across the codebase
`internal/infrastructure/http/handler/task.go:111,112,151,152,184,185,209,210`
`internal/infrastructure/http/handler/comment.go:47`

Define a package-level constant:

```go
const rfc3339 = "2006-01-02T15:04:05Z07:00"
```

---

## 2. Security

### [HIGH] No request body size limit — potential memory exhaustion
`internal/infrastructure/http/handler/task.go:82`
`internal/infrastructure/http/handler/comment.go:56`

Both handlers decode the body without constraining its size. A client can stream gigabytes and exhaust server memory. Wrap the body before decoding:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or oversized request body"})
    return
}
```

---

### [HIGH] Missing `ReadHeaderTimeout` — Slowloris attack vector
`internal/infrastructure/http/server.go:22–28`

`ReadTimeout` covers body reads, but headers are read first and are not bounded by `ReadTimeout` alone. An attacker can trickle headers indefinitely, holding a goroutine and connection open.

```go
inner: &http.Server{
    Addr:              ":" + port,
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,   // <-- add this
    ReadTimeout:       10 * time.Second,  // increase to accommodate body
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
},
```

---

### [MEDIUM] No per-request timeout middleware
`internal/infrastructure/http/router.go`

`WriteTimeout` caps the entire response window, but there's no middleware-level context deadline that carries into handlers and database calls. Add `middleware.Timeout`:

```go
r.Use(middleware.Timeout(30 * time.Second))
```

This ensures the `context.Context` passed to services and repositories is cancelled if a request stalls.

---

### [LOW] No CORS middleware
`internal/infrastructure/http/router.go`

If this API is ever called from a browser, cross-origin requests will be blocked. Not a current bug, but worth a comment if it's intentionally server-to-server only, or worth adding `go-chi/cors` now.

---

### [LOW] No rate limiting
`internal/infrastructure/http/router.go`

No middleware bounds request volume per IP. `go-chi/httprate` is a one-liner that pairs well with chi and is worth adding before exposing this to the internet.

---

## 3. Concurrency & Performance

### [MEDIUM] DB pool created with default settings — no connection limits configured
`internal/infrastructure/persistence/postgres/db.go:17`

`pgxpool.New` uses library defaults (`MaxConns = 4` per CPU). Under real load this should be explicit and tunable via config:

```go
cfg, err := pgxpool.ParseConfig(dbUrl)
if err != nil {
    return nil, fmt.Errorf("parse pool config: %w", err)
}
cfg.MaxConns = 20
cfg.MinConns = 2
cfg.MaxConnLifetime = 30 * time.Minute
cfg.MaxConnIdleTime = 5 * time.Minute

pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

---

### [MEDIUM] `DeleteTask` makes two DB round-trips unnecessarily
`internal/application/task/service.go:115–124`
`internal/infrastructure/persistence/postgres/task_repository.go:82–85`

`DeleteTask` calls `FindByID` first, then `Delete`. The `Delete` query (`WHERE id=$1 AND deleted_at IS NULL`) already encodes existence semantics — if no rows are affected, the task didn't exist. Use `pgconn.CommandTag.RowsAffected()` to collapse it into one round-trip:

```go
// In repository:
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
    tag, err := r.pool.Exec(ctx, `UPDATE tasks SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return task.ErrNotFound
    }
    return nil
}
```

Then the service simply becomes:

```go
func (s *Service) DeleteTask(ctx context.Context, id string) error {
    return s.repo.Delete(ctx, id)
}
```

The same optimization applies to `Save` (for update conflicts) and `ListComments` / `AddComment` (which both call `taskRepo.FindByID` before doing their main work — `FindByTaskID` could return an empty slice for a nonexistent task, and `Create` would fail the FK constraint, though you'd lose the `ErrNotFound` distinction in that case — so the current pattern is acceptable there).

---

### [LOW] `FindAll` scans `deleted_at` even though the query guarantees it's NULL
`internal/infrastructure/persistence/postgres/task_repository.go:57,66`

`WHERE deleted_at IS NULL` makes the scanned value always `nil`. Either drop it from the SELECT list, or keep it — but don't expose `DeletedAt` in the HTTP response (and currently it isn't, which is good).

---

## 4. Reliability & Observability

### [HIGH] Unhandled errors silently become 500s — no server-side logging
`internal/infrastructure/http/handler/task.go:102–104, 141–142, 163–164, 173–175`
`internal/infrastructure/http/handler/comment.go:72–74, 89–90`

When an unexpected error reaches the handler, it writes a 500 and discards the error:

```go
writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
```

The actual error is never logged. In production, all 500s appear as silence. Add a log line before the response:

```go
slog.ErrorContext(r.Context(), "unexpected error", "error", err, "path", r.URL.Path)
writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
```

---

### [MEDIUM] `writeJSON` silently discards encoding errors
`internal/infrastructure/http/handler/task.go:213–217`

```go
_ = json.NewEncoder(w).Encode(body)
```

Encoding errors are rare but can occur with certain types. At minimum, log them:

```go
if err := json.NewEncoder(w).Encode(body); err != nil {
    slog.Error("failed to encode response", "error", err)
}
```

---

### [MEDIUM] `Save` (update) doesn't verify rows were actually updated
`internal/infrastructure/persistence/postgres/task_repository.go:74–80`

`Save` calls `UPDATE` without checking `RowsAffected`. If the ID somehow doesn't match, the update is silently a no-op. Fix:

```go
func (r *TaskRepository) Save(ctx context.Context, t *task.Task) error {
    tag, err := r.pool.Exec(ctx,
        `UPDATE tasks SET title=$1, description=$2, status=$3, updated_at=$4 WHERE id=$5`,
        t.Title, t.Description, t.Status, t.UpdatedAt, t.ID,
    )
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return task.ErrNotFound
    }
    return nil
}
```

---

### [LOW] `errCh` in server startup is never drained after graceful shutdown
`internal/infrastructure/http/server.go:37–57`

If `ListenAndServe` errors *and* a signal arrives simultaneously, the error in `errCh` is left unread — but the channel is buffered with size 1 so it won't leak. Not a bug, but worth a comment that the goroutine exits naturally when `Shutdown` causes `ListenAndServe` to return `http.ErrServerClosed`.

---

## 5. Testing & Maintainability

### [MEDIUM] `stubRepo.FindByID` ignores the `err` field — can't test error paths
`internal/application/task/service_test.go:27–29`

```go
func (s *stubRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) {
    return s.existing, nil   // s.err is never checked
}
```

This means you cannot write a test for "FindByID fails with a DB error" without changing the stub. Fix:

```go
func (s *stubRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) {
    return s.existing, s.err
}
```

---

### [MEDIUM] Integration test `truncate` swallows errors silently
`internal/infrastructure/http/handler/task_integration_test.go:35`

```go
truncate := func() { _, _ = pool.Exec(t.Context(), "TRUNCATE TABLE tasks") }
```

If truncation fails, tests run with leftover data, producing false failures with confusing diagnostics. Use `t.Fatal` on error and add `CASCADE` to also clean comments:

```go
truncate := func() {
    if _, err := pool.Exec(t.Context(), "TRUNCATE TABLE tasks CASCADE"); err != nil {
        t.Fatalf("truncate failed: %v", err)
    }
}
```

---

### [LOW] Missing test cases

- `PATCH /tasks/{id}` to a title that already exists (should 409)
- `POST /tasks` with a missing or malformed JSON body — integration test missing (unit test exists)
- `GET /tasks/{id}` and `DELETE /tasks/{id}` with a non-UUID path param (currently 404, arguably should be 400)
- Comment service: `ListComments` when repo returns a DB error

---

### [LOW] `createTaskForUpdate` helper name is misleading
`internal/infrastructure/http/handler/task_integration_test.go:169`

The function is a general-purpose "create task and return ID" helper used across delete, list, and get tests. Rename it to `createTask(t, srv) string`.

---

## 6. Dependency & Configuration

### [MEDIUM] No validation of the `PORT` value
`internal/infrastructure/config/config.go:29–32`

`Port` is accepted as any string. An invalid value (e.g. `"abc"`) will cause `ListenAndServe` to fail at startup with a confusing error. Validate it at load time:

```go
if _, err := strconv.Atoi(port); err != nil {
    return nil, fmt.Errorf("PORT %q is not a valid port number", port)
}
```

---

### [LOW] `godotenv.Load` path is relative to the working directory
`internal/infrastructure/config/config.go:22`

When running from a directory other than the project root (e.g. `go test ./...` from a subdirectory), `.env` won't be found and the load silently fails. This is intentional (best-effort), but can cause confusing behaviour during CI. A comment explaining this is sufficient.

---

### [LOW] All four direct dependencies are well-chosen and actively maintained

`chi`, `pgx/v5`, `uuid`, `godotenv` — no concerns. All pinned to specific versions in `go.sum`.

---

## Summary

| Severity | Count | Items |
|---|---|---|
| **HIGH** | 3 | No body size limit, missing `ReadHeaderTimeout`, 500 errors not logged |
| **MEDIUM** | 8 | No pool config, extra DB roundtrips, `Save` no-op risk, `writeJSON` discards errors, no per-request timeout, duplicated `ValidationError`, stub test gap, truncate swallows errors |
| **LOW** | 9 | Per-route middleware, time format const, CORS, rate limiting, scan unused column, `errCh` comment, missing test cases, helper name, godotenv path note |

The three HIGH items are the ones to address before any production deployment: body size limits, `ReadHeaderTimeout`, and logging 500 errors. Everything else is a quality or ergonomics improvement.
