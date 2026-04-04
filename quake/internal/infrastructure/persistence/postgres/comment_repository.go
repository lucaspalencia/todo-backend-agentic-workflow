package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	domcomment "github.com/lucaspalencia/todo-backend/internal/domain/comment"
)

// CommentRepository implements comment.Repository using pgxpool.
type CommentRepository struct {
	pool *pgxpool.Pool
}

// NewCommentRepository constructs a CommentRepository with the given pool.
func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) Create(ctx context.Context, c *domcomment.Comment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO comments (id, task_id, content, created_at) VALUES ($1, $2, $3, $4)`,
		c.ID, c.TaskID, c.Content, c.CreatedAt,
	)
	return err
}

func (r *CommentRepository) FindByTaskID(ctx context.Context, taskID string) ([]domcomment.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, content, created_at FROM comments WHERE task_id = $1 ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []domcomment.Comment
	for rows.Next() {
		var c domcomment.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
