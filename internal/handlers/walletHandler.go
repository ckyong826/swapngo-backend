package handlers

import (
	"swapngo-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type WalletHandler interface {
	GetTotalBalanceByUserID(ctx *gin.Context, req *any) (any, error)
	GetMYRCBalanceByUserID(ctx *gin.Context, req *any) (any, error)
}

type walletHandler struct {
	walletService services.WalletService
}

func NewWalletHandler(walletService services.WalletService) WalletHandler {
	return &walletHandler{walletService: walletService}
}

func (h *walletHandler) GetTotalBalanceByUserID(ctx *gin.Context, req *any) (any, error) {
	userID := ctx.GetString("user_id")
	walletResponse, err := h.walletService.GetTotalBalanceByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}

func (h *walletHandler) GetMYRCBalanceByUserID(ctx *gin.Context, req *any) (any, error) {
	userID := ctx.GetString("user_id")
	balance, err := h.walletService.GetMYRCBalanceByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return balance, nil
}