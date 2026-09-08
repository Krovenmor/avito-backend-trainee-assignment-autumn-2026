SELECT webhook_url,
       is_active,
       CASE
           WHEN opens_at <= closes_at
               THEN (now() AT TIME ZONE timezone)::time >= opens_at
                   AND (now() AT TIME ZONE timezone)::time < closes_at
           ELSE (now() AT TIME ZONE timezone)::time >= opens_at
               OR (now() AT TIME ZONE timezone)::time < closes_at
           END AS is_open
FROM places
WHERE id = $1;
