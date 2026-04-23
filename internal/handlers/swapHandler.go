package handlers

import (
	"swapngo-backend/internal/bizs"
	swap "swapngo-backend/pkg/requests/swap"

	"github.com/gin-gonic/gin"
)

type SwapHandler interface {
	InitiateExecute(ctx *gin.Context, req *swap.InitiateSwapReq) (any, error)
	ViewSwap(ctx *gin.Context, _ *struct{}) (any, error)
	ViewAllSwaps(ctx *gin.Context, _ *struct{}) (any, error)
}

type swapHandler struct {
	swapBiz bizs.SwapBiz
}

func NewSwapHandler(sb bizs.SwapBiz) SwapHandler {
	return &swapHandler{swapBiz: sb}
}

// 定义请求体

// InitiateExecute 供 App 调用
func (h *swapHandler) InitiateExecute(ctx *gin.Context, req *swap.InitiateSwapReq) (any, error) {
	userID := ctx.GetString("user_id")
	
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

func (h *swapHandler) ViewSwap(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	return h.swapBiz.ViewSwap(ctx.Request.Context(), userID, id)
}

func (h *swapHandler) ViewAllSwaps(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	return h.swapBiz.ViewAllSwaps(ctx.Request.Context(), userID)
}