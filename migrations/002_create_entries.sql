-- +goose Up
-- +goose StatementBegin
CREATE TABLE entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movie_id        UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    group_number    INTEGER NOT NULL DEFAULT 1,
    added_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes           TEXT
);

-- Index for listing entries by group
CREATE INDEX idx_entries_group_number ON entries(group_number);

-- Index for movie lookups
CREATE INDEX idx_entries_movie_id ON entries(movie_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS entries;
-- +goose StatementEnd


