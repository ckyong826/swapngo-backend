package requests

import "swapngo-backend/internal/models"

type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Pin         string `json:"pin" binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	CustodyType models.CustodyType `json:"custody_type" binding:"required"`
}