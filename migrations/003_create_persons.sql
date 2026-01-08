-- +goose Up
-- +goose StatementBegin
CREATE TABLE persons (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initial     CHAR(1) NOT NULL UNIQUE,
    name        TEXT NOT NULL
);

-- Seed the Waters family
INSERT INTO persons (initial, name) VALUES
    ('D', 'Daniel'),
    ('J', 'Jennifer'),
    ('C', 'Caleb'),
    ('A', 'Aiden');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS persons;
-- +goose StatementEnd


