-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $1,
    $2
)
RETURNING id, created_at, updated_at, email;

-- name: ResetUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password
FROM users
WHERE email = $1;

-- name: UpdateUserPasswordAndEmail :one
UPDATE users
SET hashed_password = $2, email = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, created_at, updated_at, email;

-- name: UpdateUserChirpyRedStatus :one
UPDATE users
SET is_chirpy_red = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, created_at, updated_at, email, is_chirpy_red;