-- +goose Up
ALTER TABLE orders
    ADD CONSTRAINT orders_status_known
        CHECK (order_status IN ('pending', 'accepted', 'rejected', 'completed', 'cancelled', 'expired'));

-- +goose Down
ALTER TABLE orders
    DROP CONSTRAINT orders_status_known;
