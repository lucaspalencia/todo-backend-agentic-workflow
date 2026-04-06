package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/lucaspalencia/superpowers/internal/domain/comment"
)

// CommentRepository implements domain/comment.Repository using PostgreSQL.
type CommentRepository struct {
	pool *pgxpool.Pool
}

// NewCommentRepository constructs a CommentRepository backed by the given pool.
func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

// Create inserts a comment and returns the persisted row.
func (r *CommentRepository) Create(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO comments (id, task_id, content, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, task_id, content, created_at`,
		c.ID, c.TaskID, c.Content, c.CreatedAt,
	)

	var out domain.Comment
	if err := row.Scan(&out.ID, &out.TaskID, &out.Content, &out.CreatedAt); err != nil {
		return domain.Comment{}, fmt.Errorf("scan comment: %w", err)
	}
	return out, nil
}

// ListByTaskID returns all comments for a task ordered by created_at ASC.
func (r *CommentRepository) ListByTaskID(ctx context.Context, taskID string) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, content, created_at
		 FROM comments
		 WHERE task_id = $1
		 ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return comments, nil
}

// DeleteByTaskID removes all comments for a task. No-op if none exist.
func (r *CommentRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM comments WHERE task_id = $1`,
		taskID,
	); err != nil {
		return fmt.Errorf("delete comments: %w", err)
	}
	return nil
}
