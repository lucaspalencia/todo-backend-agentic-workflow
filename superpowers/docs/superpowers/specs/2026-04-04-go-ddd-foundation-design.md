# Design: Go DDD Foundation — Task Management API

## Overview

A Go HTTP server that provides the structural foundation for a task management REST API. No business features are implemented — only the project skeleton, infrastructure wiring, and health check endpoint. The goal is a clean, compiling, running server with DDD layer boundaries enforced and all tooling (migrations, Docker, Makefile) in place.

---

## Project Structure

```
superpowers/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/
│   │   └── .gitkeep
│   ├── application/
│   │   └── .gitkeep
│   └── infrastructure/
│       ├── config/
│       │   └── config.go
│       ├── persistence/
│       │   └── postgres/
│       │       └── db.go
│       └── http/
│           ├── handler/
│           │   └── health.go
│           ├── router.go
│           └── server.go
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── docker-compose.yml
├── docker-compose.test.yml
├── Makefile
├── .env.example
├── go.mod
└── go.sum
```

### Layer rules

- `internal/domain/` — no external imports; only stdlib allowed
- `internal/application/` — may import `internal/domain/` only
- `internal/infrastructure/` — may import `internal/application/` and `internal/domain/`
- Dependency direction: infrastructure → application → domain

---

## Architecture

### Option selected: Layer-first with feature subdirectories (Option B)

Each domain concept gets its own subdirectory within each layer (e.g., `domain/task/`, `application/task/`). Infrastructure is split by adapter type: `persistence/postgres/` for DB and `http/` for HTTP concerns. This scales naturally as business features are added without collapsing into crowded flat packages.

---

## Interfaces

### Config (`internal/infrastructure/config/config.go`)

```go
type Config struct {
    DBUrl string
    Port  string // default "8080"
    Env   string // default "development"
}

func Load() (Config, error)
```

- Calls `godotenv.Load(".env")` — error is explicitly ignored if `.env` is absent (env vars may be injected externally)
- Returns error if `DBUrl` is empty

### Database (`internal/infrastructure/persistence/postgres/db.go`)

```go
func Connect(dbUrl string) (*pgxpool.Pool, error)
```

- Opens pool, verifies with `Ping()`
- Returns error on bad DSN or unreachable host

### HTTP Server (`internal/infrastructure/http/server.go`)

```go
func New(router http.Handler, port string) *http.Server
func GracefulShutdown(srv *http.Server)
```

- `http.Server` with 5s read/write timeouts
- Graceful shutdown: traps SIGINT/SIGTERM, calls `Shutdown(ctx)` with 10s deadline

### Router (`internal/infrastructure/http/router.go`)

```go
func Register(pool *pgxpool.Pool) http.Handler
```

- Returns chi router with all routes mounted
- Called once at startup

### Health Handler (`internal/infrastructure/http/handler/health.go`)

```go
// GET /health
```

- `200 {"status":"ok","db":"ok"}` when DB reachable
- `503 {"status":"error","db":"unreachable"}` on ping failure
- Uses `pool.Ping(ctx)` with 2s timeout

### Entry Point (`cmd/api/main.go`)

Startup sequence:
1. Load config
2. Connect to DB
3. Build router
4. Start server
5. Block on OS signal
6. Graceful shutdown

Each step logged with `slog`. Exit code 1 on any startup failure.

---

## Data Flow

```
main.go
  → config.Load()
  → postgres.Connect(cfg.DBUrl)
  → http.Register(pool)        ← chi router with /health mounted
  → server.New(router, cfg.Port)
  → server.ListenAndServe()
  → [OS signal]
  → server.Shutdown(ctx)       ← 10s deadline
```

---

## Docker Compose

### docker-compose.yml (local dev)
- Postgres 16, port `5432`
- Named volume for data persistence
- Env vars: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`

### docker-compose.test.yml (integration tests)
- Postgres 16, port `5433`
- No volume (ephemeral — clean state per test run)
- Same env var pattern

---

## Migrations

- `000001_init.up.sql` — `SELECT 1;` (valid no-op to bootstrap `schema_migrations` table)
- `000001_init.down.sql` — `SELECT 1;` (valid no-op rollback)
- Tool: `golang-migrate` CLI
- Migration files live in `./migrations/`

---

## Makefile

| Target | Command |
|---|---|
| `run` | `go run ./cmd/api` |
| `test` | `go test ./...` |
| `migrate-up` | `migrate -path ./migrations -database $$DATABASE_URL up` |
| `migrate-down` | `migrate -path ./migrations -database $$DATABASE_URL down 1` |
| `docker-up` | `docker compose up -d` |
| `docker-down` | `docker compose down` |

Makefile auto-loads `.env` when present so `DATABASE_URL` is available in shell targets.

---

## Environment

`.env.example`:
```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable
PORT=8080
ENV=development
```

---

## Error Handling

- `DBUrl` missing → `config.Load()` returns error; `main.go` logs and exits with code 1
- DB unreachable at startup → `postgres.Connect()` returns error; `main.go` exits with code 1
- `GET /health` called while DB is down → 503 response; server keeps running
- `.env` absent → `godotenv.Load` error is ignored; server continues (env vars expected to be set externally)
- SIGTERM during slow request → in-flight request completes (up to 10s), then server exits cleanly

---

## Testing

| Component | Type | What's tested |
|---|---|---|
| `config.Load()` | Unit | Error on empty `DBUrl`; default `Port = "8080"` |
| `postgres.Connect()` | Integration | Success against test DB; error on invalid DSN |
| `GET /health` | Unit | 200 when pool ping succeeds; 503 when ping fails (mock pool) |
| `main.go` smoke test | Integration | Server starts and responds to `GET /health` within 500ms |
| Migrations | Integration | `migrate-up` and `migrate-down` apply cleanly against test DB |

---

## Module

```
module github.com/lucaspalencia/superpowers

go 1.24
```

Dependencies:
- `github.com/go-chi/chi/v5` — HTTP router
- `github.com/jackc/pgx/v5` — PostgreSQL driver (pgxpool)
- `github.com/joho/godotenv` — `.env` loading

---

## Not in Scope

- Task CRUD endpoints or any business logic
- Auth, middleware, rate limiting
- Repository implementations (interfaces declared, no DB queries)
- CI/CD, linting, or code generation tooling
- Embedded SQL via `go:embed`
