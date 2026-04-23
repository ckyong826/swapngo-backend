package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type SwapProcessor interface {
	ProcessSwapEvent(ctx context.Context, orderID uuid.UUID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) error
}

type SwapHandler struct {
	Processor SwapProcessor
}

func NewSwapHandler(processor SwapProcessor) *SwapHandler {
	return &SwapHandler{
		Processor: processor,
	}
}

// HandleSwapInitiatedEvent processes the SwapInitiated message
func (h *SwapHandler) HandleSwapInitiatedEvent(ctx context.Context, msgData []byte) error {
	var event SwapInitiated
	if err := json.Unmarshal(msgData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("[SwapHandler] Processing event for OrderID: %s, TxDigest: %s\n", event.OrderID, event.TxDigest)

	if err := h.Processor.ProcessSwapEvent(ctx, event.OrderID, event.UserAddress, event.FromToken, event.ToToken, event.TxDigest, event.AmountPaid, event.ExpectedAmount); err != nil {
		log.Printf("[SwapHandler] ERROR processing swap %s: %v", event.OrderID, err)
		return err
	}

	return nil
}
