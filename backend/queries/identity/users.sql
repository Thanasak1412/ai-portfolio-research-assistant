-- name: CreateUser :one
INSERT INTO users (
    user_id,
    normalized_email,
    password_hash,
    account_status,
    created_at,
    updated_at,
    disabled_at
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(normalized_email),
    sqlc.arg(password_hash),
    'active',
    sqlc.arg(created_at),
    sqlc.arg(created_at),
    NULL
)
RETURNING *;

-- name: GetUserByNormalizedEmail :one
SELECT *
FROM users
WHERE normalized_email = sqlc.arg(normalized_email);

-- name: GetUserByID :one
SELECT *
FROM users
WHERE user_id = sqlc.arg(user_id);

-- name: UpdatePasswordHashCompareAndSwap :one
UPDATE users
SET
    password_hash = sqlc.arg(new_password_hash),
    updated_at = sqlc.arg(updated_at)
WHERE
    user_id = sqlc.arg(user_id)
    AND password_hash = sqlc.arg(expected_password_hash)
RETURNING *;
