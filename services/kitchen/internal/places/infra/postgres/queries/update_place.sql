UPDATE places
SET name        = COALESCE($2, name),
    description = COALESCE($3, description),
    city        = COALESCE($4, city),
    address     = COALESCE($5, address),
    timezone    = COALESCE($6, timezone),
    opens_at    = COALESCE($7, opens_at),
    closes_at   = COALESCE($8, closes_at),
    place_type  = COALESCE($9, place_type),
    webhook_url = COALESCE($10, webhook_url),
    is_active   = COALESCE($11, is_active)
WHERE id = $1
RETURNING id;