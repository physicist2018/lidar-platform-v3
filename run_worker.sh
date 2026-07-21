export DATABASE_URL=postgresql://lidar_user:pass@postgres:5432/main_db?search_path=lidar&sslmode=disable
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
export MINIO_USE_SSL="false"
export NATS_URL=nats://localhost:4222
go run cmd/worker/main.go
