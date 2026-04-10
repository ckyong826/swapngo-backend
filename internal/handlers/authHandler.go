package handlers

import (
	"swapngo-backend/internal/bizs"
	"swapngo-backend/pkg/requests/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	Register(ctx *gin.Context, req *auth.RegisterRequest) (any, error)
	Login(ctx *gin.Context, req *auth.LoginRequest) (any, error)
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