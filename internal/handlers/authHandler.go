package handlers

import (
	"swapngo-backend/internal/bizs"
	"swapngo-backend/pkg/requests/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	Register(ctx *gin.Context, req *auth.RegisterRequest) (any, error)
	Login(ctx *gin.Context, req *auth.LoginRequest) (any, error)
	VerifyPin(ctx *gin.Context, req *auth.VerifyPinRequest) (any, error)
}

type authHandler struct {
	biz bizs.AuthBiz
}

func NewAuthHandler(biz bizs.AuthBiz) AuthHandler {
	return &authHandler{biz: biz}
}

func (h *authHandler) Register(ctx *gin.Context, req *auth.RegisterRequest) (any, error) {
	return h.biz.Register(ctx, req)
}

func (h *authHandler) Login(ctx *gin.Context, req *auth.LoginRequest) (any, error) {
	return h.biz.Login(ctx, req)
}

func (h *authHandler) VerifyPin(ctx *gin.Context, req *auth.VerifyPinRequest) (any, error) {
	userID := ctx.GetString("user_id")
	if err := h.biz.VerifyPin(ctx.Request.Context(), userID, req.Pin); err != nil {
		return nil, err
	}
	return gin.H{"valid": true}, nil
}