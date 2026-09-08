-- +goose Up
CREATE TABLE places (
    id           UUID PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT,
    city         TEXT NOT NULL,
    address      TEXT NOT NULL,
    timezone     TEXT NOT NULL,
    opens_at     TIME NOT NULL,
    closes_at    TIME NOT NULL,
    place_type   TEXT NOT NULL,
    webhook_url  TEXT NOT NULL,
    api_key_hash TEXT NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_places_city_type ON places (city, place_type) WHERE is_active;

-- +goose Down
DROP TABLE places;