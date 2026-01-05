-- +goose Up
-- +goose StatementBegin
ALTER TABLE entries
    ADD CONSTRAINT entries_movie_group_unique UNIQUE (movie_id, group_number);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entries
    DROP CONSTRAINT IF EXISTS entries_movie_group_unique;
-- +goose StatementEnd
