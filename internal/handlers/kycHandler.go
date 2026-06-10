package handlers

import (
	"net/http"

	"swapngo-backend/internal/bizs"
	kycReq "swapngo-backend/pkg/requests/kyc"
	"swapngo-backend/pkg/responses"

	"github.com/gin-gonic/gin"
)

type KYCHandler interface {
	SubmitKYC(ctx *gin.Context, req *kycReq.SubmitKYCRequest) (any, error)
	GetKYCStatus(ctx *gin.Context, _ *struct{}) (any, error)
	// Admin handlers (raw gin — no generic wrapper needed)
	ListPendingKYC(ctx *gin.Context)
	ApproveKYC(ctx *gin.Context)
	RejectKYC(ctx *gin.Context)
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

func (h *kycHandler) ListPendingKYC(ctx *gin.Context) {
	data, err := h.kycBiz.ListPendingKYC(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.APIResponse{Success: false, Message: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, responses.APIResponse{Success: true, Message: "Pending KYC list fetched", Data: data})
}

func (h *kycHandler) ApproveKYC(ctx *gin.Context) {
	kycID := ctx.Param("id")
	data, err := h.kycBiz.ApproveKYC(ctx.Request.Context(), kycID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.APIResponse{Success: false, Message: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, responses.APIResponse{Success: true, Message: "KYC approved", Data: data})
}

func (h *kycHandler) RejectKYC(ctx *gin.Context) {
	kycID := ctx.Param("id")
	var req kycReq.RejectKYCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.APIResponse{Success: false, Message: "remarks is required"})
		return
	}
	data, err := h.kycBiz.RejectKYC(ctx.Request.Context(), kycID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.APIResponse{Success: false, Message: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, responses.APIResponse{Success: true, Message: "KYC rejected", Data: data})
}
