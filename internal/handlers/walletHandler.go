package handlers

import (
	"swapngo-backend/internal/services"
	"swapngo-backend/pkg/constants"

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
	userID := ctx.GetString(constants.UserID)
	walletResponse, err := h.walletService.GetTotalBalanceByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}

func (h *walletHandler) GetMYRCBalanceByUserID(ctx *gin.Context, req *any) (any, error) {
	userID := ctx.GetString(constants.UserID)
	balance, err := h.walletService.GetMYRCBalanceByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return balance, nil
}