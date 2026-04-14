package handlers

import (
	"swapngo-backend/internal/bizs"
	requests "swapngo-backend/pkg/requests/withdrawal"

	"github.com/gin-gonic/gin"
)

type WithdrawHandler interface {
	WithdrawMYRC(ctx *gin.Context, req *requests.InitiateWithdrawReq) (any, error)
}

type withdrawHandler struct {
	withdrawBiz bizs.WithdrawBiz
}

func NewWithdrawHandler(withdrawBiz bizs.WithdrawBiz) WithdrawHandler {
	return &withdrawHandler{withdrawBiz: withdrawBiz}
}

func (h *withdrawHandler) WithdrawMYRC(ctx *gin.Context, req *requests.InitiateWithdrawReq) (any, error) {
	userID := ctx.GetString("user_id")
	walletResponse, err := h.withdrawBiz.InitiateWithdrawal(ctx.Request.Context(), userID, req.AmountMYRC, req.BankName, req.BankAccountNo)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}
