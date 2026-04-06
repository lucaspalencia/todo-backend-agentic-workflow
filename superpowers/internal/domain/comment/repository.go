package comment

import (
	"context"
	"errors"
)

// ErrNotFound is returned when no comment matches the given criteria.
var ErrNotFound = errors.New("comment not found")

type Repository interface {
	Create(ctx context.Context, c Comment) (Comment, error)
	ListByTaskID(ctx context.Context, taskID string) ([]Comment, error)
	// DeleteByTaskID removes all comments for the given task. No-op if none exist.
	DeleteByTaskID(ctx context.Context, taskID string) error
}
