package auth

type LoginRequest struct {
	Username    string `json:"username" binding:"omitempty"`
	PhoneNumber string `json:"phone_number" binding:"omitempty"`
	Email       string `json:"email" binding:"omitempty"`
	Password    string `json:"password" binding:"required"`
}