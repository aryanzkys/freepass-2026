-- name: CreatePayment :one
INSERT INTO payments (
	id,
	order_id,
	method,
	amount
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3
)
RETURNING id, order_id, method, amount, status, paid_at, created_at, updated_at;

-- name: GetPaymentByOrderID :one
SELECT id, order_id, method, amount, status, paid_at, created_at, updated_at
FROM payments
WHERE order_id = $1;

-- name: DeletePaymentByOrderID :exec
DELETE FROM payments
WHERE order_id = $1;
