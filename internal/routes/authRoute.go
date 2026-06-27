package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/middlewares"
	"swapngo-backend/pkg/utils"

	authReq "swapngo-backend/pkg/requests/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup, authHandler handlers.AuthHandler) {
	publicAuth := router.Group("/public/auth")
	{
		publicAuth.POST("/register", utils.Handle[authReq.RegisterRequest]("Successfully registered", authHandler.Register))
		publicAuth.POST("/login", utils.Handle[authReq.LoginRequest]("Successfully logged in", authHandler.Login))
	}

	privateAuth := router.Group("/private/auth")
	privateAuth.Use(middlewares.AuthMiddleware())
	{
		privateAuth.POST("/pin/verify", utils.Handle[authReq.VerifyPinRequest]("PIN verified", authHandler.VerifyPin))
	}
}
