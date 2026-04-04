package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lucaspalencia/todo-backend/internal/domain/task"
)

// TaskRepository implements task.Repository using pgxpool.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository constructs a TaskRepository with the given pool.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, t *task.Task) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return task.ErrDuplicateTitle
		}
		return err
	}
	return nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*task.Task, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, title, description, status, created_at, updated_at, deleted_at FROM tasks WHERE id = $1 AND deleted_at IS NULL`, id)

	t := &task.Task{}
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) FindAll(ctx context.Context) ([]task.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, description, status, created_at, updated_at, deleted_at FROM tasks WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) Save(ctx context.Context, t *task.Task) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET title=$1, description=$2, status=$3, updated_at=$4 WHERE id=$5`,
		t.Title, t.Description, t.Status, t.UpdatedAt, t.ID,
	)
	return err
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tasks SET deleted_at = now() WHERE id = $1`, id)
	return err
}
