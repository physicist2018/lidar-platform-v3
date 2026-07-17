-- +goose Up
CREATE TABLE IF NOT EXISTS identity.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    verification_token TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON identity.users (email);
CREATE INDEX IF NOT EXISTS idx_users_verification_token ON identity.users (verification_token);

-- +goose Down
DROP TABLE IF EXISTS identity.users CASCADE;
