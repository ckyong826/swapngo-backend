package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type DepositProcessor interface {
	ProcessDepositEvent(ctx context.Context, depositID uuid.UUID, accountID string, amount float64) error
}

type DepositHandler struct {
	Processor DepositProcessor
}

func NewDepositHandler(processor DepositProcessor) *DepositHandler {
	return &DepositHandler{Processor: processor}
}

func (h *DepositHandler) HandleDepositEvent(ctx context.Context, msgData []byte) error {
	var event DepositWeb3Initiated
	if err := json.Unmarshal(msgData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal deposit event: %w", err)
	}
	log.Printf("[DepositHandler] Processing web3 init for DepositID: %s", event.DepositID)
	if err := h.Processor.ProcessDepositEvent(ctx, event.DepositID, event.AccountID, event.Amount); err != nil {
		log.Printf("[DepositHandler] ERROR processing deposit %s: %v", event.DepositID, err)
		return err
	}
	return nil
}
