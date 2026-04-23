package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type WithdrawProcessor interface {
	ProcessWithdrawEvent(ctx context.Context, withdrawID uuid.UUID, userID string, amountMYRC, amountMYR float64, bankName, bankAccountNo string) error
}

type WithdrawHandler struct {
	Processor WithdrawProcessor
}

func NewWithdrawHandler(processor WithdrawProcessor) *WithdrawHandler {
	return &WithdrawHandler{Processor: processor}
}

func (h *WithdrawHandler) HandleWithdrawEvent(ctx context.Context, msgData []byte) error {
	var event WithdrawInitiated
	if err := json.Unmarshal(msgData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal withdraw event: %w", err)
	}
	log.Printf("[WithdrawHandler] Processing init for WithdrawID: %s", event.WithdrawID)
	if err := h.Processor.ProcessWithdrawEvent(ctx, event.WithdrawID, event.UserID, event.AmountMYRC, event.AmountMYR, event.BankName, event.BankAccountNo); err != nil {
		log.Printf("[WithdrawHandler] ERROR processing withdraw %s: %v", event.WithdrawID, err)
		return err
	}
	return nil
}
