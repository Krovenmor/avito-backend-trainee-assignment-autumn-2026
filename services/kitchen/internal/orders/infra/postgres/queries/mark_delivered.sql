UPDATE orders
SET order_status = 'completed',
    updated_at   = now()
WHERE id = $1
  AND place_id = $2
  AND order_status = 'accepted'
RETURNING user_id;
