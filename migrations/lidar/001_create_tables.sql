-- +goose Up
CREATE SCHEMA IF NOT EXISTS lidar;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Таблица объектов хранилища (инфраструктурный реестр)
CREATE TABLE lidar.storage_objects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket       TEXT NOT NULL,
    path         TEXT NOT NULL,
    size_bytes   BIGINT,
    etag         TEXT,
    content_type TEXT,
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_storage_objects_bucket_path_unique ON lidar.storage_objects(bucket, path);

-- Профили атмосферы
CREATE TABLE lidar.atmosphere_profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    altitude        DOUBLE PRECISION[] NOT NULL,
    temperature     DOUBLE PRECISION[] NOT NULL,
    pressure        DOUBLE PRECISION[] NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Эксперименты
CREATE TABLE lidar.experiments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                   TEXT NOT NULL,
    comments                TEXT DEFAULT '',
    zenith_angle            REAL NOT NULL,

    experiment_start        TIMESTAMPTZ NOT NULL,
    experiment_end          TIMESTAMPTZ NOT NULL,
    longitude               REAL NOT NULL DEFAULT 131.9,
    latitude                REAL NOT NULL DEFAULT 43.1,

    atmosphere_profile_id   UUID NOT NULL
        REFERENCES lidar.atmosphere_profiles(id)
        ON DELETE RESTRICT,

    experiments_storage_id  UUID REFERENCES lidar.storage_objects(id),
    background_storage_id   UUID REFERENCES lidar.storage_objects(id),
    meteo_storage_id        UUID REFERENCES lidar.storage_objects(id),

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ
);

ALTER TABLE lidar.experiments
    ADD CONSTRAINT uq_experiments_atmosphere_profile_id
    UNIQUE (atmosphere_profile_id);

CREATE INDEX idx_experiments_deleted_at ON lidar.experiments(deleted_at);
CREATE INDEX idx_experiments_created_at ON lidar.experiments(created_at);
CREATE INDEX idx_experiments_start ON lidar.experiments(experiment_start);

-- Файлы LICEL (архив/сырые данные)
CREATE TABLE lidar.licelfiles (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id        UUID NOT NULL REFERENCES lidar.experiments(id) ON DELETE CASCADE,

    measurement_start    TIMESTAMPTZ NOT NULL,
    measurement_stop     TIMESTAMPTZ NOT NULL,
    n_datasets           INT NOT NULL,
    laser_freq           INT NOT NULL,
    is_background        BOOLEAN NOT NULL DEFAULT FALSE,

    raw_storage_id        UUID NOT NULL REFERENCES lidar.storage_objects(id),
    filename             TEXT NOT NULL DEFAULT '',

    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX idx_licelfiles_experiment_id ON lidar.licelfiles(experiment_id);

-- Профили из файлов LICEL
CREATE TABLE lidar.licel_profiles (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    licelfile_id     UUID NOT NULL REFERENCES lidar.licelfiles(id) ON DELETE CASCADE,
    n_data_points    INT NOT NULL,
    high_voltage     REAL NOT NULL,
    bin_width        REAL NOT NULL,

    wavelength       REAL NOT NULL,
    polarization     TEXT NOT NULL,
    device_id        TEXT NOT NULL,

    n_shots          INT NOT NULL,
    discr_level      REAL NOT NULL,
    data             DOUBLE PRECISION[] NOT NULL,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_licel_profiles_licelfile_id
    ON lidar.licel_profiles(licelfile_id);
CREATE INDEX idx_licel_profiles_wavelength_polarization_device_id
    ON lidar.licel_profiles(wavelength, polarization, device_id);

CREATE TABLE lidar.prepared_meta(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NOT NULL REFERENCES lidar.experiments(id) ON DELETE CASCADE,
    background_type TEXT NOT NULL DEFAULT 'mean',
    background_from REAL NOT NULL DEFAULT 80000.0,
    trim_from REAL NOT NULL DEFAULT 20000.0
);

CREATE TABLE lidar.prepared_profiles(
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prepared_meta_id UUID NOT NULL REFERENCES lidar.prepared_meta(id) ON DELETE CASCADE,
    licel_profile_id     UUID NOT NULL REFERENCES lidar.licel_profiles(id) ON DELETE CASCADE,
    data             REAL[] NOT NULL,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS lidar.prepared_profiles;
DROP TABLE IF EXISTS lidar.prepared_meta;

DROP TABLE IF EXISTS lidar.licel_profiles;
DROP TABLE IF EXISTS lidar.licelfiles;

DROP TABLE IF EXISTS lidar.experiments;
DROP TABLE IF EXISTS lidar.atmosphere_profiles;

DROP TABLE IF EXISTS lidar.storage_objects;
