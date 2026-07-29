package messaging

import "time"

const (
	defaultStreamName = "lidar-tasks"
	defaultAckWait    = 30 * time.Minute
)

// Config holds the NATS connection and JetStream settings.
type Config struct {
	URL        string // nats://localhost:4222
	StreamName string // JetStream stream name (default: "lidar-tasks")
	AckWait    time.Duration
}
