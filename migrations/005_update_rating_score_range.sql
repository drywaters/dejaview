-- +goose Up
-- +goose StatementBegin
ALTER TABLE ratings
    DROP CONSTRAINT IF EXISTS ratings_score_check;

ALTER TABLE ratings
    ADD CONSTRAINT ratings_score_check CHECK (score >= 0.0 AND score <= 10.0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ratings
    DROP CONSTRAINT IF EXISTS ratings_score_check;

ALTER TABLE ratings
    ADD CONSTRAINT ratings_score_check CHECK (score >= 1.0 AND score <= 10.0);
-- +goose StatementEnd
