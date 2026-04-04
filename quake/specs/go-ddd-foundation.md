# Spec: Go DDD Foundation — Task Management API

## Behavior

A Go HTTP server starts, loads config from `.env`, connects to PostgreSQL, registers routes, and listens for requests. A `GET /health` endpoint returns the server and DB status. On SIGINT/SIGTERM the server drains in-flight requests before exiting. The project compiles and runs with `make run`; migrations apply with `make migrate-up`.

---

## Interfaces

### Task 4: Config + environment loading
**`config.Config` struct**
- Fields: `DBUrl string`, `Port string` (default `"8080"`), `Env string` (default `"development"`)
- Loaded via `godotenv.Load(".env")` then `os.Getenv`
- **Errors**: returns error if `DBUrl` is empty

### Task 5: Database connection
**`postgres.Connect(dbUrl string) (*pgxpool.Pool, error)`**
- **Output**: open pool with `Ping()` verified
- **Errors**: returns error if connection or ping fails; caller exits

### Task 6: HTTP server + router
**`server.New(router http.Handler, port string) *http.Server`**
- Standard `http.Server` with 5s read/write timeouts
- **Graceful shutdown**: listens for OS signal, calls `Shutdown(ctx)` with 10s deadline

**`router.Register(pool *pgxpool.Pool) http.Handler`**
- Returns chi router with all routes mounted
- Called once at startup; passed to server

### Task 7: Health check handler
**`GET /health`**
- **Output (healthy)**: `200 {"status":"ok","db":"ok"}`
- **Output (unhealthy)**: `503 {"status":"error","db":"unreachable"}`
- DB check: `pool.Ping(ctx)` with 2s timeout

### Task 8: Entry point
**`cmd/api/main.go`**
- Sequence: load config → connect DB → build router → start server → block on signal → graceful shutdown
- Logs each step with `slog` (level, message, relevant fields)
- Exit code 1 on any startup failure

### Task 9: Migrations
**`migrations/000001_init.up.sql`** — valid SQL no-op (`SELECT 1;` or a comment)
**`migrations/000001_init.down.sql`** — valid SQL no-op

**Makefile `migrate-up` / `migrate-down`**
- Invokes `migrate -path ./migrations -database $$DATABASE_URL up/down`

### Task 10: Docker Compose
**`docker-compose.yml`** — Postgres 16 on port `5432`, volume for data persistence
**`docker-compose.test.yml`** — Postgres 16 on port `5433`, no volume (ephemeral)
- Both use env vars: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`

### Task 11: Makefile
| Target | Command |
|---|---|
| `run` | `go run ./cmd/api` |
| `test` | `go test ./...` |
| `migrate-up` | migrate CLI up against `$DATABASE_URL` |
| `migrate-down` | migrate CLI down 1 step |
| `docker-up` | `docker compose up -d` |
| `docker-down` | `docker compose down` |

---

## Edge Cases

- `DBUrl` missing from `.env` → startup fails with clear error, not a nil-pointer panic
- DB unreachable at startup → `Connect()` returns error, main exits with code 1
- `/health` called while DB is down → 503, server keeps running
- SIGTERM during a slow request → in-flight request completes (up to 10s), then server exits cleanly
- `.env` file absent → `godotenv.Load` logs a warning but does not fail (env vars may be set externally)

---

## Constraints

- `internal/domain/` must import only stdlib — enforced by Go module visibility
- Health check DB ping must use a context with timeout (not block indefinitely)
- Graceful shutdown context must have a deadline (not `context.Background()`)
- slog must be the only logging dependency — no `log.Println` or fmt in request path

---

## Test expectations

### Task 4: Config
- Returns error when `DBUrl` is empty string
- Applies default `Port = "8080"` when `PORT` env var is unset

### Task 5: Database connection
- `Connect()` succeeds when Postgres is running (integration, uses test compose DB)
- `Connect()` returns non-nil error for invalid DSN

### Task 7: Health check handler
- Returns 200 + `{"status":"ok","db":"ok"}` when DB is reachable
- Returns 503 + `{"status":"error","db":"unreachable"}` when pool ping fails (mock pool)

### Task 8: Entry point
- Server starts and responds to `GET /health` within 500ms (smoke test via `httptest`)

### Task 9: Migrations
- `migrate-up` applies without error against test DB
- `migrate-down` reverses without error

---

## NOT in scope

- Task CRUD endpoints or any business logic
- Auth, middleware, rate limiting
- Repository implementations (interfaces declared, no DB queries)
- CI/CD, linting, or code generation tooling
- Embedded SQL via `go:embed`
