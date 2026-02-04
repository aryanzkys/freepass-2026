-- name: UpsertPaymentVerificationPending :one
INSERT INTO payment_verifications (
	id,
	order_id,
	canteen_id,
	user_id,
	status,
	rejection_reason
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	'PENDING',
	NULL
)
ON CONFLICT (order_id) DO UPDATE
SET status = 'PENDING',
	rejection_reason = NULL,
	updated_at = now()
RETURNING id, order_id, canteen_id, user_id, status, rejection_reason, created_at, updated_at;

-- name: UpdatePaymentVerificationApproved :one
UPDATE payment_verifications
SET status = 'APPROVED',
	rejection_reason = NULL,
	updated_at = now()
WHERE order_id = $1
RETURNING id, order_id, canteen_id, user_id, status, rejection_reason, created_at, updated_at;

-- name: UpdatePaymentVerificationRejected :one
UPDATE payment_verifications
SET status = 'REJECTED',
	rejection_reason = $2,
	updated_at = now()
WHERE order_id = $1
RETURNING id, order_id, canteen_id, user_id, status, rejection_reason, created_at, updated_at;

-- name: GetPaymentVerificationByOrderID :one
SELECT id, order_id, canteen_id, user_id, status, rejection_reason, created_at, updated_at
FROM payment_verifications
WHERE order_id = $1;