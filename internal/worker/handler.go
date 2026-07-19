package worker

import (
	"context"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// TaskHandler is the interface for processing a single task type.
type TaskHandler interface {
	// Subject returns the NATS subject this handler listens to.
	Subject() ports.Subject

	// Handle processes a task. Return error to trigger Nak (redelivery).
	Handle(ctx context.Context, data []byte) error
}
