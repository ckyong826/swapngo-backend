//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"swapngo-backend/internal/kafka"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	// The message we want to send to test the worker
	mockEvent := kafka.SwapInitiated{
		OrderID:        uuid.New(), // We would realistically use an existing Order ID in DB.
		UserAddress:    "0xTestUserAddress", // Replace with valid destination MYRC address
		FromToken:      "SUI",
		ToToken:        "MYRC",
		AmountPaid:     2.5,
		ExpectedAmount: 10.0,
		TxDigest:       "mock_sui_digest_12345", // Replace with a real digest from a Devnet/Testnet txn
	}

	payload, err := json.Marshal(mockEvent)
	if err != nil {
		log.Fatal(err)
	}

	// Make sure Kafka is running on this host/port. If using Docker, it's usually localhost:9092
	brokers := []string{"localhost:9092"}
	if envBroker := os.Getenv("KAFKA_BROKERS"); envBroker != "" {
		brokers = []string{envBroker}
	}

	w := &kafkago.Writer{
		Addr:     kafkago.TCP(brokers...),
		Topic:    "swap_events_topic",
		Balancer: &kafkago.LeastBytes{},
	}

	fmt.Printf("Publishing event for OrderID: %s\n", mockEvent.OrderID)
	err = w.WriteMessages(context.Background(),
		kafkago.Message{
			Key:   []byte(mockEvent.OrderID.String()),
			Value: payload,
		},
	)

	if err != nil {
		log.Fatalf("Failed to write message: %v", err)
	}

	if err := w.Close(); err != nil {
		log.Fatal("Failed to close writer:", err)
	}

	fmt.Println("Mock event successfully sent to Kafka!")
}
