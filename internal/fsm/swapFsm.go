package fsm

import "swapngo-backend/internal/models"


const (
	SwapEventStart   = "START"
	SwapEventSuccess = "SUCCESS"
	SwapEventFailed  = "FAILED"
)

func BuildSwapFSM() *StateMachine {
	rules := []Transition{
		{From: models.SwapStatePending, Event: SwapEventStart, To: models.SwapStateProcessing},
		{From: models.SwapStateProcessing, Event: SwapEventSuccess, To: models.SwapStateSuccess},
		{From: models.SwapStateProcessing, Event: SwapEventFailed, To: models.SwapStateFailed},
	}
	return New(rules)
}