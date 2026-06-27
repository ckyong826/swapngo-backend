package withdrawal

type DestinationDetails struct {
	BankAccount string `json:"bank_account"`
	BankName    string `json:"bank_name"`
	SUIAddress  string `json:"sui_address"`
}

type InitiateWithdrawReq struct {
	DestinationType    string             `json:"destination_type" binding:"required,oneof=bank sui_wallet"`
	DestinationDetails DestinationDetails `json:"destination_details" binding:"required"`
	Token              string             `json:"token" binding:"required"`
	Amount             float64            `json:"amount" binding:"required,gt=0"`
	Pin                string             `json:"pin" binding:"required,len=4,numeric"`
}