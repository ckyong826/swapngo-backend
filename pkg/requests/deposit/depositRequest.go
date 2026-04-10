package requests

type InitiateDepositReq struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type WebhookReq struct {
	// 假设这是 Billplz/Stripe 传来的参数
	ID    string `json:"id"`
	State string `json:"state"` // 例如 "paid" 或 "due"
}