UPDATE orders
SET locked_at = now()
WHERE id = ANY ($1::uuid[])
  AND locked_by = $2
  AND order_status = 'pending';
