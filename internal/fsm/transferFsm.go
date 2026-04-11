package fsm

import "swapngo-backend/internal/models"

const (
	TransferEventStart   = "START"
	TransferEventSuccess = "SUCCESS"
	TransferEventFailed  = "FAILED"
)

func BuildTransferFSM() *StateMachine {
	rules := []Transition{
		{From: models.TransferStatePending, Event: TransferEventStart, To: models.TransferStateProcessing},
		{From: models.TransferStateProcessing, Event: TransferEventSuccess, To: models.TransferStateSuccess},
		{From: models.TransferStateProcessing, Event: TransferEventFailed, To: models.TransferStateFailed},
	}
	return New(rules)
}