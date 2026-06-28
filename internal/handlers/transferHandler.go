package handlers

import (
	"swapngo-backend/internal/bizs"
	transfer "swapngo-backend/pkg/requests/transfer"
	txRes "swapngo-backend/pkg/responses/transaction"

	"github.com/gin-gonic/gin"
)

type TransferHandler interface {
	ResolveRecipient(ctx *gin.Context, req *transfer.ResolveRecipientReq) (any, error)
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

func (h *transferHandler) ResolveRecipient(ctx *gin.Context, req *transfer.ResolveRecipientReq) (any, error) {
	userID := ctx.GetString("user_id")
	return h.transferBiz.ResolveRecipient(ctx.Request.Context(), userID, req.Q)
}

func (h *transferHandler) TransferMYRC(ctx *gin.Context, req *transfer.InitiateTransferReq) (any, error) {
	userID := ctx.GetString("user_id")
	t, err := h.transferBiz.InitiateTransfer(ctx.Request.Context(), userID, req.Recipient, req.Amount, req.Pin)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":     t.ID.String(),
		"status": "pending",
	}, nil
}

func (h *transferHandler) ViewTransfer(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	t, err := h.transferBiz.ViewTransfer(ctx.Request.Context(), userID, id)
	if err != nil {
		return nil, err
	}
	return txRes.ToTransferResponse(t), nil
}

func (h *transferHandler) ViewAllTransfers(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	transfers, err := h.transferBiz.ViewAllTransfers(ctx.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	return txRes.ToTransferResponses(transfers), nil
}
