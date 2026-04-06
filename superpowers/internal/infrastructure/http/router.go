package http

import (
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appcomment "github.com/lucaspalencia/superpowers/internal/application/comment"
	apptask "github.com/lucaspalencia/superpowers/internal/application/task"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/handler"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/http/middleware"
	"github.com/lucaspalencia/superpowers/internal/infrastructure/persistence/postgres"
)

func Register(pool *pgxpool.Pool, apiKey string) nethttp.Handler {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler(pool)
	r.Get("/health", healthHandler.ServeHTTP)

	taskRepo := postgres.NewTaskRepository(pool)
	commentRepo := postgres.NewCommentRepository(pool)

	taskSvc := apptask.NewService(taskRepo)
	taskSvc.WithCommentDeleter(commentRepo)
	taskHandler := handler.NewTaskHandler(taskSvc)

	commentSvc := appcomment.NewService(commentRepo, taskRepo)
	commentHandler := handler.NewCommentHandler(commentSvc)

	r.With(middleware.APIKey(apiKey)).Post("/tasks", taskHandler.Create)
	r.With(middleware.APIKey(apiKey)).Get("/tasks", taskHandler.List)
	r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}", taskHandler.GetByID)
	r.With(middleware.APIKey(apiKey)).Patch("/tasks/{id}", taskHandler.Update)
	r.With(middleware.APIKey(apiKey)).Delete("/tasks/{id}", taskHandler.Delete)

	r.With(middleware.APIKey(apiKey)).Post("/tasks/{id}/comments", commentHandler.Create)
	r.With(middleware.APIKey(apiKey)).Get("/tasks/{id}/comments", commentHandler.List)

	return r
}
