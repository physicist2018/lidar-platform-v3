#!/usr/bin/env bash
set -e

# Подключаемся к БД и выполняем SQL
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Создаём схему identity, если её нет
    CREATE SCHEMA IF NOT EXISTS identity;

    -- Создаём пользователя identity_user, если его нет
    DO \$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'identity_user') THEN
            CREATE USER identity_user WITH PASSWORD 'pass';
        END IF;
    END
    \$$;

    GRANT CONNECT ON DATABASE main_db TO identity_user;
    GRANT ALL ON SCHEMA identity TO identity_user;
    GRANT ALL ON ALL TABLES IN SCHEMA identity TO identity_user;
    ALTER DEFAULT PRIVILEGES IN SCHEMA identity GRANT ALL ON TABLES TO identity_user;

    -- Таблицы для identity
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

    -- Повторно выдаём права, чтобы они точно были на новой таблице
    GRANT ALL ON ALL TABLES IN SCHEMA identity TO identity_user;
    GRANT ALL ON ALL SEQUENCES IN SCHEMA identity TO identity_user;


    -- Создаём схему lidar, если её нет
    CREATE SCHEMA IF NOT EXISTS lidar;

    -- Создаём пользователя lidar_user, если его нет
    DO \$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'lidar_user') THEN
            CREATE USER lidar_user WITH PASSWORD 'pass';
        END IF;
    END
    \$$;

    GRANT CONNECT ON DATABASE main_db TO lidar_user;
    GRANT ALL ON SCHEMA lidar TO lidar_user;
    GRANT ALL ON ALL TABLES IN SCHEMA lidar TO lidar_user;
    ALTER DEFAULT PRIVILEGES IN SCHEMA lidar GRANT ALL ON TABLES TO lidar_user;
EOSQL
