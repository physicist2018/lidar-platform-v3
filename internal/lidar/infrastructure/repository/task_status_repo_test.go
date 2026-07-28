package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	useContainer := os.Getenv("DOCKER_TEST") == "1"

	if dbURL == "" && useContainer {
		container, err := postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("main_db"),
			postgres.WithUsername("user"),
			postgres.WithPassword("pass"),
		)
		if err != nil {
			log.Printf("WARNING: failed to start postgres container: %v", err)
			log.Printf("INFO: set DOCKER_TEST=1 with Docker running, or set TEST_DATABASE_URL")
			return
		}

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			log.Printf("WARNING: failed to get connection string: %v", err)
			return
		}
		dbURL = connStr

		defer func() {
			if err := container.Terminate(ctx); err != nil {
				log.Printf("WARNING: failed to terminate container: %v", err)
			}
		}()
	}

	if dbURL == "" {
		log.Printf("INFO: no database configured, skipping integration tests.")
		log.Printf("INFO: set TEST_DATABASE_URL or DOCKER_TEST=1 (with Docker) to enable.")
		return
	}

	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("WARNING: failed to connect: %v", err)
		return
	}
	defer testDB.Close()

	if err := applyMigrations(testDB); err != nil {
		log.Printf("WARNING: migration failed: %v", err)
		return
	}

	os.Exit(m.Run())
}

func applyMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE SCHEMA IF NOT EXISTS lidar;`,
		`CREATE TABLE IF NOT EXISTS lidar.storage_objects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			bucket TEXT NOT NULL,
			path TEXT NOT NULL,
			size_bytes BIGINT,
			etag TEXT,
			content_type TEXT,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,
		`CREATE TABLE IF NOT EXISTS lidar.experiments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			comments TEXT DEFAULT '',
			zenith_angle REAL NOT NULL,
			experiment_start TIMESTAMPTZ NOT NULL,
			experiment_end TIMESTAMPTZ NOT NULL,
			longitude REAL NOT NULL DEFAULT 131.9,
			latitude REAL NOT NULL DEFAULT 43.1,
			experiments_storage_id UUID REFERENCES lidar.storage_objects(id),
			background_storage_id UUID REFERENCES lidar.storage_objects(id),
			meteo_storage_id UUID REFERENCES lidar.storage_objects(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,
		`CREATE TABLE IF NOT EXISTS lidar.task_statuses (
			id UUID PRIMARY KEY,
			subject TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			task_params JSONB NOT NULL DEFAULT '{}',
			error_message TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_statuses_status ON lidar.task_statuses(status);`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

func TestTaskStatusRepo_CreateAndFindByID(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)
	record := domain.NewTaskRecord(uuid.New(), "lidar.task.test", nil)

	err := repo.Create(context.Background(), &record)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), record.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, record.ID, found.ID)
	assert.Equal(t, record.Subject, found.Subject)
	assert.Equal(t, domain.TaskPending, found.Status)
	assert.NotZero(t, found.CreatedAt)
	assert.NotZero(t, found.UpdatedAt)
}

func TestTaskStatusRepo_CreateWithExperimentID(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	_ = createTestExperiment(t, testDB)
	repo := NewPostgresTaskStatusRepository(testDB)
	record := domain.NewTaskRecord(uuid.New(), "lidar.task.parse_experiment", nil)

	err := repo.Create(context.Background(), &record)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.ID, found.ID)
}

func TestTaskStatusRepo_CreateDuplicateID(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)
	id := uuid.New()
	rec1 := domain.NewTaskRecord(id, "lidar.task.test", nil)
	rec2 := domain.NewTaskRecord(id, "lidar.task.test2", nil)

	require.NoError(t, repo.Create(context.Background(), &rec1))
	assert.Error(t, repo.Create(context.Background(), &rec2))
}

func TestTaskStatusRepo_UpdateStatus(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)
	record := domain.NewTaskRecord(uuid.New(), "lidar.task.test", nil)
	require.NoError(t, repo.Create(context.Background(), &record))

	err := repo.UpdateStatus(context.Background(), record.ID, domain.TaskProcessing, "")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), record.ID)
	assert.Equal(t, domain.TaskProcessing, found.Status)
	assert.NotNil(t, found.StartedAt)

	err = repo.UpdateStatus(context.Background(), record.ID, domain.TaskCompleted, "")
	require.NoError(t, err)

	found, _ = repo.FindByID(context.Background(), record.ID)
	assert.Equal(t, domain.TaskCompleted, found.Status)
	assert.NotNil(t, found.FinishedAt)
}

func TestTaskStatusRepo_UpdateStatusFailed(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)
	record := domain.NewTaskRecord(uuid.New(), "lidar.task.test", nil)
	require.NoError(t, repo.Create(context.Background(), &record))

	err := repo.UpdateStatus(context.Background(), record.ID, domain.TaskFailed, "something went wrong")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), record.ID)
	assert.Equal(t, domain.TaskFailed, found.Status)
	assert.Equal(t, "something went wrong", found.ErrorMessage)
	assert.NotNil(t, found.FinishedAt)
}

func TestTaskStatusRepo_FindByID_NotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)
	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrObjectNotFound)
}

func TestTaskStatusRepo_FindAll(t *testing.T) {
	if testDB == nil {
		t.Skip("no test database available")
	}

	repo := NewPostgresTaskStatusRepository(testDB)

	rec1 := domain.NewTaskRecord(uuid.New(), "lidar.task.a", nil)
	rec2 := domain.NewTaskRecord(uuid.New(), "lidar.task.b", nil)

	require.NoError(t, repo.Create(context.Background(), &rec1))
	require.NoError(t, repo.Create(context.Background(), &rec2))

	all, err := repo.FindAll(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 2)
}

func createTestExperiment(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO lidar.experiments (id, title, zenith_angle, experiment_start, experiment_end, longitude, latitude)
		VALUES ($1, 'test', 0, now(), now() + interval '1 hour', 0, 0)
	`, id)
	require.NoError(t, err)
	return id
}
