-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, username, email)
VALUES(
    id = id,
    created_at = NOW(),
    updated_ad = NOW(),
    username = username,
    email = email
)
RETURNING *;
