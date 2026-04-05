package models

type User struct {
	Base

	phoneNumber string `gorm:"uniqueIndex;not null" json:"phone_number"`
	email       string `gorm:"uniqueIndex" json:"email"`
	passwordHash string `gorm:"not null" json:"-"`
	pinHash      string `gorm:"not null" json:"-"`

	// user status
	kycStatus string `gorm:"not null;default:'pending'" json:"kyc_status"`

	// user wallets (one-to-many relationship)
	wallets   []Wallet `gorm:"foreignKey:userId" json:"wallets"`
}