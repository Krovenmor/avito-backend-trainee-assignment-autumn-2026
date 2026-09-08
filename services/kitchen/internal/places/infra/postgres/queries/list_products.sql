SELECT id AS product_id, name, description, price, stock_quantity
FROM products
WHERE place_id = $1
  AND NOT is_deleted
  AND ($2::text IS NULL OR name ILIKE '%' || $2 || '%')
ORDER BY name
LIMIT $3 OFFSET $4;
