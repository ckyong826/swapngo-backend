package kafka

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Worker serves as the core Kafka consumer engine.
type Worker struct {
	reader *kafka.Reader
	router map[string]MessageHandler
}

// MessageHandler defines the func signature for processing events
type MessageHandler func(ctx context.Context, msg []byte) error

// NewWorker initializes a new Worker
func NewWorker(brokers []string, groupID string, topics []string) *Worker {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		GroupTopics:    topics,
		StartOffset:    kafka.FirstOffset, // Ensure it picks up uncommitted messages
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second * 1,
	})

	return &Worker{
		reader: r,
		router: make(map[string]MessageHandler),
	}
}

// Register route a specific topic to a handler
func (w *Worker) Register(topic string, handler MessageHandler) {
	w.router[topic] = handler
}

// Start begins consuming messages
func (w *Worker) Start(ctx context.Context) {
	log.Println("[Antigravity Worker] Started listening to Kafka")
	for {
		// Use a bounded context for each fetch
		m, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("[Antigravity Worker] Kafka reader shutting down")
				return
			}
			log.Printf("[Antigravity Worker] Error fetching message: %v\n", err)
			time.Sleep(3 * time.Second) // Prevent spam while broker orchestrates new topics
			continue
		}

		log.Printf("[Antigravity Worker] Fetched Message | Topic: %s | Key: %s", m.Topic, string(m.Key))

		handler, ok := w.router[m.Topic]
		if !ok {
			log.Printf("[Antigravity Worker] No handler for topic: %s\n", m.Topic)
			w.commitSafely(ctx, m)
			continue
		}

		// Execute handler
		err = handler(ctx, m.Value)
		if err != nil {
			// Basic error handling/retry logic
			// If verification failed on-chain because it's not ready yet, we might want to NOT commit
			// But for now, just log. The uncommitted message will be re-fetched after timeout.
			log.Printf("[Antigravity Worker] Error processing message %v: %v\n", string(m.Key), err)
			// Apply a slight backoff
			time.Sleep(2 * time.Second)
		} else {
			// On success, commit
			w.commitSafely(ctx, m)
		}
	}
}

func (w *Worker) commitSafely(ctx context.Context, m kafka.Message) {
	if err := w.reader.CommitMessages(ctx, m); err != nil {
		log.Printf("[Antigravity Worker] Failed to commit message: %v\n", err)
	}
}

// Close gracefully closes the reader
func (w *Worker) Close() error {
	return w.reader.Close()
}
