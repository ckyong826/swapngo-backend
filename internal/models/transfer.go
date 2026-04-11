package models

import (
	"github.com/google/uuid"
)

type Transfer struct {
	Base

	SenderAccountID   uuid.UUID `gorm:"type:uuid;not null;index"`
	// ReceiverID is optional: if the recipient address belongs to an internal platform user, record it for easier querying
	ReceiverAccountID *uuid.UUID `gorm:"type:uuid;index"` 
	
	ToAddress  string    `gorm:"type:varchar(100);not null"` // 目标的 SUI 地址
	Amount     float64   `gorm:"not null"`
	
	Status     string    `gorm:"type:varchar(30);default:'PENDING'"`
	TxHash     string    `gorm:"type:varchar(100)"` // 统一的凭证字段名
}

const (
	TransferStatePending    = "PENDING"
	TransferStateProcessing = "PROCESSING" // 正在链上转账
	TransferStateSuccess    = "SUCCESS"
	TransferStateFailed     = "FAILED"
)