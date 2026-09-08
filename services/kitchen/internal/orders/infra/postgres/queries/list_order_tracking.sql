SELECT source, external_code, status, occurred_at
FROM order_tracking
WHERE order_id = $1
ORDER BY occurred_at, received_at;
