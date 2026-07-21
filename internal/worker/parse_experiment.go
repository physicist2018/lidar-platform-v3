package worker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/physicist2018/licelfile/v2/licelformat"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

const defaultBucket = "experiments"

// ParseExperimentHandler handles lidar.task.parse_experiment tasks.
type ParseExperimentHandler struct {
	experimentRepo        ports.ExperimentRepository
	storageObjRepo        ports.StorageObjectRepository
	fileStorage           ports.FileStorage
	licelFileRepo         ports.LicelFileRepository
	licelProfileRepo      ports.LicelProfileRepository
	atmosphereProfileRepo ports.AtmosphereProfileRepository
}

// NewParseExperimentHandler creates a new ParseExperimentHandler.
func NewParseExperimentHandler(
	experimentRepo ports.ExperimentRepository,
	storageObjRepo ports.StorageObjectRepository,
	fileStorage ports.FileStorage,
	licelFileRepo ports.LicelFileRepository,
	licelProfileRepo ports.LicelProfileRepository,
	atmosphereProfileRepo ports.AtmosphereProfileRepository,
) *ParseExperimentHandler {
	return &ParseExperimentHandler{
		experimentRepo:        experimentRepo,
		storageObjRepo:        storageObjRepo,
		fileStorage:           fileStorage,
		licelFileRepo:         licelFileRepo,
		licelProfileRepo:      licelProfileRepo,
		atmosphereProfileRepo: atmosphereProfileRepo,
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

	// 3. Process meteo file (optional)
	if exp.StorageRefs.MeteoID != nil {
		meteoStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.MeteoID)
		if err != nil {
			return fmt.Errorf("get meteo storage object: %w", err)
		}
		if _, err := h.processMeteoFile(ctx, expUUID, meteoStorage); err != nil {
			return fmt.Errorf("process meteo: %w", err)
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

	parts := strings.Split(storageObj.Path.Path, "/")
	filename := parts[len(parts)-1]

	if err := h.createLicelFileRecords(ctx, expUUID, filename, lf, storageObj.ID, isBackground); err != nil {
		return err
	}

	log.Printf("parse_experiment: single file processed — %d profiles", len(lf.Profiles))
	return nil
}

// processMeteoFile downloads a meteo CSV, parses pressure/height/temperature columns,
// and creates an AtmosphereProfile record.
func (h *ParseExperimentHandler) processMeteoFile(
	ctx context.Context,
	expUUID uuid.UUID,

	storageObj *domain.StorageObject,
) (*domain.AtmosphereProfile, error) {
	log.Printf("parse_experiment: processing meteo file %s/%s", storageObj.Path.Bucket, storageObj.Path.Path)

	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, storageObj.Path.Bucket, storageObj.Path.Path, &buf); err != nil {
		return nil, fmt.Errorf("download meteo: %w", err)
	}

	var altitudes, temperatures, pressures []float64
	scanner := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		// Skip first 4 header lines
		if lineNum <= 4 {
			continue
		}

		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		pres, err1 := strconv.ParseFloat(fields[0], 64)
		hght, err2 := strconv.ParseFloat(fields[1], 64)
		temp, err3 := strconv.ParseFloat(fields[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		altitudes = append(altitudes, hght/1000.0) // m → km
		temperatures = append(temperatures, temp)
		pressures = append(pressures, pres)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan meteo: %w", err)
	}

	if len(altitudes) == 0 {
		return nil, fmt.Errorf("meteo file has no data rows")
	}

	profile, err := domain.NewAtmosphereProfile(expUUID, altitudes, temperatures, pressures)
	if err != nil {
		return nil, fmt.Errorf("create atmosphere profile: %w", err)
	}

	if err := h.atmosphereProfileRepo.Create(ctx, &profile); err != nil {
		return nil, fmt.Errorf("save atmosphere profile: %w", err)
	}

	log.Printf("parse_experiment: meteo processed — %d data points", len(altitudes))
	return &profile, nil
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
