-- +goose Up
ALTER TABLE orders
    ADD COLUMN locked_at TIMESTAMPTZ,
    ADD COLUMN locked_by TEXT;

-- +goose Down
ALTER TABLE orders
    DROP COLUMN locked_at,
    DROP COLUMN locked_by;
