package task

import (
	"context"
	"errors"
)

// ErrNotFound is returned by the repository when no task matches the given ID.
var ErrNotFound = errors.New("task not found")

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
	GetByID(ctx context.Context, id string) (Task, error)
	Update(ctx context.Context, task Task) (Task, error)
}
