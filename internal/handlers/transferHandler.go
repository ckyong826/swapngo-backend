package handlers

import (
	"swapngo-backend/internal/bizs"
	transfer "swapngo-backend/pkg/requests/transfer"

	"github.com/gin-gonic/gin"
)

type TransferHandler interface {
	TransferMYRC(ctx *gin.Context, req *transfer.InitiateTransferReq) (any, error)
}

type transferHandler struct {
	transferBiz bizs.TransferBiz
}

func NewTransferHandler(transferBiz bizs.TransferBiz) TransferHandler {
	return &transferHandler{transferBiz: transferBiz}
}

func (h *transferHandler) TransferMYRC(ctx *gin.Context, req *transfer.InitiateTransferReq) (any, error) {
	userID := ctx.GetString("user_id")
	walletResponse, err := h.transferBiz.InitiateTransfer(ctx.Request.Context(), userID, req.FromAddress, req.ToAddress, req.AmountMYRC)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}
