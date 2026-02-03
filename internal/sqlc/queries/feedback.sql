-- name: CreateFeedback :one
INSERT INTO feedbacks (
	id,
	order_id,
	user_id,
	canteen_id,
	rating,
	comment
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5
)
RETURNING id, order_id, user_id, canteen_id, rating, comment, is_removed, created_at, updated_at;

-- name: GetFeedbackByOrderID :one
SELECT id, order_id, user_id, canteen_id, rating, comment, is_removed, created_at, updated_at
FROM feedbacks
WHERE order_id = $1;

-- name: SoftRemoveFeedbackByID :one
UPDATE feedbacks
SET is_removed = true,
	updated_at = now()
WHERE id = $1 AND canteen_id = $2
RETURNING id, order_id, user_id, canteen_id, rating, comment, is_removed, created_at, updated_at;

-- name: CreateMenuRating :one
INSERT INTO menu_ratings (
	id,
	order_id,
	menu_item_id,
	user_id,
	canteen_id,
	rating
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5
)
RETURNING id, order_id, menu_item_id, user_id, canteen_id, rating, is_removed, created_at, updated_at;

-- name: GetMenuRatingsByOrderID :many
SELECT id, order_id, menu_item_id, user_id, canteen_id, rating, is_removed, created_at, updated_at
FROM menu_ratings
WHERE order_id = $1 AND is_removed = false;

-- name: SoftRemoveMenuRatingsByOrderID :exec
UPDATE menu_ratings
SET is_removed = true,
	updated_at = now()
WHERE order_id = $1 AND is_removed = false;

-- name: GetOrderItemsMenuIDsByOrderID :many
SELECT menu_item_id, qty
FROM order_items
WHERE order_id = $1;
