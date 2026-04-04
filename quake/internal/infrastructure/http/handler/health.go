package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger is the minimal interface the health handler needs from the DB pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler handles the GET /health endpoint.
type HealthHandler struct {
	pool Pinger
}

// NewHealthHandler constructs a HealthHandler with the given DB pool.
func NewHealthHandler(pool Pinger) *HealthHandler {
	return &HealthHandler{pool: pool}
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// Check responds with the server and database health status.
// Returns 200 when healthy, 503 when the DB is unreachable.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp := healthResponse{Status: "ok", DB: "ok"}
	statusCode := http.StatusOK

	if err := h.pool.Ping(ctx); err != nil {
		resp.Status = "error"
		resp.DB = "unreachable"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}
