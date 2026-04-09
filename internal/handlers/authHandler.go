package handlers

import (
	"swapngo-backend/internal/bizs"
	"swapngo-backend/pkg/requests"

	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	Register(ctx *gin.Context, req *requests.RegisterRequest) (any, error)
}

type authHandler struct {
	biz bizs.AuthBiz
}

func NewAuthHandler(biz bizs.AuthBiz) AuthHandler {
	return &authHandler{biz: biz}
}

func (h *authHandler) Register(ctx *gin.Context, req *requests.RegisterRequest) (any, error) {
	return h.biz.Register(ctx, req)
}