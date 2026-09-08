UPDATE products
SET is_deleted = true,
    updated_at = now()
WHERE external_id = $1
  AND place_id = $2
RETURNING id;