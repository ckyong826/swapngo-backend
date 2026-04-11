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
	AccountID       uuid.UUID `gorm:"type:uuid;not null" json:"account_id"`
	
	AmountMYR    float64   `gorm:"not null" json:"amount_myr"`
	AmountMYRC   float64   `gorm:"not null" json:"amount_myrc"`
	Status       string    `gorm:"type:varchar(30);default:'PENDING'" json:"status"`
	GatewayRefID string    `gorm:"type:varchar(100);uniqueIndex" json:"gateway_ref_id"` 
	TxHash       string    `gorm:"type:varchar(100)" json:"tx_hash"`

}