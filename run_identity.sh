unset DATABASE_URL
#export DATABASE_URL="postgresql://identity_user:pass@postgres:5432/main_db?search_path=identity,public&sslmode=disable"
export HTTP_ADDR=:8090
export JWT_SECRET="change-me-in-production"
export MIGRATIONS_DIR=migrations/identity
export SMTP_SERVER=smtp.mail.ru:587
export SMTP_USERNAME=lidar-processing@inbox.ru
export SMTP_PASSWORD=ulhEEMqGkJmhPXPF0mej
export SMTP_FROM=lidar-processing@inbox.ru
export VERIFY_BASE_URL=https://localhost
export FRONTEND_URL=https://localhost
go run cmd/identity/main.go
