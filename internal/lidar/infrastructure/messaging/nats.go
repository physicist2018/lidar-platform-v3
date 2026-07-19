package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// NatsMessageQueue implements ports.MessageQueue over NATS JetStream.
type NatsMessageQueue struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	cfg    Config
	stream jetstream.Stream
}

// NewNatsMessageQueue connects to NATS, initialises JetStream, and
// ensures the task stream exists.
func NewNatsMessageQueue(cfg Config) (*NatsMessageQueue, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	streamName := cfg.StreamName
	if streamName == "" {
		streamName = defaultStreamName
	}
	ackWait := cfg.AckWait
	if ackWait == 0 {
		ackWait = defaultAckWait
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure the stream exists (create if missing).
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:       streamName,
		Subjects:   []string{"lidar.task.>"},
		Storage:    jetstream.FileStorage,
		Duplicates: 2 * time.Minute,
	})
	if err != nil {
		// It's fine if the stream already exists — update it instead.
		stream, err = js.UpdateStream(ctx, jetstream.StreamConfig{
			Name:       streamName,
			Subjects:   []string{"lidar.task.>"},
			Storage:    jetstream.FileStorage,
			Duplicates: 2 * time.Minute,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("nats: stream setup: %w", err)
		}
	}

	return &NatsMessageQueue{
		conn:   conn,
		js:     js,
		stream: stream,
		cfg: Config{
			StreamName: streamName,
			AckWait:    ackWait,
		},
	}, nil
}

// Publish sends a message via JetStream with an optional dedup ID.
func (q *NatsMessageQueue) Publish(ctx context.Context, subject ports.Subject, data []byte, dedupID string) error {
	opts := []jetstream.PublishOpt{}
	if dedupID != "" {
		opts = append(opts, jetstream.WithMsgID(dedupID))
	}

	_, err := q.js.Publish(ctx, string(subject), data, opts...)
	if err != nil {
		return fmt.Errorf("nats: publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe creates a durable JetStream consumer and starts consuming
// messages in a background goroutine via the callback.
func (q *NatsMessageQueue) Subscribe(
	ctx context.Context,
	subject ports.Subject,
	consumer string,
	handler ports.MessageHandler,
) (ports.Subscription, error) {
	cons, err := q.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		FilterSubject: string(subject),
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       q.cfg.AckWait,
	})
	if err != nil {
		return nil, fmt.Errorf("nats: consumer %s: %w", consumer, err)
	}

	consCtx, err := cons.Consume(func(msg jetstream.Msg) {
		q.processMessage(msg, handler)
	})
	if err != nil {
		return nil, fmt.Errorf("nats: consume %s: %w", consumer, err)
	}

	return &natsSubscription{stop: consCtx.Stop}, nil
}

// Close drains the NATS connection.
func (q *NatsMessageQueue) Close() error {
	if err := q.conn.Drain(); err != nil {
		return fmt.Errorf("nats: drain: %w", err)
	}
	q.conn.Close()
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (q *NatsMessageQueue) processMessage(msg jetstream.Msg, handler ports.MessageHandler) {
	m := ports.Message{
		Subject: ports.Subject(msg.Subject()),
		Data:    msg.Data(),
	}
	if h := msg.Headers(); h != nil {
		m.DedupID = h.Get("Nats-Msg-Id")
	}

	if err := handler(context.Background(), m); err != nil {
		msg.Nak()
		return
	}
	msg.Ack()
}

// natsSubscription wraps a ConsumeContext stop function.
type natsSubscription struct {
	stop func()
}

func (s *natsSubscription) Unsubscribe() error {
	s.stop()
	return nil
}
