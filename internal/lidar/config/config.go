package config

import (
	"os"
	"strconv"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/messaging"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/storage"
)

// Config holds all configuration for the lidar service.
type Config struct {
	DatabaseURL   string
	MigrationsDir string
	MinIO         storage.Config
	NATS          messaging.Config
	HTTPAddr      string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		DatabaseURL:   env("DATABASE_URL", "postgresql://lidar_user:pass@localhost:5432/main_db?search_path=lidar&sslmode=disable"),
		MigrationsDir: env("MIGRATIONS_DIR", "migrations/lidar"),
		MinIO: storage.Config{
			Endpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:    envBool("MINIO_USE_SSL", false),
		},
		NATS: messaging.Config{
			URL: env("NATS_URL", "nats://localhost:4222"),
		},
		HTTPAddr: env("HTTP_ADDR", ":8091"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
