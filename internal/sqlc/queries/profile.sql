-- name: UpdateUserProfile :one
UPDATE users
SET name = $2,
	phone = $3,
	updated_at = now()
WHERE id = $1
RETURNING id, name, email, password_hash, role, phone, created_at, updated_at;
