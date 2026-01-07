-- +goose Up
-- +goose StatementBegin

-- Add position column to entries table for custom ordering within groups
ALTER TABLE entries ADD COLUMN position INTEGER;

-- Initialize positions based on current added_at order (descending - newest first gets position 1)
WITH ranked AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY group_number ORDER BY added_at DESC) as rn
    FROM entries
)
UPDATE entries SET position = ranked.rn
FROM ranked WHERE entries.id = ranked.id;

-- Make position NOT NULL after populating existing data
ALTER TABLE entries ALTER COLUMN position SET NOT NULL;

-- Add index for efficient ordering within groups
CREATE INDEX idx_entries_group_position ON entries(group_number, position);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_entries_group_position;
ALTER TABLE entries DROP COLUMN IF EXISTS position;
-- +goose StatementEnd
