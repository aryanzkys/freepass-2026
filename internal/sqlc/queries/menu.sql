-- name: ListMenuItemsWithRatingsByCanteenID :many
WITH agg AS (
	SELECT menu_item_id,
		AVG(rating)::float8 AS avg_rating,
		COUNT(*)::int4 AS rating_count
	FROM menu_ratings
	WHERE canteen_id = $1 AND is_removed = false
	GROUP BY menu_item_id
),
ranked AS (
	SELECT menu_item_id,
		ROW_NUMBER() OVER (ORDER BY avg_rating DESC NULLS LAST, rating_count DESC, menu_item_id ASC)::int4 AS recommended_rank
	FROM agg
)
SELECT mi.id,
	mi.canteen_id,
	mi.name,
	mi.description,
	mi.price,
	mi.stock,
	mi.is_available,
	mi.created_at,
	mi.updated_at,
	agg.avg_rating,
	COALESCE(agg.rating_count, 0)::int4 AS rating_count,
	(CASE WHEN ranked.recommended_rank <= 5 THEN true ELSE false END) AS is_recommended,
	COALESCE(ranked.recommended_rank, 0)::int4 AS recommended_rank
FROM menu_items mi
LEFT JOIN agg ON agg.menu_item_id = mi.id
LEFT JOIN ranked ON ranked.menu_item_id = mi.id
WHERE mi.canteen_id = $1
ORDER BY is_recommended DESC, recommended_rank ASC NULLS LAST, mi.created_at DESC;

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
