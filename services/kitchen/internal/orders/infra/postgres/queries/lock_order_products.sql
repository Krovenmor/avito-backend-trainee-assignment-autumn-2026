SELECT id
FROM products
WHERE place_id = $1
  AND id = ANY ($2::uuid[])
  AND NOT is_deleted
ORDER BY id
FOR UPDATE;
