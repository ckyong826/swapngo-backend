package kyc

type RejectKYCRequest struct {
	Remarks string `json:"remarks" binding:"required"`
}
