package handlers

import (
	"swapngo-backend/internal/bizs"
	requests "swapngo-backend/pkg/requests/crypto"

	"github.com/gin-gonic/gin"
)

type CryptoHandler interface {
	DepositSUI(ctx *gin.Context, req *requests.DepositSUIReq) (any, error)
	SimulateDeposit(ctx *gin.Context, req *requests.SimulateDepositReq) (any, error)
	Withdraw(ctx *gin.Context, req *requests.WithdrawReq) (any, error)
	ViewAll(ctx *gin.Context, _ *struct{}) (any, error)
}

type cryptoHandler struct {
	cryptoBiz bizs.CryptoBiz
}

func NewCryptoHandler(cryptoBiz bizs.CryptoBiz) CryptoHandler {
	return &cryptoHandler{cryptoBiz: cryptoBiz}
}

func (h *cryptoHandler) DepositSUI(ctx *gin.Context, req *requests.DepositSUIReq) (any, error) {
	userID := ctx.GetString("user_id")
	return h.cryptoBiz.DepositSUI(ctx.Request.Context(), userID, req.TxDigest, req.Amount)
}

func (h *cryptoHandler) SimulateDeposit(ctx *gin.Context, req *requests.SimulateDepositReq) (any, error) {
	userID := ctx.GetString("user_id")
	return h.cryptoBiz.SimulateDeposit(ctx.Request.Context(), userID, req.Token, req.Amount)
}

func (h *cryptoHandler) Withdraw(ctx *gin.Context, req *requests.WithdrawReq) (any, error) {
	userID := ctx.GetString("user_id")
	return h.cryptoBiz.Withdraw(ctx.Request.Context(), userID, req.Token, req.Amount, req.ToAddress)
}

func (h *cryptoHandler) ViewAll(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	return h.cryptoBiz.ViewAll(ctx.Request.Context(), userID)
}
