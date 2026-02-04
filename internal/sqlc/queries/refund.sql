-- name: CreateRefund :one
INSERT INTO refunds (
	id,
	order_id,
	canteen_id,
	user_id,
	amount,
	reason,
	status
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5,
	$6
)
RETURNING id, order_id, canteen_id, user_id, amount, reason, status, created_at, updated_at;