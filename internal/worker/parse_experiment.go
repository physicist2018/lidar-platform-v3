package worker

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

const defaultBucket = "experiments"

// ParseExperimentHandler handles lidar.task.parse_experiment tasks.
// It downloads the experiment archive from MinIO, extracts files,
// and creates LicelFile / LicelProfile records.
type ParseExperimentHandler struct {
	experimentRepo   ports.ExperimentRepository
	storageObjRepo   ports.StorageObjectRepository
	fileStorage      ports.FileStorage
	licelFileRepo    ports.LicelFileRepository
	licelProfileRepo ports.LicelProfileRepository
}

// NewParseExperimentHandler creates a new ParseExperimentHandler.
func NewParseExperimentHandler(
	experimentRepo ports.ExperimentRepository,
	storageObjRepo ports.StorageObjectRepository,
	fileStorage ports.FileStorage,
	licelFileRepo ports.LicelFileRepository,
	licelProfileRepo ports.LicelProfileRepository,
) *ParseExperimentHandler {
	return &ParseExperimentHandler{
		experimentRepo:   experimentRepo,
		storageObjRepo:   storageObjRepo,
		fileStorage:      fileStorage,
		licelFileRepo:    licelFileRepo,
		licelProfileRepo: licelProfileRepo,
	}
}

func (h *ParseExperimentHandler) Subject() ports.Subject {
	return ports.SubjectParseExperiment
}

// Handle processes a parse_experiment task.
// data contains the experiment ID as a string.
func (h *ParseExperimentHandler) Handle(ctx context.Context, data []byte) error {
	expID := string(data)
	log.Printf("parse_experiment: processing experiment %s", expID)

	expUUID, err := uuid.Parse(expID)
	if err != nil {
		return fmt.Errorf("parse experiment id: %w", err)
	}

	// 1. Get experiment
	exp, err := h.experimentRepo.FindByID(ctx, expUUID)
	if err != nil {
		return fmt.Errorf("get experiment: %w", err)
	}

	// 2. Get archive StorageObject (experiments_storage_id)
	if exp.StorageRefs.ExperimentDataID == nil {
		return fmt.Errorf("experiment %s has no archive storage reference", expID)
	}
	archiveStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.ExperimentDataID)
	if err != nil {
		return fmt.Errorf("get archive storage object: %w", err)
	}
	log.Printf("parse_experiment: archive found: %s/%s", archiveStorage.Path.Bucket, archiveStorage.Path.Path)

	// 3. Download archive from MinIO
	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, archiveStorage.Path.Bucket, archiveStorage.Path.Path, &buf); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	log.Printf("parse_experiment: archive downloaded (%d bytes)", buf.Len())

	// 4. Open as zip
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	log.Printf("parse_experiment: archive contains %d entries", len(zipReader.File))

	// 5. Process each file in archive
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		log.Printf("parse_experiment: processing file %s", f.Name)

		// 5a. Upload file to MinIO
		objInfo, err := h.uploadFile(ctx, expUUID, f)
		if err != nil {
			return fmt.Errorf("upload file %s: %w", f.Name, err)
		}

		// 5b. Create StorageObject record
		objPath, _ := domain.NewObjectPath(objInfo.Bucket, objInfo.Path)
		storageObj := &domain.StorageObject{
			ID:          uuid.New(),
			Path:        objPath,
			Size:        objInfo.Size,
			ETag:        objInfo.ETag,
			ContentType: objInfo.ContentType,
			CreatedAt:   time.Now(),
		}
		storageObj, err = h.storageObjRepo.Create(ctx, storageObj)
		if err != nil {
			return fmt.Errorf("create storage object: %w", err)
		}

		// 5c. Create LicelFile record
		timeRange, _ := domain.NewTimeRange(time.Now(), time.Now().Add(time.Hour))
		licelFile := domain.NewLicelFile(
			expUUID,
			timeRange,
			0,     // nDatasets
			0,     // laserFreq
			false, // isBackground
			storageObj.ID,
			domain.WithFilename(f.Name),
		)
		if err := h.licelFileRepo.Create(ctx, &licelFile); err != nil {
			return fmt.Errorf("create licelfile: %w", err)
		}

		// TODO: Parse LICEL file contents, extract time range,
		// measurement metadata, create LicelProfile records
		// with actual data arrays.

		log.Printf("parse_experiment: created licelfile %s for %s", licelFile.ID, f.Name)
	}

	log.Printf("parse_experiment: experiment %s processed successfully (%d files)", expID, len(zipReader.File))
	return nil
}

// uploadFile uploads a single file from a zip archive to MinIO.
func (h *ParseExperimentHandler) uploadFile(ctx context.Context, expID uuid.UUID, f *zip.File) (*ports.ObjectInfo, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open zip entry: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read zip entry: %w", err)
	}

	path := fmt.Sprintf("%s/raw/%s", expID, f.Name)
	return h.fileStorage.UploadBytes(ctx, defaultBucket, path, data, "application/octet-stream")
}
