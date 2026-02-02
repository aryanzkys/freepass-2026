-- name: ListCanteens :many
SELECT id, name, location, owner_id, created_at, updated_at
FROM canteens
ORDER BY created_at DESC;

-- name: GetCanteenByID :one
SELECT id, name, location, owner_id, created_at, updated_at
FROM canteens
WHERE id = $1;

-- name: CreateCanteen :one
INSERT INTO canteens (
	id,
	name,
	location,
	owner_id
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3
)
RETURNING id, name, location, owner_id, created_at, updated_at;

-- name: ListCanteensByOwnerID :many
SELECT id, name, location, owner_id, created_at, updated_at
FROM canteens
WHERE owner_id = $1
ORDER BY created_at DESC;
