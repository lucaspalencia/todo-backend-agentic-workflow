package http

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps http.Server with graceful shutdown logic.
type Server struct {
	inner *http.Server
}

// New constructs a Server on the given port with sensible timeouts.
func New(router http.Handler, port string) *Server {
	return &Server{
		inner: &http.Server{
			Addr:         ":" + port,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start begins listening and blocks until a SIGINT or SIGTERM is received,
// then drains in-flight requests with a 10-second deadline.
func (s *Server) Start() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", s.inner.Addr)
		if err := s.inner.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.inner.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("server stopped gracefully")
	return nil
}
