-- +goose Up
-- +goose StatementBegin
-- Ensure each position is unique within a group
ALTER TABLE entries
    ADD CONSTRAINT entries_group_position_unique UNIQUE (group_number, position);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entries
    DROP CONSTRAINT IF EXISTS entries_group_position_unique;
-- +goose StatementEnd
