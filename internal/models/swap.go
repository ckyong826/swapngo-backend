package models

import (
	"github.com/google/uuid"
)

type Swap struct {
	Base

	AccountID            uuid.UUID `gorm:"type:uuid;not null;index"`
	FromToken         string    `gorm:"type:varchar(20);not null"` // 例如 "MYRC"
	ToToken           string    `gorm:"type:varchar(20);not null"` // 例如 "SUI"
	
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