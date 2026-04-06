# Go API Code Review

Overall this is a clean, well-structured project with good layering, parameterized queries (no SQL injection), proper sentinel errors, and a working integration test suite. The issues below are real gaps, not nitpicks.

---

## 1. Code Organization & Idiomatic Go

### `handler/task.go:38` and `handler/comment.go:26` — HIGH
**Handlers hold concrete service types, not interfaces**

```go
type TaskHandler struct {
    svc *apptask.Service  // concrete — impossible to unit-test the handler layer
}
```

This is why there are no handler unit tests: you can't inject a mock. Define service interfaces at the handler package boundary:

```go
type taskService interface {
    CreateTask(ctx context.Context, in apptask.CreateTaskInput) (domain.Task, error)
    UpdateTask(ctx context.Context, in apptask.UpdateTaskInput) (domain.Task, error)
    DeleteTask(ctx context.Context, id string) error
    ListTasks(ctx context.Context) ([]domain.Task, error)
    GetTaskByID(ctx context.Context, id string) (domain.Task, error)
}

type TaskHandler struct {
    svc taskService
}
```

The concrete `*apptask.Service` satisfies it automatically; nothing else changes.

---

### `application/task/service.go:22` and `application/comment/service.go:14` — MEDIUM
**`ValidationError` is duplicated across two packages**

Both packages define identical `ValidationError` structs. Callers in `handler/comment.go` already import two different `ValidationError` types. Move it once to a shared package:

```
internal/apperror/validation.go
```

```go
package apperror

import (
    "fmt"
    "strings"
)

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
```

---

### `application/task/service.go:53` — MEDIUM
**`WithCommentDeleter` is a mutation-after-construction trap**

```go
taskSvc := apptask.NewService(taskRepo)
taskSvc.WithCommentDeleter(commentRepo)  // silently skipped if forgotten
```

If a future caller forgets this step, `DeleteTask` silently skips comment cleanup with no warning. Accept the dependency at construction time:

```go
func NewService(repo domain.Repository, commentDeleter CommentDeleter) *Service {
    return &Service{repo: repo, commentDeleter: commentDeleter}
}
```

Wire `nil` explicitly if not needed — at least the omission is visible at the call site.

---

### `internal/infrastructure/http/` — LOW
**Package named `http` shadows the stdlib**

Every file in the infrastructure layer needs `nethttp "net/http"` aliases. Rename to `httpserver` or `server` to eliminate the cognitive overhead and alias noise.

---

### `go.mod:12` — LOW
**`github.com/google/uuid` is marked `// indirect` but used directly**

It's called in `service.go` and `comment/service.go`. Run `go mod tidy` — it will be promoted to a direct dependency.

---

## 2. Security

### `router.go` — HIGH
**No rate limiting**

Any IP can send unlimited requests. Add rate limiting before the auth middleware so that even failed auth attempts are throttled:

```go
import "github.com/go-chi/httprate"

r.Use(httprate.LimitByIP(100, 1*time.Minute))
```

---

### `handler/task.go:48`, `handler/comment.go:40` — HIGH
**No request body size cap — unbounded reads**

`json.NewDecoder(r.Body).Decode(...)` will read until EOF. A client sending a 1 GB body will pin a goroutine and exhaust memory. Apply globally in the router:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
        next.ServeHTTP(w, r)
    })
})
```

---

### `router.go` — HIGH
**No CORS headers**

If this API is ever consumed from a browser, any origin can call it. Add the `go-chi/cors` middleware with an explicit allow-list:

```go
import "github.com/go-chi/cors"

r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"https://yourdomain.com"},
    AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: false,
}))
```

---

### `server.go:13-20` — MEDIUM
**Missing `IdleTimeout` and `ReadHeaderTimeout`; `WriteTimeout` too tight**

```go
return &nethttp.Server{
    Addr:              ":" + port,
    Handler:           router,
    ReadTimeout:       5 * time.Second,
    ReadHeaderTimeout: 5 * time.Second,   // add: prevents Slowloris
    WriteTimeout:      30 * time.Second,  // 5s will cut off large list responses
    IdleTimeout:       120 * time.Second, // add: prevents keep-alive pile-up
}
```

---

### `task_repository.go:62-69` — MEDIUM
**`Update` does not guard against updating soft-deleted tasks**

```sql
UPDATE tasks
SET title = $1, description = $2, status = $3, updated_at = $4
WHERE id = $5              -- missing: AND deleted_at IS NULL
RETURNING ...
```

A client who knows a deleted task's ID can still mutate it at the repository level. Fix:

```sql
WHERE id = $5 AND deleted_at IS NULL
```

Then handle the zero-rows case as `ErrNotFound` (same pattern as `Delete`).

---

### `config.go:18` — LOW
**Silently ignores `.env` parse errors**

A malformed `.env` (e.g. unquoted value with spaces) causes `godotenv.Load` to return an error, but all vars load as empty and then fail required-field validation with a confusing message. Log a warning:

```go
if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
    slog.Warn(".env found but could not be parsed", "error", err)
}
```

---

## 3. Concurrency & Performance

### `postgres/db.go:11,16` — MEDIUM
**`pgxpool.New` and `Ping` use `context.Background()` — no startup timeout**

If the database is unreachable, the ping blocks until the OS TCP timeout (minutes). Use a bounded context:

```go
func Connect(dbUrl string) (*pgxpool.Pool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pool, err := pgxpool.New(ctx, dbUrl)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping db: %w", err)
    }
    return pool, nil
}
```

---

### `postgres/db.go` — MEDIUM
**No connection pool limits**

pgxpool defaults to `4 × NumCPU` max connections. On a box with 16 cores, that's 64 connections — potentially exhausting Postgres's `max_connections`. Make it explicit:

```go
cfg, err := pgxpool.ParseConfig(dbUrl)
if err != nil {
    return nil, fmt.Errorf("parse db url: %w", err)
}
cfg.MaxConns = 20
cfg.MinConns = 2
cfg.MaxConnIdleTime = 5 * time.Minute
pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

---

### `application/task/service.go:158-167` — MEDIUM
**`DeleteTask` is non-atomic: two writes, no transaction**

```go
s.repo.Delete(ctx, id)                        // soft-deletes the task
s.commentDeleter.DeleteByTaskID(ctx, id)      // may fail after the above succeeds
```

If the second call fails, the task is gone but its comments are orphaned. The DB schema already defines `ON DELETE CASCADE` on the FK — but that only fires on physical row deletion, not on soft-delete (setting `deleted_at`).

Options, in order of preference:
1. Wrap both ops in a `pgx.Tx` passed through the call chain
2. Accept eventual consistency and log the partial failure rather than returning an error that leaves the soft-delete half-done
3. Replace the soft-delete with a physical delete and rely solely on the FK cascade

---

### `comment_integration_test.go:307`, `task_integration_test.go:585` — LOW
**JSON built by string concatenation in test helpers**

```go
body := `{"title":"` + title + `"}`  // breaks if title contains " or \
```

Use `json.Marshal`:

```go
b, _ := json.Marshal(map[string]string{"title": title})
req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewReader(b))
```

---

## 4. Reliability & Observability

### `router.go` — HIGH
**No panic recovery middleware**

An unhandled panic in any handler kills the entire server process, dropping all in-flight connections. Chi ships a `Recoverer` middleware — add it as the first middleware so it catches panics from everything downstream:

```go
import "github.com/go-chi/chi/v5/middleware"

r := chi.NewRouter()
r.Use(middleware.Recoverer)
```

---

### `server.go:36` and `cmd/api/main.go:35` — HIGH
**`os.Exit(1)` bypasses deferred cleanup**

In `server.go`:
```go
if err := srv.Shutdown(ctx); err != nil {
    os.Exit(1)  // skips defer pool.Close() in main
}
```

In `main.go`:
```go
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        os.Exit(1)  // same problem
    }
}()
```

Restructure so errors surface to `main` before exiting:

```go
// main.go
serverErr := make(chan error, 1)
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        serverErr <- err
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

select {
case err := <-serverErr:
    slog.Error("server error", "error", err)
    os.Exit(1) // defer pool.Close() still won't run here; accept it for fatal errors
case <-quit:
    if err := infrahttp.GracefulShutdown(srv); err != nil {
        slog.Error("shutdown error", "error", err)
    }
}
// defer pool.Close() runs here on clean shutdown
```

`GracefulShutdown` should return the error rather than exit internally.

---

### `router.go` — MEDIUM
**No request logging**

There is no log of method, path, status code, or latency per request. Chi has a built-in structured logger:

```go
r.Use(middleware.RequestID)
r.Use(middleware.Logger)
```

---

### `handler/health.go` — MEDIUM
**Health response carries no version or uptime**

In a multi-replica deployment you can't tell if you're hitting a stale instance. Embed the binary version (injected at build time via `-ldflags`) and start time:

```go
var (
    Version   = "dev"
    StartTime = time.Now()
)

json.NewEncoder(w).Encode(map[string]string{
    "status":  "ok",
    "db":      "ok",
    "version": Version,
    "uptime":  time.Since(StartTime).Round(time.Second).String(),
})
```

---

## 5. Testing & Maintainability

### `handler/` — MEDIUM
**No handler unit tests — only integration tests**

`task_test.go` does not exist. `health_test.go` correctly uses `mockPinger` and is the model to follow. Without unit tests, handler logic (error mapping, status codes) is only tested when a real Postgres instance is available.

Once `TaskHandler` takes an interface (see item 1), add:

```go
// handler/task_test.go
type mockTaskService struct {
    createErr error
    // ...
}

func TestCreate_MissingTitle_Returns400(t *testing.T) {
    h := NewTaskHandler(&mockTaskService{
        createErr: &apperror.ValidationError{Fields: map[string]string{"title": "required"}},
    })
    req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{}`))
    rec := httptest.NewRecorder()
    h.Create(rec, req)
    if rec.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", rec.Code)
    }
}
```

---

### `application/task/service_test.go:117-186` — MEDIUM
**Validation tests written as flat functions instead of table-driven**

Seven separate `TestCreateTask_*` functions test individual validation paths. A table-driven approach is more concise and easier to extend:

```go
func TestCreateTask_ValidationErrors(t *testing.T) {
    svc := apptask.NewService(&mockRepo{}, nil)
    tests := []struct {
        name    string
        input   apptask.CreateTaskInput
        wantKey string
    }{
        {"empty title",           apptask.CreateTaskInput{},                                "title"},
        {"title too long",        apptask.CreateTaskInput{Title: strings.Repeat("a", 256)}, "title"},
        {"description too long",  apptask.CreateTaskInput{
            Title:       "ok",
            Description: strings.Repeat("x", 2001),
        }, "description"},
        {"invalid status",        apptask.CreateTaskInput{Title: "ok", Status: "bogus"},    "status"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := svc.CreateTask(context.Background(), tt.input)
            var ve *apperror.ValidationError
            if !errors.As(err, &ve) {
                t.Fatalf("expected ValidationError, got %T", err)
            }
            if _, ok := ve.Fields[tt.wantKey]; !ok {
                t.Errorf("expected field %q in errors, got %v", tt.wantKey, ve.Fields)
            }
        })
    }
}
```

---

### `router.go` — LOW
**No API versioning**

All routes are at root level (`/tasks`). Adding `/v2` later requires renaming every client integration. Wrap now at zero cost:

```go
r.Route("/v1", func(r chi.Router) {
    r.Use(middleware.APIKey(apiKey))
    r.Post("/tasks", taskHandler.Create)
    r.Get("/tasks", taskHandler.List)
    // ...
})
```

---

## Summary Table

| # | File(s) | Severity | Category |
|---|---------|----------|----------|
| 1 | `handler/task.go:38`, `handler/comment.go:26` | HIGH | Testability |
| 2 | `router.go` | HIGH | Security (rate limiting) |
| 3 | `handler/task.go:48`, `handler/comment.go:40` | HIGH | Security (body size) |
| 4 | `router.go` | HIGH | Security (CORS) |
| 5 | `router.go` | HIGH | Reliability (no panic recovery) |
| 6 | `server.go:36`, `main.go:35` | HIGH | Reliability (os.Exit bypasses cleanup) |
| 7 | `service.go:22`, `comment/service.go:14` | MEDIUM | Code org (duplicated error type) |
| 8 | `service.go:53` | MEDIUM | Reliability (opt-in dependency) |
| 9 | `server.go:13` | MEDIUM | Security (missing timeouts) |
| 10 | `task_repository.go:62` | MEDIUM | Correctness (update soft-deleted rows) |
| 11 | `postgres/db.go:11` | MEDIUM | Reliability (no startup timeout) |
| 12 | `postgres/db.go` | MEDIUM | Performance (no pool limits) |
| 13 | `service.go:158` | MEDIUM | Reliability (non-atomic delete) |
| 14 | `router.go` | MEDIUM | Observability (no request logging) |
| 15 | `handler/` | MEDIUM | Testing (no handler unit tests) |
| 16 | `service_test.go:117` | MEDIUM | Testing (no table-driven tests) |
| 17 | `handler/health.go` | MEDIUM | Observability (no version/uptime) |
| 18 | `config.go:18` | LOW | Reliability (silent .env parse error) |
| 19 | `test helpers` | LOW | Correctness (JSON string concat) |
| 20 | `router.go` | LOW | Maintainability (no API versioning) |
| 21 | `internal/infrastructure/http/` | LOW | Code org (package naming clash) |
| 22 | `go.mod:12` | LOW | Module hygiene (uuid marked indirect) |
