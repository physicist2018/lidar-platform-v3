package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/config"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/messaging"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/repository"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/server"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/storage"
)

const dbRetries = 10

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// ---------------------------------------------------------------
	// 1. Database
	// ---------------------------------------------------------------
	dbConn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: open: %v", err)
	}
	defer dbConn.Close()

	var pingErr error
	for i := 0; i < dbRetries; i++ {
		if pingErr = dbConn.Ping(); pingErr == nil {
			break
		}
		if i == dbRetries-1 {
			log.Fatalf("db: ping failed after %d attempts: %v", dbRetries, pingErr)
		}
		log.Printf("db: waiting for database... (%d/%d)", i+1, dbRetries)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	log.Println("db: connected")

	goose.SetDialect("postgres")
	if err := goose.Up(dbConn, cfg.MigrationsDir); err != nil {
		log.Fatalf("db: migration failed: %v", err)
	}
	log.Println("db: migrations applied")

	// ---------------------------------------------------------------
	// 2. Repositories
	// ---------------------------------------------------------------
	experimentRepo := repository.NewPostgresExperimentRepository(dbConn)
	storageObjRepo := repository.NewPostgresStorageObjectRepository(dbConn)
	taskStatusRepo := repository.NewPostgresTaskStatusRepository(dbConn)

	// ---------------------------------------------------------------
	// 3. MinIO
	// ---------------------------------------------------------------
	fileStorage, err := storage.NewMinIOFileStorage(cfg.MinIO)
	if err != nil {
		log.Fatalf("minio: init: %v", err)
	}

	if err := fileStorage.CreateBucket(ctx, "experiments"); err != nil {
		log.Fatalf("minio: create bucket: %v", err)
	}
	log.Println("minio: ready")

	// ---------------------------------------------------------------
	// 4. NATS
	// ---------------------------------------------------------------
	msgQueue, err := messaging.NewNatsMessageQueue(cfg.NATS)
	if err != nil {
		log.Fatalf("nats: init: %v", err)
	}
	defer msgQueue.Close()
	log.Println("nats: ready")

	// ---------------------------------------------------------------
	// 5. Use cases
	// ---------------------------------------------------------------
	createExpUC := application.NewCreateExperimentUseCase(fileStorage, storageObjRepo, experimentRepo, msgQueue, taskStatusRepo)
	createTaskUC := application.NewCreateTaskUseCase(msgQueue, taskStatusRepo)
	getTaskStatusUC := application.NewGetTaskStatusUseCase(taskStatusRepo)

	// ---------------------------------------------------------------
	// 6. HTTP server
	// ---------------------------------------------------------------
	expHandler := server.NewExperimentHandler(createExpUC)
	taskHandler := server.NewTaskHandler(createTaskUC, getTaskStatusUC)
	router := server.NewRouter(expHandler, taskHandler, cfg.JWTSecret)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: router}

	go func() {
		log.Printf("http: lidar service starting on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: server failed: %v", err)
		}
	}()

	// ---------------------------------------------------------------
	// 7. Graceful shutdown
	// ---------------------------------------------------------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("signal: received %s, shutting down...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("http: forced shutdown: %v", err)
	}
	log.Println("http: server stopped")
}
