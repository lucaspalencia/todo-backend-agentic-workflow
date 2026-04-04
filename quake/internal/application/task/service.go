package task

import (
	"github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// Service orchestrates task use cases. It depends only on the domain repository
// interface — no infrastructure imports.
type Service struct {
	repo task.Repository
}

// NewService constructs a Service with the given repository implementation.
func NewService(repo task.Repository) *Service {
	return &Service{repo: repo}
}
