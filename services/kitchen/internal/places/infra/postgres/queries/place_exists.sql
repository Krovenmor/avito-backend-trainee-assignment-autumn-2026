SELECT id
FROM places
WHERE id = $1
  AND is_active;
