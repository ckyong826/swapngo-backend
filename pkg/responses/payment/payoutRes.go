package payment

type PayoutRes struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}