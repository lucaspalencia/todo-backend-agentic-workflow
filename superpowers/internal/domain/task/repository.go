package task

import "context"

type Repository interface {
	Create(ctx context.Context, task Task) (Task, error)
}
