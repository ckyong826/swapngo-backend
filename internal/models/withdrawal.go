package models

import (
	"github.com/google/uuid"
)

const (
	WithdrawStatePending        = "PENDING"
	WithdrawStateProcessingWeb3 = "PROCESSING_WEB3" // 正在链上扣款
	WithdrawStateProcessingFiat = "PROCESSING_FIAT" // 链上扣款成功，正在请求银行打款
	WithdrawStateSuccess        = "SUCCESS"         // 银行打款成功
	WithdrawStateFailed         = "FAILED"          // 失败（需要记录失败节点）
)

type Withdrawal struct {
	Base 

	AccountID        uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	AmountMYRC    float64   `gorm:"not null" json:"amount_myrc"`
	AmountMYR     float64   `gorm:"not null" json:"amount_myr"`
	
	BankName      string    `gorm:"type:varchar(100);not null" json:"bank_name"`
	BankAccountNo string    `gorm:"type:varchar(100);not null" json:"bank_account_no"`
	
	Status        string    `gorm:"type:varchar(30);default:'PENDING'" json:"status"`
	
	TxHash        string    `gorm:"type:varchar(100)" json:"tx_hash"`
	GatewayRefID     string    `gorm:"type:varchar(100)" json:"gateway_ref_id"`
}