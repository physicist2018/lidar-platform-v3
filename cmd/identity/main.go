package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/physcist2018/lidar-platform-v3/internal/identity/application"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/mail"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/repository"
	"github.com/physcist2018/lidar-platform-v3/internal/identity/infrastructure/server"
)

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

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to database")

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

	// Use cases
	registerUC := application.NewRegisterUseCase(userRepo, mailSender)
	verifyUC := application.NewVerifyUseCase(userRepo)

	// HTTP server
	frontendURL := os.Getenv("FRONTEND_URL")

	registerHandler := server.NewRegisterHandler(registerUC)
	verifyHandler := server.NewVerifyHandler(verifyUC)
	verifyLinkHandler := server.NewVerifyLinkHandler(verifyUC, frontendURL)
	router := server.NewRouter(registerHandler, verifyHandler, verifyLinkHandler)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("starting identity service on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
