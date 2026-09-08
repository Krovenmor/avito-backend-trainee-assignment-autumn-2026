SELECT order_status
FROM orders
WHERE id = $1
  AND place_id = $2;
