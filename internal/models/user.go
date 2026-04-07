package models

type User struct {
	Base

	// user details
	phoneNumber string `gorm:"uniqueIndex;not null" json:"phone_number"`
	email       string `gorm:"uniqueIndex" json:"email"`
	passwordHash string `gorm:"not null" json:"-"`
	pinHash      string `gorm:"not null" json:"-"`
	kycStatus KycStatus `gorm:"not null;default:'PENDING'" json:"kyc_status"`

	// user accounts (one-to-many relationship)
	accounts []Account `gorm:"foreignKey:UserID" json:"accounts"`

}

type KycStatus string

const (
	KycPending  KycStatus = "PENDING"
	KycApproved KycStatus = "APPROVED"
	KycRejected KycStatus = "REJECTED"
)