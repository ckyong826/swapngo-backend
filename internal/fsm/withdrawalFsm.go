// internal/fsm/withdraw_fsm.go
package fsm

import "swapngo-backend/internal/models"

const (
	WithdrawEventStartWeb3   = "START_WEB3"
	WithdrawEventWeb3Success = "WEB3_SUCCESS"
	WithdrawEventWeb3Failed  = "WEB3_FAILED"
	
	WithdrawEventFiatSuccess = "FIAT_SUCCESS"
	WithdrawEventFiatFailed  = "FIAT_FAILED"
)

func BuildWithdrawFSM() *StateMachine {
	rules := []Transition{
		// 1. 发起链上扣款
		{From: models.WithdrawStatePending, Event: WithdrawEventStartWeb3, To: models.WithdrawStateProcessingWeb3},
		
		// 2. 链上扣款结果
		{From: models.WithdrawStateProcessingWeb3, Event: WithdrawEventWeb3Success, To: models.WithdrawStateProcessingFiat},
		{From: models.WithdrawStateProcessingWeb3, Event: WithdrawEventWeb3Failed, To: models.WithdrawStateFailed},
		
		// 3. 银行打款结果
		{From: models.WithdrawStateProcessingFiat, Event: WithdrawEventFiatSuccess, To: models.WithdrawStateSuccess},
		{From: models.WithdrawStateProcessingFiat, Event: WithdrawEventFiatFailed, To: models.WithdrawStateFailed},
	}
	return New(rules)
}