package models

import "github.com/google/uuid"

type Account struct {
	Base

	// foreign key
	userId uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	// relationship
	Wallets []Wallet `gorm:"foreignKey:AccountID" json:"wallets"`

	// account details
	accountName string `gorm:"not null" json:"account_name"`
	custodyType CustodyType `gorm:"not null" json:"custody_type"`
	status AccountStatus `gorm:"not null" json:"status"`
}

type AccountStatus string

const (
	AccountActive   AccountStatus = "ACTIVE"
	AccountInactive AccountStatus = "INACTIVE"
)

type CustodyType string

const (
	AccountServerManaged CustodyType = "SERVER"
	AccountSelfManaged   CustodyType = "SELF"
)	