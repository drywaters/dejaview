-- +goose Up
-- +goose StatementBegin
ALTER TABLE entries DROP COLUMN IF EXISTS watched_at;
DROP INDEX IF EXISTS idx_entries_watched_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entries ADD COLUMN watched_at DATE;
CREATE INDEX idx_entries_watched_at ON entries(watched_at) WHERE watched_at IS NOT NULL;
-- +goose StatementEnd
