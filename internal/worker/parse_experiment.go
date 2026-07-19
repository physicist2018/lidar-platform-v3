package worker

import (
	"context"
	"log"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// ParseExperimentHandler handles lidar.task.parse_experiment tasks.
// It parses uploaded LICEL files and creates LicelFile / LicelProfile records.
type ParseExperimentHandler struct {
	experimentRepo ports.ExperimentRepository
	storageObjRepo ports.StorageObjectRepository
	fileStorage    ports.FileStorage
	// Additional repos (LicelFile, LicelProfile) will be added later.
}

// NewParseExperimentHandler creates a new ParseExperimentHandler.
func NewParseExperimentHandler(
	experimentRepo ports.ExperimentRepository,
	storageObjRepo ports.StorageObjectRepository,
	fileStorage ports.FileStorage,
) *ParseExperimentHandler {
	return &ParseExperimentHandler{
		experimentRepo: experimentRepo,
		storageObjRepo: storageObjRepo,
		fileStorage:    fileStorage,
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
	// TODO: implement actual parsing logic
	return nil
}
