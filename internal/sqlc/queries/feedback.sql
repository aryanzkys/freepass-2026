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

-- name: ListFeedbacksByCanteenID :many
SELECT id, order_id, user_id, canteen_id, rating, comment, is_removed, created_at, updated_at
FROM feedbacks
WHERE canteen_id = $1 AND is_removed = false
ORDER BY created_at DESC;

-- name: SoftRemoveFeedbackByID :one
UPDATE feedbacks
SET is_removed = true,
	updated_at = now()
WHERE id = $1 AND canteen_id = $2
RETURNING id, order_id, user_id, canteen_id, rating, comment, is_removed, created_at, updated_at;
