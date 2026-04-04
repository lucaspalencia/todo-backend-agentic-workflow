package comment

import "time"

// Comment represents a comment attached to a task.
type Comment struct {
	ID        string
	TaskID    string
	Content   string
	CreatedAt time.Time
}
