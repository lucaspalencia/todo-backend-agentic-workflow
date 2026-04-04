package task

import (
	"context"
	"errors"
)

// ErrDuplicateTitle is returned when a task with the same title already exists.
var ErrDuplicateTitle = errors.New("a task with this title already exists")

// Repository defines the persistence contract for Task entities.
// Implementations live in the infrastructure layer.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id string) (*Task, error)
	FindAll(ctx context.Context) ([]Task, error)
	Save(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
}
