WITH requested AS (SELECT product_id, SUM(quantity)::int AS quantity
                   FROM unnest($1::uuid[], $2::int[]) AS r(product_id, quantity)
                   GROUP BY product_id),
     reserved AS (
         UPDATE products p
             SET stock_quantity = p.stock_quantity - r.quantity,
                 updated_at = now()
             FROM requested r
             WHERE p.id = r.product_id
                 AND p.stock_quantity >= r.quantity
             RETURNING p.id)
SELECT r.product_id, r.quantity AS requested, p.stock_quantity AS available
FROM requested r
         JOIN products p ON p.id = r.product_id
WHERE r.product_id NOT IN (SELECT id FROM reserved)
ORDER BY r.product_id;
