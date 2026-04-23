package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"log"

	config "swapngo-backend/pkg/configs"

	"github.com/segmentio/kafka-go"
)

var (
	writer *kafka.Writer
	once   sync.Once
)

// InitProducer initializes the global Kafka writer for producing messages.
func InitProducer() {
	once.Do(func() {
		brokers := strings.Split(config.Env.KafkaBrokers, ",")
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		}
		log.Println("[Kafka Producer] Globally initialized!")
	})
}

// PublishSwapInitiatedEvent sends a SwapInitiated event to the given topic
func PublishSwapInitiatedEvent(ctx context.Context, topic string, event SwapInitiated) error {
	if writer == nil {
		InitProducer()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.OrderID.String()),
		Value: payload,
	})
	if err == nil {
		log.Printf("[Kafka Producer] Successfully published SwapInitiated to %s (OrderID: %s)", topic, event.OrderID)
	}
	return err
}

// CloseProducer cleans up the producer resources
func CloseProducer() error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}

func PublishDepositWeb3InitiatedEvent(ctx context.Context, topic string, event DepositWeb3Initiated) error {
	if writer == nil { InitProducer() }
	payload, err := json.Marshal(event)
	if err != nil { return err }
	err = writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.DepositID.String()),
		Value: payload,
	})
	if err == nil {
		log.Printf("[Kafka Producer] Successfully published DepositWeb3 to %s (DepositID: %s)", topic, event.DepositID)
	}
	return err
}

func PublishWithdrawInitiatedEvent(ctx context.Context, topic string, event WithdrawInitiated) error {
	if writer == nil { InitProducer() }
	payload, err := json.Marshal(event)
	if err != nil { return err }
	err = writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.WithdrawID.String()),
		Value: payload,
	})
	if err == nil {
		log.Printf("[Kafka Producer] Successfully published WithdrawInitiated to %s (WithdrawID: %s)", topic, event.WithdrawID)
	}
	return err
}

func PublishTransferInitiatedEvent(ctx context.Context, topic string, event TransferInitiated) error {
	if writer == nil { InitProducer() }
	payload, err := json.Marshal(event)
	if err != nil { return err }
	err = writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.TransferID.String()),
		Value: payload,
	})
	if err == nil {
		log.Printf("[Kafka Producer] Successfully published TransferInitiated to %s (TransferID: %s)", topic, event.TransferID)
	}
	return err
}
