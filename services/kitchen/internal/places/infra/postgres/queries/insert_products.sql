INSERT INTO products (id, place_id, external_id, name, description, price, stock_quantity)
SELECT u.id, $1, u.external_id, u.name, u.description, u.price, u.stock_quantity
FROM unnest($2::uuid[], $3::text[], $4::text[], $5::text[], $6::numeric[], $7::integer[])
    AS u (id, external_id, name, description, price, stock_quantity);
