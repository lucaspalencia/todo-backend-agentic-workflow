package task

import "context"

// Repository defines the persistence contract for Task entities.
// Implementations live in the infrastructure layer.
type Repository interface {
	FindByID(ctx context.Context, id string) (*Task, error)
	FindAll(ctx context.Context) ([]Task, error)
	Save(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
}
