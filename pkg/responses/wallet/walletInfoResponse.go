package wallet

type TokenBalance struct {
	Token    string  `json:"token"`
	Amount   float64 `json:"amount"`
	ValueMYR float64 `json:"value_myr"`
}

type WalletInfoResponse struct {
	SUIAddress    string         `json:"sui_address"`
	Email         string         `json:"email"`
	Balances      []TokenBalance `json:"balances"`
	TotalValueMYR float64        `json:"total_value_myr"`
}
