package withdrawal

type InitiateWithdrawReq struct {
	AmountMYRC float64 `json:"amount_myrc" binding:"required,gt=0"`
	BankName   string  `json:"bank_name" binding:"required"`
	BankAccountNo string  `json:"bank_account_no" binding:"required"`
}