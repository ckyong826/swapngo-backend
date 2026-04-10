package fsm

import (
	"swapngo-backend/internal/models"
)

const (
	DepositEventPaymentSuccess = "PAYMENT_SUCCESS"
	DepositEventPaymentFailed  = "PAYMENT_FAILED"
	DepositEventWeb3Success    = "WEB3_SUCCESS"
	DepositEventWeb3Failed     = "WEB3_FAILED"
)

func BuildDepositFSM() *StateMachine {
	rules := []Transition{
		{From: models.DepositStatePending, Event: DepositEventPaymentSuccess, To: models.DepositStateProcessingWeb3},
		{From: models.DepositStatePending, Event: DepositEventPaymentFailed, To: models.DepositStateFailed},
		{From: models.DepositStateProcessingWeb3, Event: DepositEventWeb3Success, To: models.DepositStateSuccess},
		{From: models.DepositStateProcessingWeb3, Event: DepositEventWeb3Failed, To: models.DepositStateFailed},
	}
	return New(rules)
}