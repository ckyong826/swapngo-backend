package models

import "github.com/google/uuid"

const (
	KYCStatusPending  = "PENDING"
	KYCStatusApproved = "APPROVED"
	KYCStatusRejected = "REJECTED"
)

type KYC struct {
	Base
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	FullName     string    `gorm:"type:varchar(255);not null" json:"full_name"`
	ICNumber     string    `gorm:"type:text;not null" json:"-"` // AES-encrypted
	ICFrontPhoto string    `gorm:"type:text;not null" json:"-"` // AES-encrypted base64
	ICBackPhoto  string    `gorm:"type:text;not null" json:"-"` // AES-encrypted base64
	Status       string    `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
}
