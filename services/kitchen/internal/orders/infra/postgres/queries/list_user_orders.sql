SELECT o.id,
       o.place_id,
       p.name AS place_name,
       o.order_status,
       o.total_amount,
       o.created_at
FROM orders o
         JOIN places p ON p.id = o.place_id
WHERE o.user_id = $1
ORDER BY o.created_at DESC
LIMIT $2 OFFSET $3;
