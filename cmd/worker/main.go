package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/messaging"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/repository"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/infrastructure/storage"
	worker "github.com/physcist2018/lidar-platform-v3/internal/worker"
	"github.com/physcist2018/lidar-platform-v3/internal/worker/config"
)

const dbRetries = 10

func main() {
	cfg := config.Load()

	// ---------------------------------------------------------------
	// 1. Database
	// ---------------------------------------------------------------
	migrateURL := cfg.MigrationsURL
	if migrateURL == "" {
		migrateURL = cfg.DatabaseURL
	}

	migrateConn, err := sql.Open("postgres", migrateURL)
	if err != nil {
		log.Fatalf("db: open migrate: %v", err)
	}

	var pingErr error
	for i := 0; i < dbRetries; i++ {
		if pingErr = migrateConn.Ping(); pingErr == nil {
			break
		}
		if i == dbRetries-1 {
			log.Fatalf("db: ping failed after %d attempts: %v", dbRetries, pingErr)
		}
		log.Printf("db: waiting... (%d/%d)", i+1, dbRetries)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	log.Println("db: connected")

	goose.SetDialect("postgres")
	if err := goose.Up(migrateConn, cfg.MigrationsDir); err != nil {
		log.Fatalf("db: migration failed: %v", err)
	}
	migrateConn.Close()
	log.Println("db: migrations applied")

	dbConn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: open app: %v", err)
	}
	defer dbConn.Close()

	// ---------------------------------------------------------------
	// 2. Repositories
	// ---------------------------------------------------------------
	experimentRepo := repository.NewPostgresExperimentRepository(dbConn)
	storageObjRepo := repository.NewPostgresStorageObjectRepository(dbConn)

	// ---------------------------------------------------------------
	// 3. MinIO
	// ---------------------------------------------------------------
	fileStorage, err := storage.NewMinIOFileStorage(cfg.MinIO)
	if err != nil {
		log.Fatalf("minio: init: %v", err)
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
	// 5. Worker
	// ---------------------------------------------------------------
	w := worker.New(msgQueue)

	w.Register(
		worker.NewParseExperimentHandler(experimentRepo, storageObjRepo, fileStorage),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("signal: received %s", sig)
		cancel()
	}()

	log.Println("worker: starting...")
	if err := w.Run(ctx); err != nil {
		log.Fatalf("worker: %v", err)
	}

	w.Close()
	log.Println("worker: stopped")
}
