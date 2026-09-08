-- +goose Up
CREATE TABLE orders (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users (id),
    place_id     UUID NOT NULL REFERENCES places (id),
    order_status TEXT NOT NULL,
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user_id ON orders (user_id, created_at DESC);

CREATE INDEX idx_orders_pending_created_at ON orders (created_at) WHERE order_status = 'pending';

-- +goose Down
DROP TABLE orders;
