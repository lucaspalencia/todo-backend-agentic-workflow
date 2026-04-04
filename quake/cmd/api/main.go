package main

import (
	"log/slog"
	"os"

	appcomment "github.com/lucaspalencia/todo-backend/internal/application/comment"
	apptask "github.com/lucaspalencia/todo-backend/internal/application/task"
	"github.com/lucaspalencia/todo-backend/internal/infrastructure/config"
	infrahttp "github.com/lucaspalencia/todo-backend/internal/infrastructure/http"
	"github.com/lucaspalencia/todo-backend/internal/infrastructure/persistence/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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

	taskRepo := postgres.NewTaskRepository(pool)
	taskSvc := apptask.NewService(taskRepo)

	commentRepo := postgres.NewCommentRepository(pool)
	commentSvc := appcomment.NewService(taskRepo, commentRepo)

	router := infrahttp.Register(pool, taskSvc, commentSvc, cfg.APIKey)
	server := infrahttp.New(router, cfg.Port)

	if err := server.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
