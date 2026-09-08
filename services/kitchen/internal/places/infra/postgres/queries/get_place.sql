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
WHERE p.id = $1
  AND p.is_active;
