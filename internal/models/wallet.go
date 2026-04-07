package models

import "github.com/google/uuid"

type Wallet struct {
	Base
	// foreign key
	accountId uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`

	// relationship
	Account Account `gorm:"foreignKey:AccountID" json:"account"`

	// wallet details
	chainName ChainName `gorm:"not null" json:"chain_name"`
	address   string `gorm:"uniqueIndex;not null" json:"address"`
	status    WalletStatus `gorm:"not null;default:'ACTIVE'" json:"status"`

}

type WalletStatus string

const (
	WalletActive   WalletStatus = "ACTIVE"
	WalletInactive WalletStatus = "INACTIVE"
)

type ChainName string

const (
	ChainSui      ChainName = "SUI"
	ChainEthereum ChainName = "ETHEREUM"
	ChainSolana   ChainName = "SOLANA"
	ChainPolygon  ChainName = "POLYGON"
)