-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, expires_at, revoked_at, user_id)
VALUES(
    $1,
    NOW(),
    $2,
    $3, 
    $4,
    $5
)
RETURNING *;

-- name: GetUserViaRefreshToken :one
SELECT *
FROM refresh_tokens
WHERE token = $1;

-- name: SetRefreshTokenRevoked :exec
UPDATE refresh_tokens
SET revoked_at = $1, updated_at = $2
WHERE token = $3;