package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type TransferProcessor interface {
	ProcessTransferEvent(ctx context.Context, transferID uuid.UUID, senderID, fromAddress, toAddress string, amount float64) error
}

type TransferHandler struct {
	Processor TransferProcessor
}

func NewTransferHandler(processor TransferProcessor) *TransferHandler {
	return &TransferHandler{Processor: processor}
}

func (h *TransferHandler) HandleTransferEvent(ctx context.Context, msgData []byte) error {
	var event TransferInitiated
	if err := json.Unmarshal(msgData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal transfer event: %w", err)
	}
	log.Printf("[TransferHandler] Processing init for TransferID: %s", event.TransferID)
	if err := h.Processor.ProcessTransferEvent(ctx, event.TransferID, event.SenderID, event.FromAddress, event.ToAddress, event.Amount); err != nil {
		log.Printf("[TransferHandler] ERROR processing transfer %s: %v", event.TransferID, err)
		return err
	}
	return nil
}
