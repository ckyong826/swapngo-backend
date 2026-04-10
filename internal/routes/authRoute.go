package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/utils"

	authReq "swapngo-backend/pkg/requests/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup, authHandler handlers.AuthHandler) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", utils.Handle[authReq.RegisterRequest]("Successfully registered", authHandler.Register))
		authGroup.POST("/login", utils.Handle[authReq.LoginRequest]("Successfully logged in", authHandler.Login))
	}
}
