package handlers

import (
	"swapngo-backend/internal/bizs"
	kycReq "swapngo-backend/pkg/requests/kyc"

	"github.com/gin-gonic/gin"
)

type KYCHandler interface {
	SubmitKYC(ctx *gin.Context, req *kycReq.SubmitKYCRequest) (any, error)
	GetKYCStatus(ctx *gin.Context, _ *struct{}) (any, error)
}

type kycHandler struct {
	kycBiz bizs.KYCBiz
}

func NewKYCHandler(kycBiz bizs.KYCBiz) KYCHandler {
	return &kycHandler{kycBiz: kycBiz}
}

func (h *kycHandler) SubmitKYC(ctx *gin.Context, req *kycReq.SubmitKYCRequest) (any, error) {
	userID := ctx.GetString("user_id")
	return h.kycBiz.SubmitKYC(ctx.Request.Context(), userID, req)
}

func (h *kycHandler) GetKYCStatus(ctx *gin.Context, _ *struct{}) (any, error) {
	userID := ctx.GetString("user_id")
	return h.kycBiz.GetKYCStatus(ctx.Request.Context(), userID)
}
