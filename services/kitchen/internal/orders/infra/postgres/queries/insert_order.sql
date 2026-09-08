WITH priced AS (SELECT u.id, u.product_id, u.quantity, p.price
                FROM unnest($4::uuid[], $5::uuid[], $6::integer[]) AS u (id, product_id, quantity)
                         JOIN products p ON p.id = u.product_id),
     new_order AS (
         INSERT INTO orders (id, user_id, place_id, order_status, total_amount, locked_at, locked_by)
             SELECT $1, $2, $3, 'pending', COALESCE(SUM(quantity * price), 0), now(), $7
             FROM priced
             RETURNING total_amount, created_at),
     new_items AS (
         INSERT INTO order_items (id, order_id, product_id, quantity, price)
             SELECT pr.id, $1, pr.product_id, pr.quantity, pr.price
             FROM priced pr)
SELECT total_amount, created_at
FROM new_order;
