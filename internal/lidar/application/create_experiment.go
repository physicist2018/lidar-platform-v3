package application

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

const (
	defaultBucket      = "experiments"
	storageContentType = "application/octet-stream"
)

// FileUpload represents a file to be uploaded, decoupled from HTTP types.
type FileUpload struct {
	Filename string
	Size     int64
	Reader   io.Reader
}

// CreateExperimentRequest contains the parsed data for experiment creation.
type CreateExperimentRequest struct {
	Title           string
	ZenithAngle     float32
	Latitude        float32
	Longitude       float32
	Comments        string
	ExperimentFiles FileUpload
	Background      *FileUpload
	Meteo           *FileUpload
}

// CreateExperimentResponse is returned on success.
type CreateExperimentResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	ZenithAngle float32   `json:"zenith_angle"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	Comments    string    `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateExperimentUseCase orchestrates file upload and experiment creation.
type CreateExperimentUseCase struct {
	storage    ports.FileStorage
	storageObj ports.StorageObjectRepository
	experiment ports.ExperimentRepository
	queue      ports.MessageQueue
}

// NewCreateExperimentUseCase creates a new CreateExperimentUseCase.
func NewCreateExperimentUseCase(
	storage ports.FileStorage,
	storageObj ports.StorageObjectRepository,
	experiment ports.ExperimentRepository,
	queue ports.MessageQueue,
) *CreateExperimentUseCase {
	return &CreateExperimentUseCase{
		storage:    storage,
		storageObj: storageObj,
		experiment: experiment,
		queue:      queue,
	}
}

// Execute creates an experiment: uploads files, stores metadata, creates experiment.
func (uc *CreateExperimentUseCase) Execute(ctx context.Context, req *CreateExperimentRequest) (*CreateExperimentResponse, error) {
	expID := uuid.New()

	// 1. Upload experiment files and create storage object.
	experimentObj, err := uc.uploadAndCreateStorage(ctx, expID, "raw", req.ExperimentFiles)
	if err != nil {
		return nil, fmt.Errorf("upload experiment files: %w", err)
	}

	// 2. Upload background (optional).
	var backgroundObj *domain.StorageObject
	if req.Background != nil {
		backgroundObj, err = uc.uploadAndCreateStorage(ctx, expID, "background", *req.Background)
		if err != nil {
			return nil, fmt.Errorf("upload background: %w", err)
		}
	}

	// 3. Upload meteo (optional).
	var meteoObj *domain.StorageObject
	if req.Meteo != nil {
		meteoObj, err = uc.uploadAndCreateStorage(ctx, expID, "meteo", *req.Meteo)
		if err != nil {
			return nil, fmt.Errorf("upload meteo: %w", err)
		}
	}

	// 4. Build and persist experiment.
	timeRange, _ := domain.NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	geoLocation, _ := domain.NewGeoLocation(req.Latitude, req.Longitude)

	experiment := domain.NewExperiment(
		req.Title,
		req.ZenithAngle,
		timeRange,
		geoLocation,
		domain.WithComments(req.Comments),
		domain.WithStorageRefs(domain.ExperimentStorageRefs{
			ExperimentDataID: &experimentObj.ID,
			BackgroundID:     storageIDPtr(backgroundObj),
			MeteoID:          storageIDPtr(meteoObj),
		}),
	)
	// Override the generated ID with the one we used for file paths.
	experiment.ID = expID

	if err := uc.experiment.Create(ctx, &experiment); err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}

	// 5. Publish async task.
	if uc.queue != nil {
		if err := uc.queue.Publish(ctx, ports.SubjectParseExperiment, []byte(expID.String()), expID.String()); err != nil {
			log.Printf("publish task: %v", err)
		}
	}

	return &CreateExperimentResponse{
		ID:          experiment.ID,
		Title:       experiment.Title,
		ZenithAngle: experiment.ZenithAngle,
		Latitude:    experiment.GeoLocation.Latitude,
		Longitude:   experiment.GeoLocation.Longitude,
		Comments:    experiment.Comments,
		CreatedAt:   experiment.CreatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (uc *CreateExperimentUseCase) uploadAndCreateStorage(
	ctx context.Context,
	expID uuid.UUID,
	kind string,
	file FileUpload,
) (*domain.StorageObject, error) {
	objectPath := fmt.Sprintf("%s/%s/%s", expID, kind, file.Filename)

	info, err := uc.storage.Upload(ctx, defaultBucket, objectPath, file.Reader, file.Size, storageContentType)
	if err != nil {
		return nil, err
	}

	domainObj := &domain.StorageObject{
		ID:          uuid.New(),
		Size:        info.Size,
		ETag:        info.ETag,
		ContentType: info.ContentType,
		CreatedAt:   time.Now(),
	}
	domainObj.Path, _ = domain.NewObjectPath(info.Bucket, info.Path)

	return uc.storageObj.Create(ctx, domainObj)
}

func storageIDPtr(obj *domain.StorageObject) *uuid.UUID {
	if obj == nil {
		return nil
	}
	return &obj.ID
}
