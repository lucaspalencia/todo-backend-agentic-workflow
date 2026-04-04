package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
	infrahttp "github.com/lucaspalencia/todo-backend/internal/infrastructure/http"
)

// okPinger satisfies handler.Pinger without a real DB.
type okPinger struct{}

func (p *okPinger) Ping(_ context.Context) error { return nil }

// noopRepo satisfies task.Repository without any persistence.
type noopRepo struct{}

func (r *noopRepo) Create(_ context.Context, _ *domtask.Task) error        { return nil }
func (r *noopRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) { return nil, nil }
func (r *noopRepo) FindAll(_ context.Context) ([]domtask.Task, error)      { return nil, nil }
func (r *noopRepo) Save(_ context.Context, _ *domtask.Task) error          { return nil }
func (r *noopRepo) Delete(_ context.Context, _ string) error               { return nil }

func TestHealthEndpoint_Smoke(t *testing.T) {
	taskSvc := apptask.NewService(&noopRepo{})
	router := infrahttp.Register(&okPinger{}, taskSvc, "test-key")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
