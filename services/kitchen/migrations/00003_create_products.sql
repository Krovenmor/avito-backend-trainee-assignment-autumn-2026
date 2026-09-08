-- +goose Up
CREATE TABLE products (
    id             UUID PRIMARY KEY,
    external_id    TEXT NOT NULL,
    place_id       UUID NOT NULL REFERENCES places (id),
    name           TEXT NOT NULL,
    description    TEXT,
    price          NUMERIC(12, 2) NOT NULL CONSTRAINT products_price_non_negative CHECK (price >= 0),
    stock_quantity INTEGER NOT NULL DEFAULT 0 CONSTRAINT products_stock_non_negative CHECK (stock_quantity >= 0),
    is_deleted     BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (place_id, external_id)
);

CREATE INDEX idx_products_place_id ON products (place_id) WHERE NOT is_deleted;

-- +goose Down
DROP TABLE products;
