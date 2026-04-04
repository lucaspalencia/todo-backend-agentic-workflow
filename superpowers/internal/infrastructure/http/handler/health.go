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
