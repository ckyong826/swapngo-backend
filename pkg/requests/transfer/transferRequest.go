package transfer

type InitiateTransferReq struct {
	Recipient string  `json:"recipient" binding:"required"`
	Token     string  `json:"token" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Pin       string  `json:"pin" binding:"required,len=4,numeric"`
}