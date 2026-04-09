package models

type User struct {
	Base

	// user details
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	PhoneNumber string `gorm:"uniqueIndex;not null" json:"phone_number"`
	Email       string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"password_hash"`
	PinHash      string `gorm:"not null" json:"pin_hash"`
	KycStatus KycStatus `gorm:"not null;default:'PENDING'" json:"kyc_status"`

	// user accounts (one-to-many relationship)
	Accounts []Account `gorm:"foreignKey:UserID" json:"accounts"`

}

type KycStatus string

const (
	KycPending  KycStatus = "PENDING"
	KycApproved KycStatus = "APPROVED"
	KycRejected KycStatus = "REJECTED"
)