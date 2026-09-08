-- +goose Up
CREATE TABLE order_items (
    id         UUID PRIMARY KEY,
    order_id   UUID NOT NULL REFERENCES orders (id),
    product_id UUID NOT NULL REFERENCES products (id),
    quantity   INTEGER NOT NULL CHECK (quantity > 0),
    price      NUMERIC(12, 2) NOT NULL CHECK (price >= 0)
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);

-- +goose Down
DROP TABLE order_items;