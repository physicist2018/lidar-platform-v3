package config

import (
	"os"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/messaging"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/storage"
)

// Config holds all configuration for the worker.
type Config struct {
	DatabaseURL string
	MinIO       storage.Config
	NATS        messaging.Config
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL: env("DATABASE_URL", "postgresql://lidar_user:pass@localhost:5432/main_db?search_path=lidar,public&sslmode=disable"),
		MinIO: storage.Config{
			Endpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:    envBool("MINIO_USE_SSL", false),
		},
		NATS: messaging.Config{
			URL: env("NATS_URL", "nats://localhost:4222"),
		},
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
	return v == "true"
}
