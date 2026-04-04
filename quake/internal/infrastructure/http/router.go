package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/lucaspalencia/todo-backend/internal/infrastructure/http/handler"
	appmiddleware "github.com/lucaspalencia/todo-backend/internal/infrastructure/http/middleware"
)

// Register builds and returns the chi router with all routes mounted.
func Register(pool handler.Pinger, taskSvc handler.TaskCreator, apiKey string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	healthHandler := handler.NewHealthHandler(pool)
	r.Get("/health", healthHandler.Check)

	taskHandler := handler.NewTaskHandler(taskSvc)
	r.With(appmiddleware.APIKeyAuth(apiKey)).Post("/tasks", taskHandler.Create)

	return r
}
