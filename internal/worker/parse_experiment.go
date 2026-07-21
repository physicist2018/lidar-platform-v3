package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/physicist2018/licelfile/v2/licelformat"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

const defaultBucket = "experiments"

// ParseExperimentHandler handles lidar.task.parse_experiment tasks.
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

func (h *ParseExperimentHandler) Handle(ctx context.Context, data []byte) error {
	expID := string(data)
	log.Printf("parse_experiment: processing experiment %s", expID)

	expUUID, err := uuid.Parse(expID)
	if err != nil {
		return fmt.Errorf("parse experiment id: %w", err)
	}

	exp, err := h.experimentRepo.FindByID(ctx, expUUID)
	if err != nil {
		return fmt.Errorf("get experiment: %w", err)
	}

	if exp.StorageRefs.ExperimentDataID == nil {
		return fmt.Errorf("experiment %s has no archive storage reference", expID)
	}
	archiveStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.ExperimentDataID)
	if err != nil {
		return fmt.Errorf("get archive storage object: %w", err)
	}

	// 1. Process experiment archive (zip)
	if err := h.processArchive(ctx, expUUID, archiveStorage, false); err != nil {
		return fmt.Errorf("process archive: %w", err)
	}

	// 2. Process background file (single LICEL file, optional)
	if exp.StorageRefs.BackgroundID != nil {
		bgStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.BackgroundID)
		if err != nil {
			return fmt.Errorf("get background storage object: %w", err)
		}
		if err := h.processSingleLicelFile(ctx, expUUID, bgStorage, true); err != nil {
			return fmt.Errorf("process background: %w", err)
		}
	}

	log.Printf("parse_experiment: experiment %s done", expID)
	return nil
}

// processArchive downloads a zip archive from MinIO, parses all LICEL files inside,
// and creates LicelFile + LicelProfile records.
func (h *ParseExperimentHandler) processArchive(
	ctx context.Context,
	expUUID uuid.UUID,
	storageObj *domain.StorageObject,
	isBackground bool,
) error {
	log.Printf("parse_experiment: processing archive %s/%s", storageObj.Path.Bucket, storageObj.Path.Path)

	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, storageObj.Path.Bucket, storageObj.Path.Path, &buf); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

	lp, err := licelformat.NewLicelPackFromZip(tmpPath)
	if err != nil {
		return fmt.Errorf("new licel pack: %w", err)
	}

	for name, lf := range lp.Data {
		if err := h.createLicelFileRecords(ctx, expUUID, name, lf, storageObj.ID, isBackground); err != nil {
			return err
		}
	}

	log.Printf("parse_experiment: archive processed — %d files, %d profiles", len(lp.Data), countProfiles(lp))
	return nil
}

// processSingleLicelFile downloads a single LICEL file from MinIO, parses it,
// and creates LicelFile + LicelProfile records.
func (h *ParseExperimentHandler) processSingleLicelFile(
	ctx context.Context,
	expUUID uuid.UUID,
	storageObj *domain.StorageObject,
	isBackground bool,
) error {
	log.Printf("parse_experiment: processing single file %s/%s", storageObj.Path.Bucket, storageObj.Path.Path)

	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, storageObj.Path.Bucket, storageObj.Path.Path, &buf); err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	lf, err := licelformat.LoadLicelFileFromReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("parse licelfile: %w", err)
	}

	// Extract filename from path
	parts := strings.Split(storageObj.Path.Path, "/")
	filename := parts[len(parts)-1]

	if err := h.createLicelFileRecords(ctx, expUUID, filename, lf, storageObj.ID, isBackground); err != nil {
		return err
	}

	log.Printf("parse_experiment: single file processed — %d profiles", len(lf.Profiles))
	return nil
}

// createLicelFileRecords creates a LicelFile and its LicelProfiles from a parsed LICEL file.
func (h *ParseExperimentHandler) createLicelFileRecords(
	ctx context.Context,
	expUUID uuid.UUID,
	filename string,
	lf licelformat.LicelFile,
	rawStorageID uuid.UUID,
	isBackground bool,
) error {
	log.Printf("parse_experiment: creating records for %s (%d profiles)", filename, len(lf.Profiles))

	laserFreq := lf.Laser1Freq
	if laserFreq == 0 && lf.Laser2Freq != 0 {
		laserFreq = lf.Laser2Freq
	}
	if laserFreq == 0 && lf.Laser3Freq != 0 {
		laserFreq = lf.Laser3Freq
	}

	timeRange, _ := domain.NewTimeRange(lf.MeasurementStartTime, lf.MeasurementStopTime)
	licelFile := domain.NewLicelFile(
		expUUID,
		timeRange,
		int32(lf.NDatasets),
		int32(laserFreq),
		isBackground,
		rawStorageID,
		domain.WithFilename(filename),
	)
	if err := h.licelFileRepo.Create(ctx, &licelFile); err != nil {
		return fmt.Errorf("create licelfile: %w", err)
	}

	for _, lp := range lf.Profiles {
		profile, err := domain.NewLicelProfile(
			licelFile.ID,
			int32(lp.NDataPoints),
			float32(lp.HighVoltage),
			float32(lp.BinWidth),
			float32(lp.Wavelength),
			lp.Polarization,
			lp.DeviceID,
			int32(lp.NShots),
			float32(lp.DiscrLevel),
			lp.Data,
		)
		if err != nil {
			return fmt.Errorf("create licel profile: %w", err)
		}
		if err := h.licelProfileRepo.Create(ctx, &profile); err != nil {
			return fmt.Errorf("save licel profile: %w", err)
		}
	}
	return nil
}

func countProfiles(lp *licelformat.LicelPack) int {
	n := 0
	for _, lf := range lp.Data {
		n += len(lf.Profiles)
	}
	return n
}
