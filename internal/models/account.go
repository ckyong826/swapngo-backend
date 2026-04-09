package models

import "github.com/google/uuid"

type Account struct {
	Base

	// foreign key
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	// relationship
	Wallets []Wallet `gorm:"foreignKey:AccountID" json:"wallets"`

	// account details
	AccountName string `gorm:"not null" json:"account_name"`
	CustodyType CustodyType `gorm:"not null" json:"custody_type"`
	Status AccountStatus `gorm:"not null" json:"status"`
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