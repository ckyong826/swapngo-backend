package transfer

type InitiateTransferReq struct {
	ReceiverUserID string `json:"receiver_user_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}