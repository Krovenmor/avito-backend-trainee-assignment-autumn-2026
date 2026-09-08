SELECT p.id             AS product_id,
       p.name,
       p.description,
       p.price,
       p.stock_quantity,
       pl.id           AS place_id,
       pl.name         AS place_name,
       pl.city         AS place_city,
       pl.place_type   AS place_type,
       CASE
           WHEN pl.opens_at <= pl.closes_at
               THEN (now() AT TIME ZONE pl.timezone)::time >= pl.opens_at
                   AND (now() AT TIME ZONE pl.timezone)::time < pl.closes_at
           ELSE (now() AT TIME ZONE pl.timezone)::time >= pl.opens_at
               OR (now() AT TIME ZONE pl.timezone)::time < pl.closes_at
           END AS place_is_open
FROM products p
         JOIN places pl ON pl.id = p.place_id
WHERE pl.is_active
  AND NOT p.is_deleted
  AND ($1::text IS NULL OR pl.city = $1)
  AND ($2::text IS NULL OR pl.place_type = $2)
  AND ($3::text IS NULL OR p.name ILIKE '%' || $3 || '%')
ORDER BY p.name, pl.name
LIMIT $4 OFFSET $5;
