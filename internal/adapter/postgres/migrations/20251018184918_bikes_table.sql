-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS bikes (
    bike_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    bike_name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('bmx', 'mtb', 'road')),
    model VARCHAR(255),
    year INT NOT NULL,
    mileage INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bikes_user_id ON bikes(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bikes;
-- +goose StatementEnd
