-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    hashed_password TEXT DEFAULT 'unset' NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;