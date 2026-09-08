WITH locked AS MATERIALIZED (SELECT p.id
                             FROM products p
                             WHERE p.id IN (SELECT product_id FROM order_items WHERE order_id = $1)
                             ORDER BY p.id
                             FOR UPDATE)
UPDATE products p
SET stock_quantity = 0,
    updated_at     = now()
FROM order_items i
WHERE i.order_id = $1
  AND p.id = i.product_id
  AND p.id IN (SELECT id FROM locked)
  AND p.external_id = ANY ($2::text[])
RETURNING p.external_id;
