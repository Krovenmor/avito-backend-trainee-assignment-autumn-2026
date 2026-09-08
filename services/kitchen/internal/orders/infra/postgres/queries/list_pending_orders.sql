WITH claimed AS (SELECT id
                 FROM orders
                 WHERE order_status = 'pending'
                   AND NOT (id = ANY ($4::uuid[]))
                   AND (locked_by = $1
                     OR locked_at IS NULL
                     OR locked_at < now() - make_interval(secs => $2))
                 ORDER BY created_at
                 LIMIT $3
                 FOR UPDATE SKIP LOCKED),
     locked AS (
         UPDATE orders o
             SET locked_at = now(),
                 locked_by = $1
             FROM claimed c
             WHERE o.id = c.id
             RETURNING o.id, o.user_id, o.place_id, o.created_at)
SELECT l.id, l.user_id, l.place_id, l.created_at, p.webhook_url
FROM locked l
         JOIN places p ON p.id = l.place_id
ORDER BY l.created_at;
