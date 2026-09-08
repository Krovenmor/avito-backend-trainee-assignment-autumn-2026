INSERT INTO order_ratings (id, order_id, place_id, user_id, score)
SELECT $1, o.id, o.place_id, o.user_id, $4
FROM orders o
WHERE o.id = $2
  AND o.user_id = $3
  AND o.order_status = 'completed'
ON CONFLICT (order_id) DO NOTHING
RETURNING place_id;
