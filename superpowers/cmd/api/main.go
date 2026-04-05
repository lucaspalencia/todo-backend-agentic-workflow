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

	router := infrahttp.Register(pool, cfg.APIKey)
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
