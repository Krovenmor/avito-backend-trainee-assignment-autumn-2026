SELECT p.id,
       p.name,
       p.description,
       p.city,
       p.address,
       p.timezone,
       p.opens_at,
       p.closes_at,
       p.place_type,
       CASE
           WHEN p.opens_at <= p.closes_at
               THEN (now() AT TIME ZONE p.timezone)::time >= p.opens_at
                   AND (now() AT TIME ZONE p.timezone)::time < p.closes_at
           ELSE (now() AT TIME ZONE p.timezone)::time >= p.opens_at
               OR (now() AT TIME ZONE p.timezone)::time < p.closes_at
           END AS is_open,
       CASE WHEN r.rating_count > 0 THEN round(r.rating_sum::numeric / r.rating_count, 2) END AS rating,
       COALESCE(r.rating_count, 0)                                                            AS rating_count
FROM places p
         LEFT JOIN place_ratings r ON r.place_id = p.id
WHERE p.is_active
  AND ($1::text IS NULL OR p.city = $1)
  AND ($2::text IS NULL OR p.place_type = $2)
  AND ($3::text IS NULL OR p.name ILIKE '%' || $3 || '%')
  AND ($4::numeric IS NULL OR (r.rating_count > 0 AND r.rating_sum::numeric / r.rating_count >= $4))
ORDER BY CASE WHEN $5::bool THEN COALESCE(r.rating_sum::numeric / NULLIF(r.rating_count, 0), 0) END DESC,
         p.created_at DESC
LIMIT $6 OFFSET $7;
