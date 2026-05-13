package handlers

import (
	"errors"
	"swapngo-backend/internal/bizs"
	requests "swapngo-backend/pkg/requests/withdrawal"
	txRes "swapngo-backend/pkg/responses/transaction"

	"github.com/gin-gonic/gin"
)

type WithdrawHandler interface {
	WithdrawMYRC(ctx *gin.Context, req *requests.InitiateWithdrawReq) (any, error)
	ViewWithdraw(ctx *gin.Context, _ *struct{}) (any, error)
	ViewAllWithdraws(ctx *gin.Context, _ *struct{}) (any, error)
}

type withdrawHandler struct {
	withdrawBiz bizs.WithdrawBiz
}

func NewWithdrawHandler(withdrawBiz bizs.WithdrawBiz) WithdrawHandler {
	return &withdrawHandler{withdrawBiz: withdrawBiz}
}

func (h *withdrawHandler) WithdrawMYRC(ctx *gin.Context, req *requests.InitiateWithdrawReq) (any, error) {
	userID := ctx.GetString("user_id")

	var bankName, bankAccountNo string
	switch req.DestinationType {
	case "bank":
		if req.DestinationDetails.BankName == "" || req.DestinationDetails.BankAccount == "" {
			return nil, errors.New("bank_name and bank_account are required for bank withdrawal")
		}
		bankName = req.DestinationDetails.BankName
		bankAccountNo = req.DestinationDetails.BankAccount
	case "sui_wallet":
		if req.DestinationDetails.SUIAddress == "" {
			return nil, errors.New("sui_address is required for sui_wallet withdrawal")
		}
		bankName = "sui_wallet"
		bankAccountNo = req.DestinationDetails.SUIAddress
	}

	w, err := h.withdrawBiz.InitiateWithdrawal(ctx.Request.Context(), userID, req.Amount, bankName, bankAccountNo)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":     w.ID.String(),
		"status": "pending",
	}, nil
}

func (h *withdrawHandler) ViewWithdraw(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	w, err := h.withdrawBiz.ViewWithdraw(ctx.Request.Context(), userID, id)
	if err != nil {
		return nil, err
	}
	return txRes.ToWithdrawResponse(w), nil
}

func (h *withdrawHandler) ViewAllWithdraws(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	withdrawals, err := h.withdrawBiz.ViewAllWithdraws(ctx.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	return txRes.ToWithdrawResponses(withdrawals), nil
}
