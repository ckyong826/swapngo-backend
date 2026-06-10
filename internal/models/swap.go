package models

import (
	"github.com/google/uuid"
)

type Swap struct {
	Base

	AccountID            uuid.UUID `gorm:"type:uuid;not null;index"`
	FromToken         TokenType    `gorm:"type:varchar(20);not null"` // 例如 "MYRC"
	ToToken           TokenType    `gorm:"type:varchar(20);not null"` // 例如 "SUI"
	
	FromAmount        float64   `gorm:"not null"`
	EstimatedToAmount float64   `gorm:"not null"` // 用户确认兑换时的期望金额
	ActualToAmount    float64   `gorm:"default:0"`// 链上实际换到的金额 (受滑点影响)
	SlippageTolerance float64   `gorm:"not null"` // 滑点容忍度，例如 0.01 代表 1%
	
	Status            string    `gorm:"type:varchar(30);default:'PENDING'"`
	TxHash            string    `gorm:"type:varchar(100)"`
}

const(
	SwapStatePending    = "PENDING"
	SwapStateProcessing = "PROCESSING"
	SwapStateSuccess    = "SUCCESS"
	SwapStateFailed     = "FAILED"

)

type TokenType string

const (
	SUI  TokenType = "SUI"
	MYRC TokenType = "MYRC"
	BTC  TokenType = "BTC"
	ETH  TokenType = "ETH"
	USDT TokenType = "USDT"
	USDC TokenType = "USDC"
)

// IsOffChain returns true for tokens that are NOT native to the SUI blockchain.
// Their balances are tracked in the token_balances table instead of on-chain.
func (t TokenType) IsOffChain() bool {
	switch t {
	case BTC, ETH, USDT, USDC:
		return true
	default:
		return false
	}
}	