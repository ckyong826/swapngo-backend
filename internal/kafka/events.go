package kafka

import "github.com/google/uuid"

// SwapInitiated represents a message that comes from the system when a user initiates a swap
type SwapInitiated struct {
	OrderID        uuid.UUID `json:"order_id"`
	UserAddress    string    `json:"user_address"`
	FromToken      string    `json:"from_token"`
	ToToken        string    `json:"to_token"`
	AmountPaid     float64   `json:"amount_paid"`
	ExpectedAmount float64   `json:"expected_amount"`
	TxDigest       string    `json:"tx_digest"`
}

type DepositWeb3Initiated struct {
	DepositID uuid.UUID `json:"deposit_id"`
	AccountID string    `json:"account_id"`
	Amount    float64   `json:"amount"`
}

type WithdrawInitiated struct {
	WithdrawID    uuid.UUID `json:"withdraw_id"`
	UserID        string    `json:"user_id"`
	AmountMYRC    float64   `json:"amount_myrc"`
	AmountMYR     float64   `json:"amount_myr"`
	BankName      string    `json:"bank_name"`
	BankAccountNo string    `json:"bank_account_no"`
}

type TransferInitiated struct {
	TransferID  uuid.UUID `json:"transfer_id"`
	SenderID    string    `json:"sender_id"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Amount      float64   `json:"amount"`
}
