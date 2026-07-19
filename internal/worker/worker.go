package worker

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/ports"
)

// Worker listens for tasks on NATS and dispatches them to registered handlers.
type Worker struct {
	msgQueue ports.MessageQueue
	handlers []TaskHandler
	subs     []ports.Subscription
	mu       sync.Mutex
}

// New creates a new Worker.
func New(msgQueue ports.MessageQueue) *Worker {
	return &Worker{
		msgQueue: msgQueue,
	}
}

// Register adds a handler to the worker.
func (w *Worker) Register(h ...TaskHandler) {
	w.handlers = append(w.handlers, h...)
}

// Run subscribes to all registered handlers' subjects and starts processing.
// Blocks until ctx is cancelled or a fatal error occurs.
func (w *Worker) Run(ctx context.Context) error {
	if len(w.handlers) == 0 {
		return fmt.Errorf("worker: no handlers registered")
	}

	for _, h := range w.handlers {
		h := h
		sub, err := w.msgQueue.Subscribe(ctx, h.Subject(), string(h.Subject())+"-worker",
			func(_ context.Context, msg ports.Message) error {
				log.Printf("worker: received task %s (dedup=%q)", msg.Subject, msg.DedupID)
				if err := h.Handle(context.Background(), msg.Data); err != nil {
					log.Printf("worker: handler %s failed: %v", msg.Subject, err)
					return err
				}
				log.Printf("worker: handler %s completed", msg.Subject)
				return nil
			})
		if err != nil {
			return fmt.Errorf("worker: subscribe %s: %w", h.Subject(), err)
		}

		w.mu.Lock()
		w.subs = append(w.subs, sub)
		w.mu.Unlock()

		log.Printf("worker: subscribed to %s", h.Subject())
	}

	<-ctx.Done()
	log.Println("worker: shutting down...")
	return nil
}

// Close unsubscribes all subscriptions.
func (w *Worker) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, sub := range w.subs {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("worker: unsubscribe error: %v", err)
		}
	}
}
