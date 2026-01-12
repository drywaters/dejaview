-- +goose Up
-- +goose StatementBegin
ALTER TABLE entries DROP COLUMN IF EXISTS notes;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entries ADD COLUMN notes TEXT;
-- +goose StatementEnd
