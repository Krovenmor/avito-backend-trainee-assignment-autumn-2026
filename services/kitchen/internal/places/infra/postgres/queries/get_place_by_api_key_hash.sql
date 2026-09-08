SELECT id, is_active
FROM places
WHERE api_key_hash = $1;
