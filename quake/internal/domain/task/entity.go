package task

import "time"

// Task is the core domain entity for a task management item.
type Task struct {
	ID          string
	Title       string
	Description string
	Done        bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
