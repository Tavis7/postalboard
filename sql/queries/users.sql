-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, username, email)
VALUES(
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = $1;
