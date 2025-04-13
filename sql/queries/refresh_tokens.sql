-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: GetUserFromRefreshToken :one
Select * from refresh_tokens
Where token=$1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
set revoked_at = NOW(), updated_at = NOW()
Where user_id=$1;
