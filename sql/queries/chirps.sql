-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES(
    $1,
    NOW(),
    $2,
    $3, 
    $4
)
RETURNING *;

-- name: GetAllChirpsSinceCreation :many
SELECT * 
FROM chirps
WHERE id IS NOT NULL
ORDER BY created_at ASC;

-- name: GetChirpViaID :one
SELECT * 
FROM chirps
WHERE id = $1;