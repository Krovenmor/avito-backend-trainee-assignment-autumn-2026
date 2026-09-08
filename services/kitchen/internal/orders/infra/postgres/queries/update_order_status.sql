UPDATE orders
SET order_status = $2,
    updated_at   = now()
WHERE id = $1
  AND order_status = 'pending';
