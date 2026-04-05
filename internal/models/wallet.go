package models

import "github.com/google/uuid"

type Wallet struct {
	Base
	// foreign key to user
	userId uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	// wallet details
	chainName string `gorm:"not null" json:"chain_name"`
	address   string `gorm:"uniqueIndex;not null" json:"address"`
	walletType string `gorm:"not null;default:'server_managed'" json:"wallet_type"`
	isDefault  bool   `gorm:"not null;default:false" json:"is_default"`
	status     string `gorm:"not null;default:'active'" json:"status"`
}