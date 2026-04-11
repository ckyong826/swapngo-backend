package handlers

import (
	"swapngo-backend/internal/bizs"

	"github.com/gin-gonic/gin"
)

type SwapHandler struct {
	swapBiz bizs.SwapBiz
}

func NewSwapHandler(sb bizs.SwapBiz) *SwapHandler {
	return &SwapHandler{swapBiz: sb}
}

// 定义请求体
type InitiateSwapReq struct {
	FromToken       string  `json:"from_token" binding:"required"`
	ToToken         string  `json:"to_token" binding:"required"`
	FromAmount      float64 `json:"from_amount" binding:"required,gt=0"`
	EstimatedAmount float64 `json:"estimated_amount" binding:"required,gt=0"`
	Slippage        float64 `json:"slippage" binding:"required,gte=0,lte=0.5"` // 最高容忍 50% 滑点
}

// InitiateExecute 供 App 调用
func (h *SwapHandler) InitiateExecute(ctx *gin.Context, req *InitiateSwapReq) (any, error) {
	userID := ctx.GetString("userID")
	
	swap, err := h.swapBiz.InitiateSwap(
		ctx.Request.Context(),
		userID,
		req.FromToken,
		req.ToToken,
		req.FromAmount,
		req.EstimatedAmount,
		req.Slippage,
	)
	
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"swap_id": swap.ID.String(),
		"status":  swap.Status, // PROCESSING
		"message": "Swap initiated, confirming on blockchain...",
	}, nil
}