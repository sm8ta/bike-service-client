-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY,
    bike_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL CHECK (name IN ('handlebars', 'frame', 'wheels')),
    brand VARCHAR(100),
    model VARCHAR(100),
    installed_at TIMESTAMP NOT NULL,
    installed_mileage INT NOT NULL DEFAULT 0,
    max_mileage INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_bike FOREIGN KEY (bike_id) REFERENCES bikes(bike_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS components;
-- +goose StatementEnd
