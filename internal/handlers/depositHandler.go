package handlers

import (
	"swapngo-backend/internal/bizs"
	requests "swapngo-backend/pkg/requests/deposit"

	"github.com/gin-gonic/gin"
)

type DepositHandler interface {
	DepositMYRC(ctx *gin.Context, req *requests.InitiateDepositReq) (any, error)
	HandleWebhook(ctx *gin.Context)
	ViewDeposit(ctx *gin.Context, _ *struct{}) (any, error)
	ViewAllDeposits(ctx *gin.Context, _ *struct{}) (any, error)
}

type depositHandler struct {
	depositBiz bizs.DepositBiz
}

func NewDepositHandler(depositBiz bizs.DepositBiz) DepositHandler {
	return &depositHandler{depositBiz: depositBiz}
}

func (h *depositHandler) DepositMYRC(ctx *gin.Context, req *requests.InitiateDepositReq) (any, error) {
	userID := ctx.GetString("user_id")
	walletResponse, err := h.depositBiz.InitiateDepositMYRC(ctx, req, userID)
	if err != nil {
		return nil, err
	}
	return walletResponse, nil
}

func (h *depositHandler) HandleWebhook(ctx *gin.Context) {
	var req requests.WebhookReq
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid payload"})
		return
	}

	isPaid := req.State == "paid"
	err := h.depositBiz.HandlePaymentWebhook(ctx.Request.Context(), req.ID, isPaid)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "failed to process webhook"})
		return
	}
	ctx.JSON(200, gin.H{"status": "ok"})
}

func (h *depositHandler) ViewDeposit(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	id := ctx.Param("id")
	return h.depositBiz.ViewDeposit(ctx.Request.Context(), userID, id)
}

func (h *depositHandler) ViewAllDeposits(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	return h.depositBiz.ViewAllDeposits(ctx.Request.Context(), userID)
}