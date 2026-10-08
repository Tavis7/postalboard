-- name: GetUserByRefreshToken :one
SELECT sqlc.embed(users), sqlc.embed(refresh_tokens)
FROM users
JOIN refresh_tokens ON refresh_tokens.user_id = users.id
WHERE refresh_tokens.token = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, expires_at, user_id)
VALUES(
    $1,
    $2,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = $2, updated_at = $2
WHERE refresh_tokens.token = $1
RETURNING *;
