UPDATE orders
SET order_status = 'cancelled',
    updated_at   = now()
WHERE id = $1
  AND user_id = $2
  AND order_status = 'pending';
