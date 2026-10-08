-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, username, email, hashed_password)
VALUES(
    $1,
    $2,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE users.username = $1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE users.id = $1;
