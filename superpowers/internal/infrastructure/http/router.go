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
