package comment

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a comment cannot be found.
var ErrNotFound = errors.New("comment not found")

// Repository defines the persistence contract for comments.
type Repository interface {
	Create(ctx context.Context, comment *Comment) error
	FindByTaskID(ctx context.Context, taskID string) ([]Comment, error)
}
