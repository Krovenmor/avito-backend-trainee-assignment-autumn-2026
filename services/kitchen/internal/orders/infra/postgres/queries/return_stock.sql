WITH locked AS MATERIALIZED (SELECT p.id
                             FROM products p
                             WHERE p.id IN (SELECT product_id FROM order_items WHERE order_id = $1)
                             ORDER BY p.id
                             FOR UPDATE),
     requested AS (SELECT product_id, SUM(quantity) AS quantity
                   FROM order_items
                   WHERE order_id = $1
                   GROUP BY product_id)
UPDATE products p
SET stock_quantity = p.stock_quantity + r.quantity,
    updated_at     = now()
FROM requested r
WHERE p.id = r.product_id
  AND p.id IN (SELECT id FROM locked)
  AND NOT (p.external_id = ANY ($2::text[]));
