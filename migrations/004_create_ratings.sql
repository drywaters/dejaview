-- +goose Up
-- +goose StatementBegin
CREATE TABLE ratings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL REFERENCES persons(id),
    entry_id    UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    score       DECIMAL(3,1) NOT NULL CHECK (score >= 1.0 AND score <= 10.0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(person_id, entry_id)
);

-- Index for looking up ratings by entry
CREATE INDEX idx_ratings_entry_id ON ratings(entry_id);

-- Index for looking up ratings by person
CREATE INDEX idx_ratings_person_id ON ratings(person_id);

-- Trigger to auto-update updated_at
CREATE TRIGGER update_ratings_updated_at
    BEFORE UPDATE ON ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_ratings_updated_at ON ratings;
DROP TABLE IF EXISTS ratings;
-- +goose StatementEnd


