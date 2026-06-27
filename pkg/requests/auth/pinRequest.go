package auth

type VerifyPinRequest struct {
	Pin string `json:"pin" binding:"required,len=4,numeric"`
}
