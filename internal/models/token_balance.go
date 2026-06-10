package models

import "github.com/google/uuid"

// TokenBalance stores off-chain token holdings for tokens that do not natively
// exist on the SUI blockchain (BTC, ETH, USDT, USDC).
// One row per (AccountID, Token) pair — updated in-place on each swap.
type TokenBalance struct {
	Base

	AccountID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_account_token" json:"account_id"`
	Token     TokenType `gorm:"type:varchar(20);not null;uniqueIndex:idx_account_token" json:"token"`
	Balance   float64   `gorm:"not null;default:0" json:"balance"`
}
