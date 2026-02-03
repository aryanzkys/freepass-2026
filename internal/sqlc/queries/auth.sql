-- name: CreateUser :one
INSERT INTO users (
	id,
	name,
	email,
	password_hash,
	role,
	phone
) VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5
)
RETURNING id, name, email, password_hash, role, phone, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, role, phone, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, role, phone, created_at, updated_at
FROM users
WHERE id = $1;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE id = $1;

-- name: UpdateUserAdminWithPassword :one
UPDATE users
SET name = $2,
	email = $3,
	phone = $4,
	password_hash = $5,
	updated_at = now()
WHERE id = $1
RETURNING id, name, email, password_hash, role, phone, created_at, updated_at;

-- name: UpdateUserAdminNoPassword :one
UPDATE users
SET name = $2,
	email = $3,
	phone = $4,
	updated_at = now()
WHERE id = $1
RETURNING id, name, email, password_hash, role, phone, created_at, updated_at;
