package transfer

type InitiateTransferReq struct {
	FromAddress string `json:"from_address" binding:"required"`
	ToAddress string `json:"to_address" binding:"required"`
	AmountMYRC float64 `json:"amount_myrc" binding:"required,gt=0"`
}