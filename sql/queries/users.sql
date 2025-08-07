-- name: CreateUser :one
INSERT INTO users (id, hashed_password, created_at, updated_at, email)
VALUES(
    $1,
    $2,
    NOW(),
    $3,
    $4
)
RETURNING *;

-- name: RemoveAllUsers :exec
DELETE FROM users;

-- name: GetUserViaEmail :one
SELECT *
FROM users
WHERE email = $1;