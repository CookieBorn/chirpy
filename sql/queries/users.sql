-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: Reset :exec
DELETE from users;

-- name: GetUserEmail :one
SELECT * from users
where email=$1;

-- name: UpdateUserEmailPassword :exec
UPDATE users
set email = $1, updated_at = NOW(), password = $2
Where id=$3;

-- name: GetUserEmailFromID :one
SELECT email from users
where id=$1;

-- name: SetUserToRed :exec
UPDATE users
set is_chirpy_red = true
Where id=$1;
