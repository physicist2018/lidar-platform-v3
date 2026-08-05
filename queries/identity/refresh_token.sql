-- name: CreateRefreshToken :one
INSERT INTO identity.refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindRefreshTokenByHash :one
SELECT * FROM identity.refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :one
UPDATE identity.refresh_tokens
SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL
RETURNING *;

-- name: RevokeAllUserTokens :exec
UPDATE identity.refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;
