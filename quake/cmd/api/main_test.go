package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
	domtask "github.com/lucaspalencia/todo-backend/internal/domain/task"
	infrahttp "github.com/lucaspalencia/todo-backend/internal/infrastructure/http"
)

// okPinger satisfies handler.Pinger without a real DB.
type okPinger struct{}

func (p *okPinger) Ping(_ context.Context) error { return nil }

// noopTaskRepo satisfies task.Repository without any persistence.
type noopTaskRepo struct{}

func (r *noopTaskRepo) Create(_ context.Context, _ *domtask.Task) error             { return nil }
func (r *noopTaskRepo) FindByID(_ context.Context, _ string) (*domtask.Task, error) { return nil, nil }
func (r *noopTaskRepo) FindAll(_ context.Context) ([]domtask.Task, error)           { return nil, nil }
func (r *noopTaskRepo) Save(_ context.Context, _ *domtask.Task) error               { return nil }
func (r *noopTaskRepo) Delete(_ context.Context, _ string) error                    { return nil }

// noopCommentRepo satisfies comment.Repository without any persistence.
type noopCommentRepo struct{}

func (r *noopCommentRepo) Create(_ context.Context, _ *domcomment.Comment) error          { return nil }
func (r *noopCommentRepo) FindByTaskID(_ context.Context, _ string) ([]domcomment.Comment, error) {
	return nil, nil
}

func TestHealthEndpoint_Smoke(t *testing.T) {
	taskRepo := &noopTaskRepo{}
	taskSvc := apptask.NewService(taskRepo)
	commentSvc := appcomment.NewService(taskRepo, &noopCommentRepo{})
	router := infrahttp.Register(&okPinger{}, taskSvc, commentSvc, "test-key")
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
