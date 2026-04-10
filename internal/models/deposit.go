package models

import (
	"github.com/google/uuid"
)

const (
	DepositStatePending        = "PENDING"          
	DepositStateProcessingWeb3 = "PROCESSING_WEB3"  // Already process the fund but web3 havent mint
	DepositStateSuccess        = "SUCCESS"          
	DepositStateFailed         = "FAILED"           
)

type Deposit struct {
	Base
	// foreign key
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	
	AmountMYR    float64   `gorm:"not null"`
	AmountMYRC   float64   `gorm:"not null"`
	Status       string    `gorm:"type:varchar(30);default:'PENDING'"`
	GatewayRefID string    `gorm:"type:varchar(100);uniqueIndex"` 
	TxHash       string    `gorm:"type:varchar(100)"`

}