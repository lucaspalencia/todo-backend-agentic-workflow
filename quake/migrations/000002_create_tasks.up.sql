CREATE TABLE tasks (
    id          UUID PRIMARY KEY,
    title       VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(2000) NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
