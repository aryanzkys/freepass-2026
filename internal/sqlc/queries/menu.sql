-- name: ListMenuItemsByCanteenID :many
SELECT id, canteen_id, name, description, price, stock, is_available, created_at, updated_at
FROM menu_items
WHERE canteen_id = $1
ORDER BY created_at DESC;

-- name: GetMenuItemByID :one
SELECT id, canteen_id, name, description, price, stock, is_available, created_at, updated_at
FROM menu_items
WHERE id = $1;

-- name: CreateMenuItem :one
INSERT INTO menu_items (
	id,
	canteen_id,
	name,
	description,
	price,
	stock,
	is_available
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5,
	$6
)
RETURNING id, canteen_id, name, description, price, stock, is_available, created_at, updated_at;

-- name: UpdateMenuItem :one
UPDATE menu_items
SET name = $2,
	description = $3,
	price = $4,
	stock = $5,
	is_available = $6,
	updated_at = now()
WHERE id = $1
RETURNING id, canteen_id, name, description, price, stock, is_available, created_at, updated_at;

-- name: DeleteMenuItem :exec
DELETE FROM menu_items
WHERE id = $1 AND canteen_id = $2;

-- name: LockMenuItemForUpdate :one
SELECT id, canteen_id, name, description, price, stock, is_available, created_at, updated_at
FROM menu_items
WHERE id = $1
FOR UPDATE;

-- name: DecreaseMenuItemStock :one
UPDATE menu_items
SET stock = stock - $2,
	updated_at = now()
WHERE id = $1 AND stock >= $2
RETURNING id, canteen_id, name, description, price, stock, is_available, created_at, updated_at;
