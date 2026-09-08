-- +goose Up
CREATE TABLE order_tracking (
    id            UUID PRIMARY KEY,
    order_id      UUID NOT NULL REFERENCES orders (id),
    source        TEXT NOT NULL CHECK (source IN ('place', 'delivery')),
    external_code TEXT NOT NULL,
    status        TEXT NOT NULL CHECK (status IN ('accepted', 'cooking', 'ready', 'handed_over')),
    occurred_at   TIMESTAMPTZ NOT NULL,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_order_tracking_order_id ON order_tracking (order_id, occurred_at);

-- +goose Down
DROP TABLE order_tracking;
