package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/auth"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/mail"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/repository"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/server"
)

const dbRetries = 10

func main() {
	// Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://identity_user:pass@localhost:5432/main_db?search_path=identity&sslmode=disable"
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer dbConn.Close()

	// Retry ping with backoff — the database may still be starting up.
	var pingErr error
	for i := 0; i < dbRetries; i++ {
		if pingErr = dbConn.Ping(); pingErr == nil {
			break
		}
		if i == dbRetries-1 {
			log.Fatalf("failed to ping database after %d attempts: %v", dbRetries, pingErr)
		}
		log.Printf("waiting for database... (%d/%d)", i+1, dbRetries)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	log.Println("connected to database")

	// Run migrations
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations/identity"
	}
	goose.SetDialect("postgres")
	if err := goose.Up(dbConn, migrationsDir); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migrations applied")

	// Repositories
	userRepo := repository.NewPostgresUserRepository(dbConn)

	// Mail sender
	verifyBaseURL := os.Getenv("VERIFY_BASE_URL")
	if verifyBaseURL == "" {
		verifyBaseURL = "http://localhost:8080"
	}
	smtpCfg := mail.Config{
		Server:        os.Getenv("SMTP_SERVER"),
		Username:      os.Getenv("SMTP_USERNAME"),
		Password:      os.Getenv("SMTP_PASSWORD"),
		From:          os.Getenv("SMTP_FROM"),
		VerifyBaseURL: verifyBaseURL,
	}
	mailSender := mail.NewSmtpMailSender(smtpCfg)

	// JWT token service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		b := make([]byte, 32)
		rand.Read(b) // nolint: errcheck
		jwtSecret = hex.EncodeToString(b)
		log.Println("WARNING: JWT_SECRET not set, using ephemeral secret — tokens will be invalidated on restart")
	}
	tokenService := auth.NewJWTTokenService(jwtSecret)

	// Use cases
	registerUC := application.NewRegisterUseCase(userRepo, mailSender)
	verifyUC := application.NewVerifyUseCase(userRepo)
	loginUC := application.NewLoginUseCase(userRepo, tokenService)

	// HTTP server
	frontendURL := os.Getenv("FRONTEND_URL")

	registerHandler := server.NewRegisterHandler(registerUC)
	verifyLinkHandler := server.NewVerifyLinkHandler(verifyUC, frontendURL)
	loginHandler := server.NewLoginHandler(loginUC)
	router := server.NewRouter(registerHandler, verifyLinkHandler, loginHandler)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("starting identity service on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
