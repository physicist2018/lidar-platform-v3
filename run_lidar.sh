unset DATABASE_URL
# export DATABASE_URL="postgresql://lidar_user:pass@postgres:5432/main_db?search_path=lidar,public&sslmode=disable"
# DATABASE_URL="postgresql://lidar_user:pass@localhost:5432/main_db?search_path=lidar,public&sslmode=disable" \
export HTTP_ADDR=:8091
export MIGRATIONS_DIR=migrations/lidar
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
export MINIO_USE_SSL="false"
export NATS_URL=nats://localhost:4222
echo $DATABASE_URL
go run cmd/lidar/main.go
