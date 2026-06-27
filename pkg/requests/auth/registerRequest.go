package auth

import "swapngo-backend/internal/models"

type RegisterRequest struct {
	Email       string             `json:"email" binding:"required,email"`
	Password    string             `json:"password" binding:"required"`
	Username    string             `json:"username" binding:"omitempty"`
	PhoneNumber string             `json:"phone_number" binding:"omitempty"`
	Pin         string             `json:"pin" binding:"required,len=4,numeric"`
	AccountName string             `json:"account_name" binding:"omitempty"`
	CustodyType models.CustodyType `json:"custody_type" binding:"omitempty"`
}