package comment

import "time"

type Comment struct {
	ID        string
	TaskID    string
	Content   string
	CreatedAt time.Time
}
