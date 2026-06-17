package models

import "github.com/google/uuid"

// CryptoTxn records a crypto deposit or withdrawal against the ledger.
// Under the CEX ledger model the actual balance lives in token_balances; this
// table is the audit/history trail for crypto in/out movements.
type CryptoTxn struct {
	Base

	AccountID uuid.UUID          `gorm:"type:uuid;not null;index" json:"account_id"`
	Token     TokenType          `gorm:"type:varchar(20);not null" json:"token"`
	Direction CryptoTxnDirection `gorm:"type:varchar(20);not null" json:"direction"`
	Amount    float64            `gorm:"not null" json:"amount"`

	// TxRef holds the on-chain digest (real SUI deposit/withdraw) or a synthetic
	// reference (simulated tokens). For deposits it is also used to guard replay.
	TxRef string `gorm:"type:varchar(120);index" json:"tx_ref"`

	// Simulated is true for tokens whose chain is mocked (BTC/ETH/USDT/USDC).
	Simulated bool   `gorm:"not null;default:false" json:"simulated"`
	Status    string `gorm:"type:varchar(20);not null;default:'SUCCESS'" json:"status"`
}

type CryptoTxnDirection string

const (
	CryptoDirectionDeposit  CryptoTxnDirection = "DEPOSIT"
	CryptoDirectionWithdraw CryptoTxnDirection = "WITHDRAW"
)
