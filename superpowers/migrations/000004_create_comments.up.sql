CREATE TABLE comments (
    id         TEXT        PRIMARY KEY,
    task_id    TEXT        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
