-- name: CreateOrder :one
INSERT INTO orders (
	id,
	user_id,
	canteen_id,
	total_amount
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3
)
RETURNING id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at;

-- name: CreateOrderItem :one
INSERT INTO order_items (
	id,
	order_id,
	menu_item_id,
	qty,
	price_snapshot,
	subtotal
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5
)
RETURNING id, order_id, menu_item_id, qty, price_snapshot, subtotal, created_at, updated_at;

-- name: ListOrdersByUserID :many
SELECT id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetOrderByID :one
SELECT id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: GetOrderWithItemsByID :many
SELECT
	o.id AS order_id,
	o.user_id AS order_user_id,
	o.canteen_id AS order_canteen_id,
	o.payment_status AS order_payment_status,
	o.order_status AS order_status,
	o.total_amount AS order_total_amount,
	o.created_at AS order_created_at,
	o.updated_at AS order_updated_at,
	oi.id AS item_id,
	oi.menu_item_id AS item_menu_item_id,
	oi.qty AS item_qty,
	oi.price_snapshot AS item_price_snapshot,
	oi.subtotal AS item_subtotal,
	oi.created_at AS item_created_at,
	oi.updated_at AS item_updated_at,
	m.name AS menu_item_name,
	m.description AS menu_item_description
FROM orders o
JOIN order_items oi ON oi.order_id = o.id
JOIN menu_items m ON m.id = oi.menu_item_id
WHERE o.id = $1
ORDER BY oi.created_at ASC;

-- name: ListOrdersByCanteenID :many
SELECT id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at
FROM orders
WHERE canteen_id = $1
ORDER BY created_at DESC;

-- name: UpdateOrderStatusIfPaid :one
UPDATE orders
SET order_status = $2,
	updated_at = now()
WHERE id = $1 AND payment_status = 'PAID'
RETURNING id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at;

-- name: MarkOrderPaid :one
UPDATE orders
SET payment_status = 'PAID',
	updated_at = now()
WHERE id = $1 AND payment_status = 'UNPAID'
RETURNING id, user_id, canteen_id, payment_status, order_status, total_amount, created_at, updated_at;
