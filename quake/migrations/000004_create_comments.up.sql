CREATE TABLE comments (
    id         UUID PRIMARY KEY,
    task_id    UUID         NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    VARCHAR(2000) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
