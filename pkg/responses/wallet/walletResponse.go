package wallet

type WalletResponse struct {
	ChainName string `json:"chain_name"`
	PublicAddress string `json:"public_address"`
	Balance   string `json:"balance"`
}