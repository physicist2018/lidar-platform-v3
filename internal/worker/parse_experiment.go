package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/physicist2018/licelfile/v2/licelformat"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// ParseExperimentHandler handles lidar.task.parse_experiment tasks.
type ParseExperimentHandler struct {
	experimentRepo        ports.ExperimentRepository
	storageObjRepo        ports.StorageObjectRepository
	fileStorage           ports.FileStorage
	licelFileRepo         ports.LicelFileRepository
	licelProfileRepo      ports.LicelProfileRepository
	atmosphereProfileRepo ports.AtmosphereProfileRepository
	taskStatusRepo        ports.TaskStatusRepository
}

// NewParseExperimentHandler creates a new ParseExperimentHandler.
func NewParseExperimentHandler(
	experimentRepo ports.ExperimentRepository,
	storageObjRepo ports.StorageObjectRepository,
	fileStorage ports.FileStorage,
	licelFileRepo ports.LicelFileRepository,
	licelProfileRepo ports.LicelProfileRepository,
	atmosphereProfileRepo ports.AtmosphereProfileRepository,
	taskStatusRepo ports.TaskStatusRepository,
) *ParseExperimentHandler {
	return &ParseExperimentHandler{
		experimentRepo:        experimentRepo,
		storageObjRepo:        storageObjRepo,
		fileStorage:           fileStorage,
		licelFileRepo:         licelFileRepo,
		licelProfileRepo:      licelProfileRepo,
		atmosphereProfileRepo: atmosphereProfileRepo,
		taskStatusRepo:        taskStatusRepo,
	}
}

func (h *ParseExperimentHandler) Subject() ports.Subject {
	return ports.SubjectParseExperiment
}

func (h *ParseExperimentHandler) Handle(ctx context.Context, data []byte) error {
	// Parse the NATS message produced by CreateTaskUseCase.
	var msg struct {
		TaskID  string          `json:"task_id"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	expUUID, err := uuid.Parse(msg.TaskID)
	if err != nil {
		return fmt.Errorf("parse experiment id: %w", err)
	}

	log.Printf("parse_experiment: processing experiment %s", expUUID)

	h.updateTaskStatus(ctx, expUUID, domain.TaskProcessing, "")

	exp, err := h.experimentRepo.FindByID(ctx, expUUID)
	if err != nil {
		h.failTask(ctx, expUUID, err)
		return fmt.Errorf("get experiment: %w", err)
	}

	if exp.StorageRefs.ExperimentDataID == nil {
		err := fmt.Errorf("experiment %s has no archive storage reference", expUUID)
		h.failTask(ctx, expUUID, err)
		return err
	}
	archiveStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.ExperimentDataID)
	if err != nil {
		h.failTask(ctx, expUUID, err)
		return fmt.Errorf("get archive storage object: %w", err)
	}

	// 1. Process experiment archive (zip)
	timeRange, err := h.processArchive(ctx, expUUID, archiveStorage)
	if err != nil {
		h.failTask(ctx, expUUID, err)
		return fmt.Errorf("process archive: %w", err)
	}

	// Update experiment with actual TimeRange spanning from the earliest
	// file start to the latest file stop across all LICEL files in the archive.
	exp.TimeRange = timeRange
	exp.UpdatedAt = time.Now()
	if err := h.experimentRepo.Update(ctx, exp); err != nil {
		h.failTask(ctx, expUUID, err)
		return fmt.Errorf("update experiment time range: %w", err)
	}

	// 2. Process background file (single LICEL file, optional)
	if exp.StorageRefs.BackgroundID != nil {
		bgStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.BackgroundID)
		if err != nil {
			h.failTask(ctx, expUUID, err)
			return fmt.Errorf("get background storage object: %w", err)
		}
		if err := h.processSingleLicelFile(ctx, expUUID, bgStorage, true); err != nil {
			h.failTask(ctx, expUUID, err)
			return fmt.Errorf("process background: %w", err)
		}
	}

	// 3. Process meteo file (optional)
	if exp.StorageRefs.MeteoID != nil {
		meteoStorage, err := h.storageObjRepo.FindByID(ctx, *exp.StorageRefs.MeteoID)
		if err != nil {
			h.failTask(ctx, expUUID, err)
			return fmt.Errorf("get meteo storage object: %w", err)
		}
		if _, err := h.processMeteoFile(ctx, expUUID, meteoStorage); err != nil {
			h.failTask(ctx, expUUID, err)
			return fmt.Errorf("process meteo: %w", err)
		}
	}

	h.updateTaskStatus(ctx, expUUID, domain.TaskCompleted, "")
	log.Printf("parse_experiment: experiment %s done", expUUID)
	return nil
}

// updateTaskStatus best-effort updates the task status in the database.
func (h *ParseExperimentHandler) updateTaskStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errMsg string) {
	if h.taskStatusRepo == nil {
		return
	}
	if err := h.taskStatusRepo.UpdateStatus(ctx, id, status, errMsg); err != nil {
		log.Printf("parse_experiment: update task status: %v", err)
	}
}

// failTask best-effort marks the task as failed with the given error.
func (h *ParseExperimentHandler) failTask(ctx context.Context, id uuid.UUID, err error) {
	if h.taskStatusRepo == nil {
		return
	}
	if err := h.taskStatusRepo.UpdateStatus(ctx, id, domain.TaskFailed, err.Error()); err != nil {
		log.Printf("parse_experiment: update task status: %v", err)
	}
}

// processArchive downloads a zip archive from MinIO, parses all LICEL files inside,
// creates LicelFile + LicelProfile records, and returns the global TimeRange
// spanning from the earliest start to the latest stop across all files.
func (h *ParseExperimentHandler) processArchive(
	ctx context.Context,
	expUUID uuid.UUID,
	storageObj *domain.StorageObject,
) (domain.TimeRange, error) {
	log.Printf("parse_experiment: processing archive %s/%s", storageObj.Path.Bucket, storageObj.Path.Path)

	var buf bytes.Buffer
	if err := h.fileStorage.Download(ctx, storageObj.Path.Bucket, storageObj.Path.Path, &buf); err != nil {
		return domain.TimeRange{}, fmt.Errorf("download archive: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "*.zip")
	if err != nil {
		return domain.TimeRange{}, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		return domain.TimeRange{}, fmt.Errorf("write temp file: %w", err)
	}

	lp, err := licelformat.NewLicelPackFromZip(tmpPath)
	if err != nil {
		return domain.TimeRange{}, fmt.Errorf("new licel pack: %w", err)
	}

	var globalStart, globalEnd time.Time
	for name, lf := range lp.Data {
		if globalStart.IsZero() || lf.MeasurementStartTime.Before(globalStart) {
			globalStart = lf.MeasurementStartTime
		}
		if globalEnd.IsZero() || lf.MeasurementStopTime.After(globalEnd) {
			globalEnd = lf.MeasurementStopTime
		}

		if err := h.createLicelFileRecords(ctx, expUUID, name, lf, storageObj.ID, false); err != nil {
			return domain.TimeRange{}, err
		}
	}

	if globalStart.IsZero() || globalEnd.IsZero() {
		return domain.TimeRange{}, fmt.Errorf("archive contains no licel files")
	}

	timeRange, err := domain.NewTimeRange(globalStart, globalEnd)
	if err != nil {
		return domain.TimeRange{}, fmt.Errorf("compute experiment time range: %w", err)
	}

	log.Printf("parse_experiment: archive processed — %d files, %d profiles", len(lp.Data), countProfiles(lp))
	return timeRange, nil
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

	altitudes, temperatures, pressures, err := parseMeteoCSV(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse meteo: %w", err)
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

	laserFreq := resolveLaserFrequency(lf)
	timeRange, _ := domain.NewTimeRange(lf.MeasurementStartTime, lf.MeasurementStopTime)
	licelFile := domain.NewLicelFile(
		expUUID,
		timeRange,
		int32(lf.NDatasets),
		laserFreq,
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

// resolveLaserFrequency returns the first non-zero laser frequency from the LICEL file.
func resolveLaserFrequency(lf licelformat.LicelFile) int32 {
	freq := lf.Laser1Freq
	if freq == 0 && lf.Laser2Freq != 0 {
		freq = lf.Laser2Freq
	}
	if freq == 0 && lf.Laser3Freq != 0 {
		freq = lf.Laser3Freq
	}
	return int32(freq)
}

// parseMeteoCSV parses a meteo CSV buffer. Expected format:
// first 4 header lines, then columns: pressure (hPa), altitude (m), temperature (°C).
// Converts units: m → km, °C → K, hPa → Pa.
func parseMeteoCSV(data []byte) (altitudes, temperatures, pressures []float64, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
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

		altitudes = append(altitudes, hght/1000.0)       // m → km
		temperatures = append(temperatures, temp+273.15) // °C → K
		pressures = append(pressures, pres*100)          // hPa → Pa
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, nil, fmt.Errorf("scan meteo: %w", scanErr)
	}

	if len(altitudes) == 0 {
		return nil, nil, nil, fmt.Errorf("meteo file has no data rows")
	}

	return altitudes, temperatures, pressures, nil
}

func countProfiles(lp *licelformat.LicelPack) int {
	n := 0
	for _, lf := range lp.Data {
		n += len(lf.Profiles)
	}
	return n
}
