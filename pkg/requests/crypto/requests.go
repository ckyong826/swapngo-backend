package crypto

// DepositSUIReq claims a real on-chain SUI deposit by its transaction digest.
// The user has already sent testnet SUI to the treasury address; the backend
// verifies the transfer and credits the ledger.
type DepositSUIReq struct {
	TxDigest string  `json:"tx_digest" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
}

// SimulateDepositReq credits a simulated (mocked-chain) token deposit.
// Valid tokens: BTC, ETH, USDT, USDC.
type SimulateDepositReq struct {
	Token  string  `json:"token" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// WithdrawReq withdraws crypto out of the ledger. SUI is sent on-chain for real;
// BTC/ETH/USDT/USDC are simulated (ledger debit + synthetic txid).
type WithdrawReq struct {
	Token     string  `json:"token" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	ToAddress string  `json:"to_address"`
}
