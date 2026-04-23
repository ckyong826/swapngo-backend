package handlers

import (
	"swapngo-backend/internal/bizs"
	transfer "swapngo-backend/pkg/requests/transfer"

	"github.com/gin-gonic/gin"
)

type TransferHandler interface {
	TransferMYRC(ctx *gin.Context, req *transfer.InitiateTransferReq) (any, error)
	ViewTransfer(ctx *gin.Context, _ *struct{}) (any, error)
	ViewAllTransfers(ctx *gin.Context, _ *struct{}) (any, error)
}

type transferHandler struct {
	transferBiz bizs.TransferBiz
}

func NewTransferHandler(transferBiz bizs.TransferBiz) TransferHandler {
	return &transferHandler{transferBiz: transferBiz}
}

func (h *transferHandler) TransferMYRC(ctx *gin.Context, req *transfer.InitiateTransferReq) (any, error) {
	userID := ctx.GetString("user_id")
	walletResponse, err := h.transferBiz.InitiateTransfer(ctx.Request.Context(), userID, req.ReceiverUserID, req.Amount)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}

func (h *transferHandler) ViewTransfer(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	return h.transferBiz.ViewTransfer(ctx.Request.Context(), userID, id)
}

func (h *transferHandler) ViewAllTransfers(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	return h.transferBiz.ViewAllTransfers(ctx.Request.Context(), userID)
}
