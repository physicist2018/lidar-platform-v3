package ports

import "context"

// Subject is a NATS subject for task messages.
// Extend this list as needed.
type Subject string

const (
	SubjectParseExperiment   Subject = "lidar.task.parse_experiment"
	SubjectPrepareExperiment Subject = "lidar.task.prepare_experiment"
	SubjectProcessExperiment Subject = "lidar.task.process_experiment"
)

// Message represents a JetStream message.
type Message struct {
	Subject Subject
	Data    []byte
	DedupID string
}

// MessageHandler processes a single message.
// Return an error to trigger a Nak (redelivery).
type MessageHandler func(ctx context.Context, msg Message) error

// Subscription represents an active consumer subscription.
type Subscription interface {
	Unsubscribe() error
}

// MessageQueue provides async publish/subscribe over NATS JetStream.
type MessageQueue interface {
	// Publish sends a message to the given subject.
	// If dedupID is non-empty, JetStream will reject duplicates within the
	// stream's deduplication window.
	Publish(ctx context.Context, subject Subject, data []byte, dedupID string) error

	// Subscribe creates a durable JetStream consumer on the given subject.
	// The consumer name must be unique per subject.
	Subscribe(ctx context.Context, subject Subject, consumer string, handler MessageHandler) (Subscription, error)

	// Close drains and closes the NATS connection.
	Close() error
}
