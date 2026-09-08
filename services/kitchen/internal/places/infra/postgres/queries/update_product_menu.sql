UPDATE products
SET name           = COALESCE($3, name),
    description    = COALESCE($4, description),
    price          = COALESCE($5, price),
    stock_quantity = stock_quantity + COALESCE($6, 0),
    updated_at     = now()
WHERE external_id = $1
  AND place_id = $2
RETURNING id;
