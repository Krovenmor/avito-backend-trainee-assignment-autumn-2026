-- +goose Up
CREATE TABLE order_ratings (
    id         UUID PRIMARY KEY,
    order_id   UUID NOT NULL UNIQUE REFERENCES orders (id),
    place_id   UUID NOT NULL REFERENCES places (id),
    user_id    UUID NOT NULL REFERENCES users (id),
    score      SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_ratings_place_id ON order_ratings (place_id, created_at DESC);

CREATE TABLE place_ratings (
    place_id     UUID PRIMARY KEY REFERENCES places (id),
    rating_sum   BIGINT NOT NULL DEFAULT 0 CHECK (rating_sum >= 0),
    rating_count INTEGER NOT NULL DEFAULT 0 CHECK (rating_count >= 0)
);

-- +goose Down
DROP TABLE order_ratings;

DROP TABLE place_ratings;
