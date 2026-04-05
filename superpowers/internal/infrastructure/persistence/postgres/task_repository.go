package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/lucaspalencia/superpowers/internal/domain/task"
)

// TaskRepository implements domain/task.Repository using PostgreSQL.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository constructs a TaskRepository backed by the given pool.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// Create inserts a task and returns the persisted row.
func (r *TaskRepository) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, title, description, status, created_at, updated_at`,
		task.ID, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}
