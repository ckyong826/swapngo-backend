package handlers

import (
	"swapngo-backend/internal/bizs"
	swap "swapngo-backend/pkg/requests/swap"
	txRes "swapngo-backend/pkg/responses/transaction"

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

func (h *swapHandler) InitiateExecute(ctx *gin.Context, req *swap.InitiateSwapReq) (any, error) {
	userID := ctx.GetString("user_id")

	slippage := req.Slippage
	if slippage == 0 {
		slippage = 0.01
	}

	s, err := h.swapBiz.InitiateSwap(
		ctx.Request.Context(),
		userID,
		req.FromToken,
		req.ToToken,
		req.Amount,
		req.EstimatedAmount,
		slippage,
		req.Pin,
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":     s.ID.String(),
		"status": "pending",
	}, nil
}

func (h *swapHandler) ViewSwap(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	s, err := h.swapBiz.ViewSwap(ctx.Request.Context(), userID, id)
	if err != nil {
		return nil, err
	}
	return txRes.ToSwapResponse(s), nil
}

func (h *swapHandler) ViewAllSwaps(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	swaps, err := h.swapBiz.ViewAllSwaps(ctx.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	return txRes.ToSwapResponses(swaps), nil
}