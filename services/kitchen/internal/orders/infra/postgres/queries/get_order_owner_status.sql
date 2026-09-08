SELECT user_id, order_status
FROM orders
WHERE id = $1;
