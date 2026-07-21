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

	// Download archive from MinIO
	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, archiveStorage.Path.Bucket, archiveStorage.Path.Path, &buf); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	log.Printf("parse_experiment: archive downloaded (%d bytes)", buf.Len())

	// Save to temp file (LicelPack requires a file path)
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

	// Parse all files in the archive with LicelPack
	lp, err := licelformat.NewLicelPackFromZip(tmpPath)
	if err != nil {
		return fmt.Errorf("new licel pack: %w", err)
	}
	log.Printf("parse_experiment: parsing complete: %d files, %d total profiles",
		len(lp.Data), countProfiles(lp))

	// Create LicelFile + LicelProfile records for each parsed file
	for name, lf := range lp.Data {
		log.Printf("parse_experiment: processing file %s (%d profiles)", name, len(lf.Profiles))

		isBackground := strings.Contains(strings.ToLower(name), "background")

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
			archiveStorage.ID,
			domain.WithFilename(name),
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
	}

	totalProfiles := countProfiles(lp)
	log.Printf("parse_experiment: experiment %s done — %d files, %d profiles",
		expID, len(lp.Data), totalProfiles)
	return nil
}

func countProfiles(lp *licelformat.LicelPack) int {
	n := 0
	for _, lf := range lp.Data {
		n += len(lf.Profiles)
	}
	return n
}
