-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $2,
    $3,
    NULL
)
RETURNING token, created_at, updated_at, user_id, expires_at;

-- name: GetRefreshTokenByToken :one
SELECT token, created_at, updated_at, user_id, expires_at, revoked_at
FROM refresh_tokens
WHERE token = $1;

-- name: GetValidRefreshTokenByToken :one
SELECT token, created_at, updated_at, user_id, expires_at, revoked_at
FROM refresh_tokens
WHERE token = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP;

-- name: GetUserIdFromRefreshToken :one
SELECT user_id
FROM refresh_tokens
WHERE token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE token = $1;