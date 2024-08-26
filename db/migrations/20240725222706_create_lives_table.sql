-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE lives (
                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                       title VARCHAR(255) NOT NULL,
                       link VARCHAR(255) NOT NULL,
                       created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS lives;
