package swap

type InitiateSwapReq struct {
	FromToken       string  `json:"from_token" binding:"required"`
	ToToken         string  `json:"to_token" binding:"required"`
	FromAmount      float64 `json:"from_amount" binding:"required,gt=0"`
	EstimatedAmount float64 `json:"estimated_amount" binding:"required,gt=0"`
	Slippage        float64 `json:"slippage" binding:"required,gte=0,lte=0.5"` // 最高容忍 50% 滑点
}
