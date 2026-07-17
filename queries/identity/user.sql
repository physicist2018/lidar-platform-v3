-- name: CreateUser :one
INSERT INTO identity.users (email, password_hash, status, verification_token, token_expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindByEmail :one
SELECT * FROM identity.users WHERE email = $1;

-- name: FindByVerificationToken :one
SELECT * FROM identity.users WHERE verification_token = $1::text AND token_expires_at > NOW();

-- name: UpdateUserStatus :one
UPDATE identity.users
SET status = $2, verification_token = NULL, token_expires_at = NULL, updated_at = NOW()
WHERE id = $1
RETURNING *;
