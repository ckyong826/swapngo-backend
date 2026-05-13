package swap

type InitiateSwapReq struct {
	FromToken       string  `json:"from_token" binding:"required"`
	ToToken         string  `json:"to_token" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	EstimatedAmount float64 `json:"estimated_amount" binding:"omitempty,gte=0"`
	Slippage        float64 `json:"slippage" binding:"omitempty,gte=0,lte=0.5"`
}
