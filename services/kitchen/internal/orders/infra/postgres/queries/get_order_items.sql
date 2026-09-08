SELECT p.id AS product_id, p.external_id, p.name, i.quantity, i.price
FROM order_items i
    JOIN products p ON p.id = i.product_id
WHERE i.order_id = $1
ORDER BY p.name;
