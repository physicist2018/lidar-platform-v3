package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"

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
		log.Printf("db: waiting... (%d/%d)", i+1, dbRetries)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	log.Println("db: connected")

	// ---------------------------------------------------------------
	// 2. Repositories
	// ---------------------------------------------------------------
	experimentRepo := repository.NewPostgresExperimentRepository(dbConn)
	storageObjRepo := repository.NewPostgresStorageObjectRepository(dbConn)
	licelFileRepo := repository.NewPostgresLicelFileRepository(dbConn)
	licelProfileRepo := repository.NewPostgresLicelProfileRepository(dbConn)
	atmosphereProfileRepo := repository.NewPostgresAtmosphereProfileRepository(dbConn)

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
		worker.NewParseExperimentHandler(
			experimentRepo, storageObjRepo, fileStorage,
			licelFileRepo, licelProfileRepo, atmosphereProfileRepo,
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := w.Run(ctx); err != nil {
			log.Fatalf("worker: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("signal: received %s, shutting down...", sig)
	cancel()

	wg.Wait()
	w.Close()
	log.Println("worker: stopped")
}
