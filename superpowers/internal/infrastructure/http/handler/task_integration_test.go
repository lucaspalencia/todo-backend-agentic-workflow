package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	infrahttp "github.com/lucaspalencia/superpowers/internal/infrastructure/http"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

const integTestAPIKey = "test-api-key"

const createTasksSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
)`

var (
	integPool   *pgxpool.Pool
	integServer *httptest.Server
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// No DB — unit tests still run; integration tests skip individually.
		os.Exit(m.Run())
	}

	pool, err := postgres.Connect(dbURL)
	if err != nil {
		fmt.Printf("connect test db: %v\n", err)
		os.Exit(1)
	}

	if _, err = pool.Exec(context.Background(), createTasksSQL); err != nil {
		fmt.Printf("create tasks table: %v\n", err)
		pool.Close()
		os.Exit(1)
	}

	router := infrahttp.Register(pool, integTestAPIKey)
	srv := httptest.NewServer(router)

	integPool = pool
	integServer = srv

	code := m.Run()

	pool.Exec(context.Background(), "DROP TABLE IF EXISTS tasks") //nolint:errcheck
	srv.Close()
	pool.Close()

	os.Exit(code)
}

func TestCreateTask_Integration_Success(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	body := `{"title":"Buy groceries","description":"Milk and eggs"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["id"] == "" || result["id"] == nil {
		t.Error("expected non-empty id in response")
	}
	if result["title"] != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %v", result["title"])
	}
	if result["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", result["status"])
	}
	if result["created_at"] == "" || result["created_at"] == nil {
		t.Error("expected non-empty created_at in response")
	}
}

func TestCreateTask_Integration_ValidationError(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	body := `{"title":""}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+integTestAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errs, ok := result["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'errors' object in response body, got %v", result)
	}
	if _, ok := errs["title"]; !ok {
		t.Error("expected errors[\"title\"] to be set")
	}
}

func TestCreateTask_Integration_DuplicateSubmission(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	t.Cleanup(func() {
		integPool.Exec(context.Background(), "TRUNCATE tasks") //nolint:errcheck
	})

	body := `{"title":"Same title","description":"Same description"}`

	send := func() string {
		req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+integTestAPIKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /tasks: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		id, _ := result["id"].(string)
		return id
	}

	id1 := send()
	id2 := send()

	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty IDs from both requests")
	}
	if id1 == id2 {
		t.Errorf("expected different IDs for duplicate submission, both got %q", id1)
	}
}

func TestCreateTask_Integration_Unauthorized(t *testing.T) {
	if integServer == nil {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	body := `{"title":"Buy groceries"}`
	req, _ := http.NewRequest(http.MethodPost, integServer.URL+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
