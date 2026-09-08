WITH target AS (SELECT id, user_id
                FROM orders
                WHERE id = $2
                  AND place_id = $3),
     inserted AS (
         INSERT INTO order_tracking (id, order_id, source, external_code, status, occurred_at)
             SELECT $1, t.id, $4, $5, $6, $7
             FROM target t
             RETURNING order_id)
SELECT t.user_id
FROM target t
         JOIN inserted i ON i.order_id = t.id;
