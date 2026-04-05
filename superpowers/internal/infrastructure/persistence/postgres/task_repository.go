package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// GetByID fetches a single active task by its ID.
// Returns domain.ErrNotFound if no row matches or the task is soft-deleted.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, domain.ErrNotFound
		}
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}

// Update writes the full task back to the database and returns the persisted row.
func (r *TaskRepository) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE tasks
		 SET title = $1, description = $2, status = $3, updated_at = $4
		 WHERE id = $5
		 RETURNING id, title, description, status, created_at, updated_at`,
		task.Title, task.Description, task.Status, task.UpdatedAt, task.ID,
	)

	var t domain.Task
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	return t, nil
}

// Delete soft-deletes a task by setting deleted_at to the current time.
// Returns domain.ErrNotFound if the task does not exist or is already deleted.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// List returns all active (non-deleted) tasks ordered by created_at DESC.
func (r *TaskRepository) List(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, description, status, created_at, updated_at
		 FROM tasks
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}
