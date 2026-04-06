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
	// Delete soft-deletes a task. Returns ErrNotFound if the task does not
	// exist or has already been deleted.
	Delete(ctx context.Context, id string) error
	// List returns all active (non-deleted) tasks ordered by created_at DESC.
	List(ctx context.Context) ([]Task, error)
}
