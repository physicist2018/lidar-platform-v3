package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// natsTaskMessage is the top-level NATS message structure produced by CreateTaskUseCase.
type natsTaskMessage struct {
	TaskID    string          `json:"task_id"`
	Subject   string          `json:"subject"`
	TaskType  string          `json:"task_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"created_at"`
}

// PrepareExperimentPayload is the inner payload for a prepare_experiment task.
type PrepareExperimentPayload struct {
	ExperimentID   string  `json:"experiment_id"`
	BackgroundType string  `json:"background_type"`
	BackgroundFrom float32 `json:"background_from"` // meters
	TrimFrom       float32 `json:"trim_from"`       // meters
}

// PrepareExperimentHandler handles lidar.task.prepare_experiment tasks.
type PrepareExperimentHandler struct {
	licelProfileRepo    ports.LicelProfileRepository
	preparedMetaRepo    ports.PreparedMetaRepository
	preparedProfileRepo ports.PreparedProfileRepository
	taskStatusRepo      ports.TaskStatusRepository
}

// NewPrepareExperimentHandler creates a new PrepareExperimentHandler.
func NewPrepareExperimentHandler(
	licelProfileRepo ports.LicelProfileRepository,
	preparedMetaRepo ports.PreparedMetaRepository,
	preparedProfileRepo ports.PreparedProfileRepository,
	taskStatusRepo ports.TaskStatusRepository,
) *PrepareExperimentHandler {
	return &PrepareExperimentHandler{
		licelProfileRepo:    licelProfileRepo,
		preparedMetaRepo:    preparedMetaRepo,
		preparedProfileRepo: preparedProfileRepo,
		taskStatusRepo:      taskStatusRepo,
	}
}

func (h *PrepareExperimentHandler) Subject() ports.Subject {
	return ports.SubjectPrepareExperiment
}

func (h *PrepareExperimentHandler) Handle(ctx context.Context, data []byte) error {
	// 1. Parse top-level NATS message.
	var msg natsTaskMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	taskUUID, err := uuid.Parse(msg.TaskID)
	if err != nil {
		return fmt.Errorf("parse task_id: %w", err)
	}

	// 2. Parse payload.
	var payload PrepareExperimentPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	expUUID, err := uuid.Parse(payload.ExperimentID)
	if err != nil {
		return fmt.Errorf("parse experiment_id: %w", err)
	}

	bgType := domain.BackgroundType(payload.BackgroundType)
	if bgType != domain.BackgroundFromFile && bgType != domain.BackgroundMean {
		return fmt.Errorf("unknown background_type: %q", payload.BackgroundType)
	}
	if payload.TrimFrom <= 0 {
		return fmt.Errorf("trim_from must be positive, got %f", payload.TrimFrom)
	}
	if bgType == domain.BackgroundMean && payload.BackgroundFrom <= 0 {
		return fmt.Errorf("background_from must be positive for mean type, got %f", payload.BackgroundFrom)
	}

	log.Printf("prepare_experiment: processing experiment %s (bg=%s, bg_from=%.0f, trim=%.0f)",
		expUUID, bgType, payload.BackgroundFrom, payload.TrimFrom)

	h.updateTaskStatus(ctx, taskUUID, domain.TaskProcessing, "")

	// 3. Create PreparedMeta.
	meta := domain.NewPreparedMeta(expUUID, bgType, payload.BackgroundFrom, payload.TrimFrom)
	if err := h.preparedMetaRepo.Create(ctx, &meta); err != nil {
		h.failTask(ctx, taskUUID, fmt.Errorf("create prepared meta: %w", err))
		return fmt.Errorf("create prepared meta: %w", err)
	}

	// 4. Fetch paired profiles (signal + background).
	pairedProfiles, err := h.licelProfileRepo.FindProfilesWithBackground(ctx, expUUID)
	if err != nil {
		h.failTask(ctx, taskUUID, fmt.Errorf("find profiles: %w", err))
		return fmt.Errorf("find profiles: %w", err)
	}
	if len(pairedProfiles) == 0 {
		err := fmt.Errorf("no profiles found for experiment %s", expUUID)
		h.failTask(ctx, taskUUID, err)
		return err
	}

	log.Printf("prepare_experiment: found %d paired profiles for experiment %s",
		len(pairedProfiles), expUUID)

	// 5. Process each paired profile.
	for i, pp := range pairedProfiles {
		if i%100 == 0 {
			log.Printf("prepare_experiment: progress %d/%d profiles", i, len(pairedProfiles))
		}
		prepared, err := h.processProfile(pp, meta, bgType, payload)
		if err != nil {
			h.failTask(ctx, taskUUID, fmt.Errorf("process profile %s: %w", pp.Signal.ProfileID, err))
			return fmt.Errorf("process profile %s: %w", pp.Signal.ProfileID, err)
		}
		if err := h.preparedProfileRepo.Create(ctx, &prepared); err != nil {
			h.failTask(ctx, taskUUID, fmt.Errorf("save prepared profile: %w", err))
			return fmt.Errorf("save prepared profile: %w", err)
		}
	}

	h.updateTaskStatus(ctx, taskUUID, domain.TaskCompleted, "")
	log.Printf("prepare_experiment: experiment %s done — %d profiles processed", expUUID, len(pairedProfiles))
	return nil
}

// medianFilter3 applies a median filter with window size 3 to the input slice.
// At boundaries (i=0 and i=n-1) it falls back to the mean of the two available neighbors.
func medianFilter3(data []float32) []float32 {
	if len(data) == 0 {
		return nil
	}
	if len(data) == 1 {
		result := make([]float32, 1)
		copy(result, data)
		return result
	}
	result := make([]float32, len(data))
	for i := range data {
		switch {
		case i == 0:
			result[i] = (data[0] + data[1]) / 2
		case i == len(data)-1:
			result[i] = (data[len(data)-2] + data[len(data)-1]) / 2
		default:
			a, b, c := data[i-1], data[i], data[i+1]
			// Sort three values, pick the middle one.
			if a <= b && a <= c {
				if b <= c {
					result[i] = b
				} else {
					result[i] = c
				}
			} else if b <= a && b <= c {
				if a <= c {
					result[i] = a
				} else {
					result[i] = c
				}
			} else {
				if a <= b {
					result[i] = a
				} else {
					result[i] = b
				}
			}
		}
	}
	return result
}

// processProfile performs background removal and trimming on a single paired profile.
func (h *PrepareExperimentHandler) processProfile(
	pp domain.PairedProfile,
	meta domain.PreparedMeta,
	bgType domain.BackgroundType,
	payload PrepareExperimentPayload,
) (domain.PreparedProfile, error) {
	signalData := pp.Signal.Data
	binWidth := float64(pp.Signal.BinWidth)
	if binWidth <= 0 {
		return domain.PreparedProfile{}, fmt.Errorf("invalid bin_width %f for profile %s", binWidth, pp.Signal.ProfileID)
	}

	// Convert signal from []float64 to []float32.
	result := make([]float32, len(signalData))
	for i, v := range signalData {
		result[i] = float32(v)
	}

	// Background subtraction.
	switch bgType {
	case domain.BackgroundFromFile:
		if pp.Background != nil {
			bgData := pp.Background.Data
			n := min(len(bgData), len(result))
			for i := range n {
				result[i] = float32(signalData[i] - bgData[i])
			}
		}
		// No background file — keep original signal.

	case domain.BackgroundMean:
		tailStart := int(payload.BackgroundFrom / float32(binWidth))
		if tailStart >= len(result) {
			tailStart = len(result) / 2
		}
		if tailStart < len(result) {
			// Apply median filter (window=3) to remove noise spikes,
			// then calculate mean on the smoothed tail for a robust
			// background estimate.
			smoothed := medianFilter3(result)
			var sum float64
			count := 0
			for _, v := range smoothed[tailStart:] {
				sum += float64(v)
				count++
			}
			if count > 0 {
				mean := float32(sum / float64(count))
				for i := range smoothed {
					smoothed[i] -= mean
				}
				copy(result, smoothed)
			}
		}
	}

	// Trim by trim_from.
	trimBins := int(payload.TrimFrom / float32(binWidth))
	if trimBins > 0 && trimBins < len(result) {
		result = result[:trimBins]
	}

	return domain.PreparedProfile{
		PreparedMetaID: meta.ID,
		LicelProfileID: pp.Signal.ProfileID,
		Data:           result,
	}, nil
}

// updateTaskStatus best-effort updates the task status in the database.
func (h *PrepareExperimentHandler) updateTaskStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errMsg string) {
	if h.taskStatusRepo == nil {
		return
	}
	if err := h.taskStatusRepo.UpdateStatus(ctx, id, status, errMsg); err != nil {
		log.Printf("prepare_experiment: update task status: %v", err)
	}
}

// failTask best-effort marks the task as failed with the given error.
func (h *PrepareExperimentHandler) failTask(ctx context.Context, id uuid.UUID, err error) {
	if h.taskStatusRepo == nil {
		return
	}
	if err := h.taskStatusRepo.UpdateStatus(ctx, id, domain.TaskFailed, err.Error()); err != nil {
		log.Printf("prepare_experiment: update task status: %v", err)
	}
}
