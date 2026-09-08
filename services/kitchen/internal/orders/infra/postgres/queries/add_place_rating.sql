INSERT INTO place_ratings (place_id, rating_sum, rating_count)
VALUES ($1, $2, 1)
ON CONFLICT (place_id) DO UPDATE
    SET rating_sum   = place_ratings.rating_sum + EXCLUDED.rating_sum,
        rating_count = place_ratings.rating_count + 1;
