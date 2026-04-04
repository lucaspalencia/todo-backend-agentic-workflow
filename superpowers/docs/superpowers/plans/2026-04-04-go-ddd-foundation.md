# Go DDD Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold a compiling, running Go HTTP server with DDD layer structure, PostgreSQL wiring, health check endpoint, migrations tooling, and Docker Compose — no business features.

**Architecture:** Layer-first with feature subdirectories (infrastructure → application → domain). `internal/domain/` imports only stdlib; `internal/application/` imports domain only; `internal/infrastructure/` imports application and domain. The HTTP package is named `http` (aliased as `infrahttp` at call sites to avoid conflict with `net/http`).

**Tech Stack:** Go 1.24, chi v5, pgx v5 (pgxpool), godotenv, golang-migrate CLI, Docker Compose, slog (stdlib).

---

## File Map

| File | Responsibility |
|---|---|
| `go.mod` | Module definition and dependencies |
| `.gitignore` | Ignore binaries, .env |
| `.env.example` | Document required env vars |
| `Makefile` | All build/run/migrate/docker targets |
| `docker-compose.yml` | Local dev Postgres 16 on port 5432 with volume |
| `docker-compose.test.yml` | Ephemeral test Postgres 16 on port 5433 |
| `migrations/000001_init.up.sql` | No-op bootstrap migration |
| `migrations/000001_init.down.sql` | No-op rollback |
| `internal/domain/.gitkeep` | Placeholder — preserves empty layer dir |
| `internal/application/.gitkeep` | Placeholder — preserves empty layer dir |
| `internal/infrastructure/config/config.go` | Load .env + os.Getenv, validate DBUrl |
| `internal/infrastructure/config/config_test.go` | Unit tests for config loading |
| `internal/infrastructure/persistence/postgres/db.go` | pgxpool.New + Ping |
| `internal/infrastructure/persistence/postgres/db_test.go` | Integration tests for DB connection |
| `internal/infrastructure/http/handler/health.go` | GET /health with DBPinger interface |
| `internal/infrastructure/http/handler/health_test.go` | Unit tests with mock DBPinger |
| `internal/infrastructure/http/router.go` | chi router, mounts /health |
| `internal/infrastructure/http/server.go` | http.Server with timeouts + GracefulShutdown |
| `cmd/api/main.go` | Entry point: wire and start server |
| `cmd/api/main_test.go` | Integration smoke test via httptest |

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `.env.example`
- Create: `internal/domain/.gitkeep`
- Create: `internal/application/.gitkeep`

- [ ] **Step 1: Create go.mod**

```
module github.com/lucaspalencia/superpowers

go 1.24
```

Save to `go.mod`.

- [ ] **Step 2: Create .gitignore**

```
.env
api
*.exe
```

Save to `.gitignore`.

- [ ] **Step 3: Create .env.example**

```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable
PORT=8080
ENV=development

# Used by integration tests (docker-compose.test.yml runs Postgres on 5433)
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable
```

Save to `.env.example`.

- [ ] **Step 4: Create placeholder files for empty DDD layers**

```bash
mkdir -p internal/domain internal/application
touch internal/domain/.gitkeep internal/application/.gitkeep
```

- [ ] **Step 5: Fetch dependencies**

```bash
go get github.com/go-chi/chi/v5
go get github.com/jackc/pgx/v5
go get github.com/joho/godotenv
go mod tidy
```

Expected: `go.sum` is created with no errors.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum .gitignore .env.example internal/domain/.gitkeep internal/application/.gitkeep
git commit -m "chore: scaffold Go module with DDD directory structure"
```

---

## Task 2: Docker Compose

**Files:**
- Create: `docker-compose.yml`
- Create: `docker-compose.test.yml`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: taskdb
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

Save to `docker-compose.yml`.

- [ ] **Step 2: Create docker-compose.test.yml**

```yaml
services:
  db-test:
    image: postgres:16
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: taskdb
    ports:
      - "5433:5432"
```

Save to `docker-compose.test.yml`. No volume — ephemeral state for test isolation.

- [ ] **Step 3: Verify both compose files start cleanly**

```bash
docker compose up -d
docker compose ps
docker compose down

docker compose -f docker-compose.test.yml up -d
docker compose -f docker-compose.test.yml ps
docker compose -f docker-compose.test.yml down
```

Expected: both show Postgres containers as healthy, then stop cleanly.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml docker-compose.test.yml
git commit -m "chore: add Docker Compose for local dev and test Postgres"
```

---

## Task 3: Migrations

**Files:**
- Create: `migrations/000001_init.up.sql`
- Create: `migrations/000001_init.down.sql`

- [ ] **Step 1: Create up migration**

```sql
-- bootstrap: no schema changes yet
SELECT 1;
```

Save to `migrations/000001_init.up.sql`.

- [ ] **Step 2: Create down migration**

```sql
-- bootstrap rollback: no-op
SELECT 1;
```

Save to `migrations/000001_init.down.sql`.

- [ ] **Step 3: Commit**

```bash
git add migrations/
git commit -m "chore: add no-op bootstrap migration"
```

---

## Task 4: Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create Makefile**

```makefile
.PHONY: run test migrate-up migrate-down docker-up docker-down docker-test-up docker-test-down

ifneq (,$(wildcard .env))
  include .env
  export
endif

run:
	go run ./cmd/api

test:
	go test ./...

migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-test-up:
	docker compose -f docker-compose.test.yml up -d

docker-test-down:
	docker compose -f docker-compose.test.yml down
```

Note: indentation in Makefile rules must use **tabs**, not spaces.

- [ ] **Step 2: Verify make targets are parsed correctly**

```bash
make --dry-run docker-up
make --dry-run docker-down
make --dry-run test
```

Expected: prints the commands without executing them, no syntax errors.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile with run/test/migrate/docker targets"
```

---

## Task 5: Config (TDD)

**Files:**
- Create: `internal/infrastructure/config/config_test.go`
- Create: `internal/infrastructure/config/config.go`

- [ ] **Step 1: Write the failing tests**

Save to `internal/infrastructure/config/config_test.go`:

```go
package config_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/config"
)

func TestLoad_ErrorOnEmptyDBUrl(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is empty, got nil")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("PORT")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port=8080, got %s", cfg.Port)
	}
}

func TestLoad_DefaultEnv(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("ENV")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env=development, got %s", cfg.Env)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/infrastructure/config/...
```

Expected: FAIL — `config.Load` undefined.

- [ ] **Step 3: Implement config.go**

Save to `internal/infrastructure/config/config.go`:

```go
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl string
	Port  string
	Env   string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env") // ignore error: .env may not exist when env vars are injected externally

	cfg := Config{
		DBUrl: os.Getenv("DATABASE_URL"),
		Port:  os.Getenv("PORT"),
		Env:   os.Getenv("ENV"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.DBUrl == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/infrastructure/config/...
```

Expected: PASS — all 3 tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/config/
git commit -m "feat: add config loader with env var validation"
```

---

## Task 6: Database Connection (TDD)

**Files:**
- Create: `internal/infrastructure/persistence/postgres/db_test.go`
- Create: `internal/infrastructure/persistence/postgres/db.go`

- [ ] **Step 1: Write the failing tests**

Save to `internal/infrastructure/persistence/postgres/db_test.go`:

```go
package postgres_test

import (
	"os"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := postgres.Connect("not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestConnect_Success(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("expected successful connection, got: %v", err)
	}
	defer pool.Close()
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/infrastructure/persistence/postgres/...
```

Expected: FAIL — `postgres.Connect` undefined.

- [ ] **Step 3: Implement db.go**

Save to `internal/infrastructure/persistence/postgres/db.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(dbUrl string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 4: Run unit test (no DB needed)**

```bash
go test ./internal/infrastructure/persistence/postgres/... -run TestConnect_InvalidDSN
```

Expected: PASS.

- [ ] **Step 5: Start test DB and run integration test**

```bash
make docker-test-up
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  go test ./internal/infrastructure/persistence/postgres/... -run TestConnect_Success -v
```

Expected: PASS — pool opens and pings successfully.

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/persistence/postgres/
git commit -m "feat: add postgres connection pool with ping verification"
```

---

## Task 7: Health Handler (TDD)

**Files:**
- Create: `internal/infrastructure/http/handler/health_test.go`
- Create: `internal/infrastructure/http/handler/health.go`

- [ ] **Step 1: Write the failing tests**

Save to `internal/infrastructure/http/handler/health_test.go`:

```go
package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/handler"
)

type mockPinger struct {
	err error
}

func (m mockPinger) Ping(_ context.Context) error { return m.err }

func TestHealthHandler_OK(t *testing.T) {
	h := handler.NewHealthHandler(mockPinger{err: nil})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("expected body to contain status:ok, got %s", body)
	}
	if !strings.Contains(body, `"db":"ok"`) {
		t.Errorf("expected body to contain db:ok, got %s", body)
	}
}

func TestHealthHandler_DBUnreachable(t *testing.T) {
	h := handler.NewHealthHandler(mockPinger{err: errors.New("connection refused")})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"error"`) {
		t.Errorf("expected body to contain status:error, got %s", body)
	}
	if !strings.Contains(body, `"db":"unreachable"`) {
		t.Errorf("expected body to contain db:unreachable, got %s", body)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/infrastructure/http/handler/...
```

Expected: FAIL — `handler.NewHealthHandler` undefined.

- [ ] **Step 3: Implement health.go**

Save to `internal/infrastructure/http/handler/health.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DBPinger is satisfied by *pgxpool.Pool — defined as interface for testability.
type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db DBPinger
}

func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	if err := h.db.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "db": "unreachable"}) //nolint:errcheck
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "ok"}) //nolint:errcheck
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/infrastructure/http/handler/...
```

Expected: PASS — both tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/http/handler/
git commit -m "feat: add health handler with DBPinger interface"
```

---

## Task 8: Router

**Files:**
- Create: `internal/infrastructure/http/router.go`

- [ ] **Step 1: Create router.go**

Save to `internal/infrastructure/http/router.go`:

```go
package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/handler"
)

func Register(pool *pgxpool.Pool) nethttp.Handler {
	r := chi.NewRouter()

	h := handler.NewHealthHandler(pool)
	r.Get("/health", h.ServeHTTP)

	return r
}
```

Note: the package is named `http` (matching the directory convention). `net/http` is aliased as `nethttp` within this file to avoid the name collision.

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/http/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/router.go
git commit -m "feat: add chi router with /health route"
```

---

## Task 9: HTTP Server

**Files:**
- Create: `internal/infrastructure/http/server.go`

- [ ] **Step 1: Create server.go**

Save to `internal/infrastructure/http/server.go`:

```go
package http

import (
	"context"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func New(router nethttp.Handler, port string) *nethttp.Server {
	return &nethttp.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

// GracefulShutdown blocks until SIGINT or SIGTERM is received, then shuts
// down srv with a 10-second deadline for in-flight requests to complete.
func GracefulShutdown(srv *nethttp.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/infrastructure/http/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/http/server.go
git commit -m "feat: add HTTP server with timeouts and graceful shutdown"
```

---

## Task 10: Entry Point

**Files:**
- Create: `cmd/api/main.go`

- [ ] **Step 1: Create main.go**

Save to `cmd/api/main.go`:

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/lucaspalencia/superpowers/internal/infrastructure/config"
	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "port", cfg.Port, "env", cfg.Env)

	pool, err := postgres.Connect(cfg.DBUrl)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	router := infrahttp.Register(pool)
	srv := infrahttp.New(router, cfg.Port)
	slog.Info("server starting", "port", cfg.Port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	infrahttp.GracefulShutdown(srv)
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o api ./cmd/api
```

Expected: binary `api` is produced with no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat: add entry point wiring config, DB, router, and graceful shutdown"
```

---

## Task 11: Smoke Test (Integration)

**Files:**
- Create: `cmd/api/main_test.go`

- [ ] **Step 1: Write the failing smoke test**

Save to `cmd/api/main_test.go`:

```go
package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func TestSmokeTest_HealthEndpoint(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping smoke test")
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	router := infrahttp.Register(pool)
	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test — verify it skips (no DB env)**

```bash
go test ./cmd/api/... -v
```

Expected: `SKIP — TEST_DATABASE_URL not set`.

- [ ] **Step 3: Ensure test DB is up**

```bash
make docker-test-up
```

- [ ] **Step 4: Run integration smoke test**

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  go test ./cmd/api/... -run TestSmokeTest_HealthEndpoint -v
```

Expected: PASS — server starts, `/health` returns 200 within 500ms.

- [ ] **Step 5: Run full test suite**

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  go test ./...
```

Expected: all tests PASS (unit tests + integration tests with real DB).

- [ ] **Step 6: Commit**

```bash
git add cmd/api/main_test.go
git commit -m "test: add integration smoke test for /health endpoint"
```

---

## Task 12: Migrations Verification

**Files:** none — verification only.

- [ ] **Step 1: Ensure test DB is running**

```bash
make docker-test-up
```

- [ ] **Step 2: Run migrate-up against test DB**

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  make migrate-up
```

Expected output includes `1/u 000001_init` with no errors.

- [ ] **Step 3: Run migrate-down**

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5433/taskdb?sslmode=disable \
  make migrate-down
```

Expected output includes `1/d 000001_init` with no errors.

- [ ] **Step 4: Stop test DB**

```bash
make docker-test-down
```

- [ ] **Step 5: Final full build check**

```bash
go build ./...
go vet ./...
```

Expected: no errors, no warnings.
