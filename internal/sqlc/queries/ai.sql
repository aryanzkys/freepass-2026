-- name: ListFeedbackTextsByCanteenAndRange :many
SELECT f.id, f.order_id, f.created_at, f.rating, f.comment
FROM feedbacks f
JOIN orders o ON o.id = f.order_id
WHERE f.canteen_id = sqlc.arg(canteen_id)
	AND f.is_removed = false
	AND o.order_status = 'COMPLETED'
	AND f.created_at >= sqlc.arg(start_at)
	AND f.created_at <= sqlc.arg(end_at)
ORDER BY f.created_at DESC
LIMIT 200;

-- name: AggregateMenuRatingsByCanteen :many
SELECT menu_item_id,
	AVG(rating)::float8 AS avg_rating,
	COUNT(*)::int4 AS rating_count
FROM menu_ratings
WHERE canteen_id = sqlc.arg(canteen_id)
	AND is_removed = false
GROUP BY menu_item_id;

-- name: CanteenOrdersStatsByRange :one
SELECT
	COUNT(*)::int4 AS total_orders,
	COUNT(*) FILTER (WHERE order_status = 'COMPLETED')::int4 AS completed_orders,
	COUNT(*) FILTER (WHERE payment_status = 'REJECTED')::int4 AS rejected_payments,
	COUNT(*) FILTER (WHERE payment_status = 'REFUNDED')::int4 AS refunded_payments,
	COUNT(*) FILTER (WHERE payment_method = 'CASHLESS_QRIS')::int4 AS cashless_orders,
	COUNT(*) FILTER (WHERE payment_method = 'CASH')::int4 AS cash_orders
FROM orders
WHERE canteen_id = sqlc.arg(canteen_id)
	AND created_at >= sqlc.arg(start_at)
	AND created_at <= sqlc.arg(end_at);

-- name: ListCanteensBasic :many
SELECT id, name
FROM canteens
ORDER BY created_at DESC;